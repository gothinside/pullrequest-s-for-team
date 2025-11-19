package pr

import (
	"encoding/json"
	"errors"
	"net/http"
	"pullreq/internal/errs"
	jsonutils "pullreq/internal/json_utils"
)

// PullRequest represents a pull request object
type PullRequest struct {
	ID                string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
}

// CreatePullRequestRequest represents the request payload
type CreatePullRequestRequest struct {
	ID              string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type PrRouter struct {
	PR PullRequestRepoInterface
}

type MergeRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type ReassignRequest struct {
	PullRequestID     string `json:"pull_request_id"`
	CurrentReviewerID string `json:"old_user_id"` // ID ревьювера для замены
}

func (pr *PrRouter) AssignedReviewer(w http.ResponseWriter, r *http.Request) {
	var reqBody ReassignRequest

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	updatedPR, replacedByID, err := pr.PR.AssignedReviewer(r.Context(), reqBody.PullRequestID, reqBody.CurrentReviewerID)
	if err != nil {
		if errors.Is(err, errs.NotFountError) {
			errs.JsonCodeResp(w, errs.CodeNotFound, "PR or user not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, errs.PRMergedError) {
			errs.JsonCodeResp(w, "PR_MERGED", "cannot reassign on merged PR", http.StatusConflict)
			return
		}
		if errors.Is(err, errs.NotAssignedError) {
			errs.JsonCodeResp(w, "NOT_ASSIGNED", "reviewer is not assigned to this PR", http.StatusConflict)
			return
		}
		if errors.Is(err, errs.NoCandidateError) {
			errs.JsonCodeResp(w, "NO_CANDIDATE", "no active replacement candidate in team", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"pr":          updatedPR,
		"replaced_by": replacedByID,
	}

	jsonutils.JsonResponse(w, resp, http.StatusOK)
}

func (pr *PrRouter) Merge(w http.ResponseWriter, r *http.Request) {
	var id MergeRequest
	if err := json.NewDecoder(r.Body).Decode(&id); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	res, err := pr.PR.Merged(r.Context(), id.PullRequestID)
	if err != nil {
		if errors.Is(err, errs.NotFountError) {
			errs.JsonCodeResp(w, errs.CodeNotFound, "PR not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal", http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{"pr": res}
	jsonutils.JsonResponse(w, resp, http.StatusOK)
}

func (pr *PrRouter) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req CreatePullRequestRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	res, err := pr.PR.Create(r.Context(), req)
	if err != nil {
		if errors.Is(err, errs.NotFountError) {
			errs.JsonCodeResp(w, errs.CodeNotFound, "Author/Team not exist", http.StatusNotFound)
			return
		}
		if errors.Is(err, errs.ExistError) {
			errs.JsonCodeResp(w, errs.CodePRExists, "Pre already exist", 409)
			return
		}
		// if errors.Is(err, errs.NoCandidateError) {
		// 	errs.JsonCodeResp(w, errs.CodePRExists, "Team have no active users", 409)
		// 	return
		// }
		http.Error(w, "Internal", 500)
		return
	}
	resp := map[string]interface{}{"pr": res}
	jsonutils.JsonResponse(w, resp, http.StatusCreated)
}
