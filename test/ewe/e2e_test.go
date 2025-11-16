package e2e

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strconv"
// 	"testing"
// 	"time"
// )

// func TestE2E(t *testing.T) {
// 	baseURL := "http://localhost:8080"

// 	// Generate dynamic IDs
// 	ts := strconv.FormatInt(time.Now().UnixNano(), 10)
// 	teamName := "team-" + ts
// 	user1 := "u1-" + ts
// 	user2 := "u2-" + ts
// 	prID := "pr-" + ts

// 	client := &http.Client{}

// 	// ----------------------------
// 	// 1. Create team with users
// 	// ----------------------------
// 	teamBody := map[string]interface{}{
// 		"team_name": teamName,
// 		"members": []map[string]interface{}{
// 			{"user_id": user1, "username": "Alice", "is_active": true},
// 			{"user_id": user2, "username": "Bob", "is_active": true},
// 		},
// 	}
// 	b, _ := json.Marshal(teamBody)
// 	resp, err := client.Post(baseURL+"/team/add", "application/json", bytes.NewReader(b))
// 	if err != nil {
// 		t.Fatalf("failed to create team: %v", err)
// 	}
// 	if resp.StatusCode != http.StatusCreated {
// 		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
// 	}
// 	resp.Body.Close()

// 	// ----------------------------
// 	// 2. Create a Pull Request
// 	// ----------------------------
// 	prBody := map[string]interface{}{
// 		"pull_request_id":   prID,
// 		"pull_request_name": "Add search feature",
// 		"author_id":         user1,
// 	}
// 	b, _ = json.Marshal(prBody)
// 	resp, err = client.Post(baseURL+"/pullRequest/create", "application/json", bytes.NewReader(b))
// 	if err != nil {
// 		t.Fatalf("failed to create PR: %v", err)
// 	}
// 	if resp.StatusCode != http.StatusCreated {
// 		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
// 	}
// 	var prResp struct {
// 		PR struct {
// 			PullRequestID     string   `json:"pull_request_id"`
// 			PullRequestName   string   `json:"pull_request_name"`
// 			AuthorID          string   `json:"author_id"`
// 			Status            string   `json:"status"`
// 			AssignedReviewers []string `json:"assigned_reviewers"`
// 		} `json:"pr"`
// 	}
// 	bodyBytes, _ := io.ReadAll(resp.Body)
// 	json.Unmarshal(bodyBytes, &prResp)
// 	resp.Body.Close()
// 	if prResp.PR.Status != "OPEN" || len(prResp.PR.AssignedReviewers) == 0 {
// 		t.Fatalf("PR creation failed: %+v", prResp.PR)
// 	}
// 	oldReviewer := prResp.PR.AssignedReviewers[0]

// 	// ----------------------------
// 	// 3. Reassign a reviewer
// 	// ----------------------------
// 	reassignBody := map[string]interface{}{
// 		"pull_request_id": prID,
// 		"old_user_id":     oldReviewer,
// 	}
// 	b, _ = json.Marshal(reassignBody)
// 	resp, err = client.Post(baseURL+"/pullRequest/reassign", "application/json", bytes.NewReader(b))
// 	if err != nil {
// 		t.Fatalf("failed to reassign reviewer: %v", err)
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected 200 OK for reassign, got %d", resp.StatusCode)
// 	}
// 	var reassignResp struct {
// 		PR         interface{} `json:"pr"`
// 		ReplacedBy string      `json:"replaced_by"`
// 	}
// 	bodyBytes, _ = io.ReadAll(resp.Body)
// 	json.Unmarshal(bodyBytes, &reassignResp)
// 	resp.Body.Close()

// 	if reassignResp.ReplacedBy == oldReviewer {
// 		t.Fatalf("expected new reviewer different from old reviewer")
// 	}

// 	// ----------------------------
// 	// 4. Merge the PR
// 	// ----------------------------
// 	mergeBody := map[string]interface{}{
// 		"pull_request_id": prID,
// 	}
// 	b, _ = json.Marshal(mergeBody)
// 	resp, err = client.Post(baseURL+"/pullRequest/merge", "application/json", bytes.NewReader(b))
// 	if err != nil {
// 		t.Fatalf("failed to merge PR: %v", err)
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected 200 OK for merge, got %d", resp.StatusCode)
// 	}

// 	// ----------------------------
// 	// 5. Check user assigned PRs
// 	// ----------------------------
// 	resp, err = client.Get(fmt.Sprintf("%s/users/getReview?user_id=%s", baseURL, user2))
// 	if err != nil {
// 		t.Fatalf("failed to get user reviews: %v", err)
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected 200 OK for getReview, got %d", resp.StatusCode)
// 	}
// 	resp.Body.Close()

// 	// ----------------------------
// 	// 6. Set a user inactive
// 	// ----------------------------
// 	setInactiveBody := map[string]interface{}{
// 		"user_id":   user2,
// 		"is_active": false,
// 	}
// 	b, _ = json.Marshal(setInactiveBody)
// 	resp, err = client.Post(baseURL+"/users/setIsActive", "application/json", bytes.NewReader(b))
// 	if err != nil {
// 		t.Fatalf("failed to set user inactive: %v", err)
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected 200 OK for setIsActive, got %d", resp.StatusCode)
// 	}
// 	resp.Body.Close()
// }
