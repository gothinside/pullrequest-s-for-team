package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"pullreq/internal/errs"
	jsonutils "pullreq/internal/json_utils"
)

type UserRouter struct {
	UR UserRepoInterface
}

type UserShortInput struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

func (ur *UserRouter) GetStat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id query parameter", http.StatusBadRequest)
		return
	}
	count, err := ur.UR.GetStatAboutUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, errs.NotFountError) {
			errs.JsonCodeResp(w, errs.CodeNotFound, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
	result := map[string]interface{}{
		userID: count,
	}
	jsonutils.JsonResponse(w, result, http.StatusOK)
}

func (ur *UserRouter) GetUserReviewsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id query parameter", http.StatusBadRequest)
		return
	}

	prs, err := ur.UR.GetUsersPrShort(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	}
	jsonutils.JsonResponse(w, response, http.StatusOK)

}

func (ur *UserRouter) RouterSetActiviry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	var newTeam UserShortInput

	if err = json.Unmarshal(body, &newTeam); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	user, err := ur.UR.UpdateUserActivity(context.Background(), newTeam.UserID, newTeam.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		errs.JsonCodeResp(w, errs.CodeNotFound, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	jsonutils.JsonResponse(w, map[string]interface{}{"user": user}, http.StatusOK)
}
