package user_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"pullreq/internal/errs"
	"pullreq/internal/user"
	routermocks "pullreq/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetUserReviewsHandler(t *testing.T) {
	mockUR := routermocks.NewUserRepoInterface(t)
	router := &user.UserRouter{UR: mockUR}

	t.Run("success", func(t *testing.T) {
		userID := "u1"
		mockUR.On("GetUsersPrShort", mock.Anything, userID).Return([]user.PullRequestShort{
			{PullRequestID: "pr1", PullRequestName: "feat", AuthorID: "u1", Status: "OPEN"},
		}, nil)

		req := httptest.NewRequest(http.MethodGet, "/?user_id="+userID, nil)
		w := httptest.NewRecorder()

		router.GetUserReviewsHandler(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, userID, resp["user_id"])
		require.Len(t, resp["pull_requests"].([]interface{}), 1)
	})

	t.Run("missing_user_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		router.GetUserReviewsHandler(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("internal_error", func(t *testing.T) {
		userID := "u2"
		mockUR.On("GetUsersPrShort", mock.Anything, userID).Return(nil, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/?user_id="+userID, nil)
		w := httptest.NewRecorder()

		router.GetUserReviewsHandler(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRouterSetActiviry(t *testing.T) {
	mockUR := routermocks.NewUserRepoInterface(t)
	router := &user.UserRouter{UR: mockUR}

	t.Run("success", func(t *testing.T) {
		input := user.UserShortInput{UserID: "u1", IsActive: true}
		updatedUser := &user.User{Id: "u1", Username: "Alice", IsActive: true}
		mockUR.On("UpdateUserActivity", mock.Anything, input.UserID, input.IsActive).Return(updatedUser, nil)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		router.RouterSetActiviry(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "u1", resp["user"].(map[string]interface{})["Id"])
	})

	t.Run("user_not_found", func(t *testing.T) {
		input := user.UserShortInput{UserID: "missing", IsActive: true}
		mockUR.On("UpdateUserActivity", mock.Anything, input.UserID, input.IsActive).Return(nil, sql.ErrNoRows)

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		router.RouterSetActiviry(w, req)
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{invalid")))
		w := httptest.NewRecorder()

		router.RouterSetActiviry(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("internal_error", func(t *testing.T) {
		input := user.UserShortInput{UserID: "u2", IsActive: false}
		mockUR.On("UpdateUserActivity", mock.Anything, input.UserID, input.IsActive).Return(nil, errors.New("db error"))

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		router.RouterSetActiviry(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetStat(t *testing.T) {
	mockUR := routermocks.NewUserRepoInterface(t)
	router := &user.UserRouter{UR: mockUR}

	t.Run("success", func(t *testing.T) {
		userID := "u1"
		mockUR.On("GetStatAboutUser", mock.Anything, userID).Return(5, nil)

		req := httptest.NewRequest(http.MethodGet, "/?user_id="+userID, nil)
		w := httptest.NewRecorder()

		router.GetStat(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		require.Equal(t, float64(5), resp[userID]) // JSON numbers → float64
	})

	t.Run("missing_user_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		router.GetStat(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("user_not_found", func(t *testing.T) {
		userID := "missing"
		mockUR.
			On("GetStatAboutUser", mock.Anything, userID).
			Return(0, errs.NotFountError)

		req := httptest.NewRequest(http.MethodGet, "/?user_id="+userID, nil)
		w := httptest.NewRecorder()

		router.GetStat(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("internal_error", func(t *testing.T) {
		userID := "u2"
		mockUR.
			On("GetStatAboutUser", mock.Anything, userID).
			Return(0, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/?user_id="+userID, nil)
		w := httptest.NewRecorder()

		router.GetStat(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
