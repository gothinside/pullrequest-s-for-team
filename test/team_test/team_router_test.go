package team_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"pullreq/internal/team"
	"pullreq/internal/user"
	routermocks "pullreq/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleAddTeam(t *testing.T) {
	mockTR := routermocks.NewTeamRepoInterface(t)
	router := &team.TeamRouter{TR: mockTR}

	t.Run("success", func(t *testing.T) {
		input := team.TeamInput{
			TeamName: "TeamA",
			Members: []team.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}
		mockTR.On("AddTeam", mock.Anything, input.TeamName, input.Members).Return(&team.Team{
			TeamName: input.TeamName,
		}, nil)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		router.HandleAddTeam(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "TeamA", resp["team"].(map[string]interface{})["team_name"])
	})

	t.Run("team_exists", func(t *testing.T) {
		input := team.TeamInput{
			TeamName: "TeamB",
			Members:  []team.TeamMember{},
		}
		mockTR.On("AddTeam", mock.Anything, input.TeamName, input.Members).Return(nil, errors.New("exists"))

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		router.HandleAddTeam(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{invalid")))
		w := httptest.NewRecorder()

		router.HandleAddTeam(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		router.HandleAddTeam(w, req)
		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestDeactivateTeamHandler(t *testing.T) {
	mockTR := routermocks.NewTeamRepoInterface(t)
	router := &team.TeamRouter{TR: mockTR}

	t.Run("success", func(t *testing.T) {
		input := team.DeactivateTeamRequest{
			TeamID: 10,
		}

		// Ожидаемый вызов репозитория
		mockTR.On("Deactivation", mock.Anything, input.TeamID).
			Return(nil)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		router.DeactivateTeam(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, float64(10), resp["team_id"])
		require.Equal(t, "ok", resp["status"])
	})

	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad_json"))
		w := httptest.NewRecorder()

		router.DeactivateTeam(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing_team_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"team_id": 0}`))
		w := httptest.NewRecorder()

		router.DeactivateTeam(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		router.DeactivateTeam(w, req)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("repo_error", func(t *testing.T) {
		input := team.DeactivateTeamRequest{TeamID: 99}

		mockTR.On("Deactivation", mock.Anything, 99).
			Return(errors.New("db error"))

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		router.DeactivateTeam(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetTeamWithMembersHandler(t *testing.T) {
	mockTR := routermocks.NewTeamRepoInterface(t)
	router := &team.TeamRouter{TR: mockTR}

	t.Run("success", func(t *testing.T) {
		teamName := "TeamA"
		mockTR.On("GetTeamWithMembers", mock.Anything, teamName).Return(&team.Team{
			TeamName: teamName,
			Members: []*user.User{
				{Id: "u1", Username: "Alice", TeamID: 1, IsActive: true},
			},
		}, nil)

		req := httptest.NewRequest(http.MethodGet, "/?team_name="+teamName, nil)
		w := httptest.NewRecorder()

		router.GetTeamWithMembersHandler(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "TeamA", resp["team"].(map[string]interface{})["team_name"])
	})

	t.Run("team_not_found", func(t *testing.T) {
		teamName := "Unknown"
		mockTR.On("GetTeamWithMembers", mock.Anything, teamName).Return(nil, sql.ErrNoRows)

		req := httptest.NewRequest(http.MethodGet, "/?team_name="+teamName, nil)
		w := httptest.NewRecorder()

		router.GetTeamWithMembersHandler(w, req)
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("internal_error", func(t *testing.T) {
		teamName := "TeamError"
		mockTR.On("GetTeamWithMembers", mock.Anything, teamName).Return(nil, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/?team_name="+teamName, nil)
		w := httptest.NewRecorder()

		router.GetTeamWithMembersHandler(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing_query_param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		router.GetTeamWithMembersHandler(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
