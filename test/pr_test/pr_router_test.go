package pr_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pullreq/internal/errs"
	"pullreq/internal/pr"
	routermocks "pullreq/mocks"
)

func TestCreatePullRequest(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)

	// mock Create function
	reqBody := pr.CreatePullRequestRequest{
		ID:              "pr-1001",
		PullRequestName: "Add search",
		AuthorID:        "u1",
	}

	mockRepo.On("Create", context.Background(), reqBody).Return(&pr.PullRequest{
		ID:                "pr-1001",
		PullRequestName:   "Add search",
		AuthorID:          "u1",
		Status:            "OPEN",
		AssignedReviewers: []string{"u2", "u3"},
	}, nil)

	router := &pr.PrRouter{PR: mockRepo}

	bodyJSON := `{"pull_request_id":"pr-1001","pull_request_name":"Add search","author_id":"u1"}`
	req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer([]byte(bodyJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.CreatePullRequest(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(body), `"pull_request_id":"pr-1001"`) {
		t.Fatalf("unexpected response body: %s", string(body))
	}

	mockRepo.AssertExpectations(t)
}

func TestMergePullRequest(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)

	mockRepo.On("Merged", context.Background(), "pr-1001").Return(&pr.PullRequest{
		ID:                "pr-1001",
		PullRequestName:   "Add search",
		AuthorID:          "u1",
		Status:            "MERGED",
		AssignedReviewers: []string{"u2", "u3"},
	}, nil)

	router := &pr.PrRouter{PR: mockRepo}

	bodyJSON := `{"pull_request_id":"pr-1001"}`
	req := httptest.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer([]byte(bodyJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.Merge(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(body), `"status":"MERGED"`) {
		t.Fatalf("unexpected response body: %s", string(body))
	}

	mockRepo.AssertExpectations(t)
}

func TestReass_1(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)

	mockRepo.On("AssignedReviewer", context.Background(), "pr-1001", "u2").Return(&pr.PullRequest{
		ID:                "pr-1001",
		PullRequestName:   "Add search",
		AuthorID:          "u1",
		Status:            "OPEN",
		AssignedReviewers: []string{"u3", "u5"},
	}, "u5", nil)

	router := &pr.PrRouter{PR: mockRepo}

	bodyJSON := `{"pull_request_id":"pr-1001","old_user_id":"u2"}`
	req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer([]byte(bodyJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.AssignedReviewer(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(body), `"replaced_by":"u5"`) {
		t.Fatalf("unexpected response body: %s", string(body))
	}

	mockRepo.AssertExpectations(t)
}

// Test invalid JSON payload for CreatePullRequest
func TestCreatePullRequest_InvalidJSON(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)
	router := &pr.PrRouter{PR: mockRepo}

	req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer([]byte("{invalid-json}")))
	w := httptest.NewRecorder()

	router.CreatePullRequest(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	if string(body) != "Invalid request payload\n" {
		t.Fatalf("unexpected response body: %s", string(body))
	}
}

// Test CreatePullRequest when PR already exists
func TestCreatePullRequest_ExistError(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)
	router := &pr.PrRouter{PR: mockRepo}

	reqBody := pr.CreatePullRequestRequest{
		ID:              "pr-1001",
		PullRequestName: "Add search",
		AuthorID:        "u1",
	}

	mockRepo.On("Create", context.Background(), reqBody).Return(nil, errs.ExistError)

	bodyJSON := `{"pull_request_id":"pr-1001","pull_request_name":"Add search","author_id":"u1"}`
	req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer([]byte(bodyJSON)))
	w := httptest.NewRecorder()

	router.CreatePullRequest(w, req)

	resp := w.Result()
	if resp.StatusCode != 409 {
		t.Fatalf("expected status 409, got %d", resp.StatusCode)
	}
}

// Test Merge with invalid JSON
func TestMerge_InvalidJSON(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)
	router := &pr.PrRouter{PR: mockRepo}

	req := httptest.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer([]byte("{bad-json}")))
	w := httptest.NewRecorder()

	router.Merge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

// Test Merge when PR not found
func TestMerge_PRNotFound(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)
	router := &pr.PrRouter{PR: mockRepo}

	mockRepo.On("Merged", context.Background(), "pr-1001").Return(nil, errs.NotFountError)

	bodyJSON := `{"pull_request_id":"pr-1001"}`
	req := httptest.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer([]byte(bodyJSON)))
	w := httptest.NewRecorder()

	router.Merge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}
}

// Test AssignedReviewer with domain errors
func TestAssignedReviewer_DomainErrors(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)
	router := &pr.PrRouter{PR: mockRepo}

	tests := []struct {
		err          error
		expectedCode int
	}{
		{errs.PRMergedError, http.StatusConflict},
		{errs.NotAssignedError, http.StatusConflict},
		{errs.NoCandidateError, http.StatusConflict},
	}

	for _, tt := range tests {
		mockRepo.On("AssignedReviewer", context.Background(), "pr-1001", "u2").
			Return(nil, "", tt.err)

		bodyJSON := `{"pull_request_id":"pr-1001","old_user_id":"u2"}`
		req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer([]byte(bodyJSON)))
		w := httptest.NewRecorder()

		router.AssignedReviewer(w, req)
		resp := w.Result()

		if resp.StatusCode != tt.expectedCode {
			t.Fatalf("expected status %d, got %d for error %v", tt.expectedCode, resp.StatusCode, tt.err)
		}
	}
}

// Test AssignedReviewer with invalid JSON
func TestAssignedReviewer_InvalidJSON(t *testing.T) {
	mockRepo := routermocks.NewPullRequestRepoInterface(t)
	router := &pr.PrRouter{PR: mockRepo}

	req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer([]byte("{bad-json}")))
	w := httptest.NewRecorder()

	router.AssignedReviewer(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}
