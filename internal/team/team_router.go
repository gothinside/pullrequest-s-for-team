package team

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pullreq/internal/errs"
	jsonutils "pullreq/internal/json_utils"
)

type TeamInput struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type TeamRouter struct {
	TR TeamRepoInterface
}

type UserResp struct {
	Id        string `json:"id"`
	Username  string `json:"username"`
	Teamname  string `json:"team_name"`
	Is_active bool   `json:"is_active"`
}

type TeamRes struct {
	ID       int         `json:"-"`
	TeamName string      `json:"team_name"`
	Members  []*UserResp `json:"members"`
}

type DeactivateTeamRequest struct {
	TeamName string `json:"team_name"`
}

func (tr *TeamRouter) DeactivateTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
	ctx := r.Context()

	var req DeactivateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.TeamName == "" {
		http.Error(w, "team_id is required", http.StatusBadRequest)
		return
	}

	err := tr.TR.Deactivation(ctx, req.TeamName)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	team, err := tr.TR.GetTeamWithMembers(ctx, req.TeamName)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	jsonutils.JsonResponse(w, map[string]interface{}{"team": team}, http.StatusOK)
}

func (tr *TeamRouter) HandleAddTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var newTeam TeamInput
	if err := json.Unmarshal(body, &newTeam); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	_, err = tr.TR.AddTeam(r.Context(), newTeam.TeamName, newTeam.Members)
	if err != nil {
		// Team already exists
		errs.JsonCodeResp(w, errs.CodeTeamExists, fmt.Sprintf("Team '%s' already exists", newTeam.TeamName), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"team": newTeam,
	}
	jsonutils.JsonResponse(w, response, http.StatusCreated)
}

func (tr *TeamRouter) GetTeamWithMembersHandler(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "Отсутствует обязательный параметр запроса: team_name", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	Team, err := tr.TR.GetTeamWithMembers(ctx, teamName)

	if err != nil {
		if err == sql.ErrNoRows {
			errs.JsonCodeResp(w, errs.CodeNotFound, "Team not found", http.StatusNotFound)
			return
		} else {
			http.Error(w, "Internal error", http.StatusBadRequest)
			return
		}
	}
	var resTeam TeamRes
	resTeam.TeamName = Team.TeamName
	for _, x := range Team.Members {
		resTeam.Members = append(resTeam.Members, &UserResp{Id: x.Id, Username: x.Username, Teamname: resTeam.TeamName, Is_active: x.IsActive})
	}
	response := map[string]interface{}{
		"team": resTeam,
	}
	jsonutils.JsonResponse(w, response, http.StatusOK)
}
