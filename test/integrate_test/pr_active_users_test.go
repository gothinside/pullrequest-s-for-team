package integratetest

// import (
// 	"context"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"math/rand"
// 	"testing"

// 	"pullreq/internal/pr"
// 	"pullreq/internal/team"
// 	"pullreq/internal/user"

// 	_ "github.com/lib/pq"
// )

// var testDB *sql.DB

// func setupDB(t *testing.T) {
// 	var err error
// 	testDB, err = sql.Open("postgres", "host=localhost port=5429 user=postgres password=123 dbname=tr sslmode=disable")
// 	if err != nil {
// 		t.Fatalf("failed to connect db: %v", err)
// 	}

// 	_, err = testDB.Exec(`TRUNCATE TABLE userspr, pr, users, teams RESTART IDENTITY CASCADE`)
// 	if err != nil {
// 		t.Fatalf("failed to truncate tables: %v", err)
// 	}

// 	_, err = testDB.Exec(`INSERT INTO teams (id, team_name) VALUES ($1, $2)`, 1, "Team A")
// 	if err != nil {
// 		t.Fatalf("failed to insert team: %v", err)
// 	}

// 	users := []struct {
// 		id       string
// 		username string
// 		active   bool
// 	}{
// 		{"author1", "Author1", true},
// 		{"uu1", "Alice1", true},
// 		{"uu2", "Bob1", true},
// 		{"uu3", "Charlie1", false}, // inactive user
// 	}
// 	for _, u := range users {
// 		_, err := testDB.Exec(`INSERT INTO users (id, username, team_id, is_active) VALUES ($1,$2,$3,$4)`, u.id, u.username, 1, u.active)
// 		if err != nil {
// 			t.Fatalf("failed to insert user %s: %v", u.id, err)
// 		}
// 	}
// }

// func TestCreatePR_OnlyActiveUsersAssigned(t *testing.T) {
// 	setupDB(t)
// 	ctx := context.Background()

// 	UR := &user.UserRepo{DB: testDB}
// 	TR := &team.TeamRepo{DB: testDB, UR: UR}
// 	RR := &pr.PullRequestRepo{DB: testDB, UR: UR, TR: TR}

// 	req := pr.CreatePullRequestRequest{
// 		ID:              "pr2",
// 		PullRequestName: "ActiveOnlyPR",
// 		AuthorID:        "author1",
// 	}

// 	rand.Seed(42) // deterministic test
// 	prRes, err := RR.Create(ctx, req)
// 	if err != nil {
// 		t.Fatalf("failed to create PR: %v", err)
// 	}

// 	// Verify inactive users are not assigned
// 	for _, reviewer := range prRes.AssignedReviewers {
// 		if reviewer == "uu3" {
// 			t.Fatalf("inactive user was assigned as reviewer")
// 		}
// 	}

// 	// Should have at least 1 reviewer
// 	if len(prRes.AssignedReviewers) == 0 {
// 		t.Fatalf("expected at least 1 reviewer, got none")
// 	}
// }

// func TestReassignReviewer_OnlyToActiveUsers(t *testing.T) {
// 	setupDB(t)
// 	ctx := context.Background()

// 	UR := &user.UserRepo{DB: testDB}
// 	TR := &team.TeamRepo{DB: testDB, UR: UR}
// 	RR := &pr.PullRequestRepo{DB: testDB, UR: UR, TR: TR}

// 	// Create PR
// 	req := pr.CreatePullRequestRequest{
// 		ID:              "pr3",
// 		PullRequestName: "ReassignTestPR",
// 		AuthorID:        "author",
// 	}
// 	rand.Seed(42)
// 	prRes, err := RR.Create(ctx, req)
// 	if err != nil {
// 		t.Fatalf("failed to create PR: %v", err)
// 	}

// 	oldReviewer := prRes.AssignedReviewers[0]
// 	updatedPR, newReviewer, err := RR.AssignedReviewer(ctx, "pr2", oldReviewer)
// 	if err != nil {
// 		t.Fatalf("failed to reassign reviewer: %v", err)
// 	}

// 	// Should not reassign to inactive user
// 	if newReviewer == "uu3" {
// 		t.Fatalf("inactive user was assigned as new reviewer")
// 	}

// 	// The old reviewer should be replaced
// 	foundOld := false
// 	for _, r := range updatedPR.AssignedReviewers {
// 		if r == oldReviewer {
// 			foundOld = true
// 			break
// 		}
// 	}
// 	if foundOld {
// 		t.Fatalf("old reviewer was not replaced")
// 	}

// 	// JSON check (optional)
// 	data, _ := json.Marshal(updatedPR)
// 	fmt.Printf("Updated PR after reassignment: %s\n", string(data))
// }
