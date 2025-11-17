package pr_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"pullreq/internal/pr"
	"pullreq/internal/team"
	"pullreq/internal/user"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = sql.Open("postgres", "host=localhost port=5429 user=postgres password=123 dbname=tr sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer testDB.Close()

	os.Exit(m.Run())
}

func cleanDB(db *sql.DB) error {
	_, err := db.Exec(`TRUNCATE TABLE userspr, pr, users, teams RESTART IDENTITY CASCADE;`)
	return err
}

type mockTeamRepo struct{}

func (m *mockTeamRepo) GetTeamByUserID(ctx context.Context, userID string) (string, error) {
	return "team1", nil
}

func (m *mockTeamRepo) GetTeamMember(ctx context.Context, teamID string) ([]*user.User, error) {
	return []*user.User{
		{Id: "user1"},
		{Id: "user2"},
		{Id: "user3"},
	}, nil
}

func setupDB(ctx context.Context, db *sql.DB) error {
	return cleanDB(db)
}

func setupTeam(ctx context.Context, db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO teams (team_name) VALUES ($1)`, "Awesome Team")
	if err != nil {
		return err
	}

	users := []struct {
		id   string
		name string
	}{
		{"user1", "Alice"},
		{"user2", "Bob"},
		{"user3", "Charlie"},
		{"u", "Author"},
	}

	for _, u := range users {
		if _, err := db.Exec(`INSERT INTO users (id, username, team_id, is_active) VALUES ($1, $2, $3, $4)`, u.id, u.name, 1, true); err != nil {
			return err
		}
	}
	return nil
}

func TestPullRequestRepo_Create_GetMerge(t *testing.T) {
	ctx := context.Background()
	if err := setupDB(ctx, testDB); err != nil {
		t.Fatalf(err.Error())
	}

	UR := &user.UserRepo{DB: testDB}
	TR := &team.TeamRepo{DB: testDB, UR: UR}
	repo := &pr.PullRequestRepo{DB: testDB, TR: TR, UR: UR}

	if err := setupTeam(ctx, testDB); err != nil {
		t.Fatalf("failed to setup team: %v", err)
	}

	prReq := pr.CreatePullRequestRequest{ID: "pr1", PullRequestName: "My First PR", AuthorID: "u"}
	createdPR, err := repo.Create(ctx, prReq)
	if err != nil {
		t.Fatalf("failed to create PR: %v", err)
	}
	if len(createdPR.AssignedReviewers) == 0 {
		t.Fatalf("expected assigned reviewers, got none")
	}

	gotPR, err := repo.GetPr(ctx, "pr1")
	if err != nil {
		t.Fatalf("failed to get PR: %v", err)
	}
	if gotPR.ID != prReq.ID {
		t.Fatalf("expected PR ID %s, got %s", prReq.ID, gotPR.ID)
	}

	mergedPR, err := repo.Merged(ctx, "pr1")
	if err != nil {
		t.Fatalf("failed to merge PR: %v", err)
	}
	if mergedPR.Status != "MERGED" {
		t.Fatalf("expected PR status MERGED, got %s", mergedPR.Status)
	}
}

func insertPR(t *testing.T, prID string, authorID string, reviewers []string) {
	_, err := testDB.Exec(`INSERT INTO pr (id, pr_name, author_id, pr_status) VALUES ($1, $2, $3, $4)`, prID, "My PR", authorID, "OPEN")
	if err != nil {
		t.Fatalf("failed to insert PR: %v", err)
	}

	for _, r := range reviewers {
		_, err := testDB.Exec(`INSERT INTO userspr (user_id, request_id) VALUES ($1, $2)`, r, prID)
		if err != nil {
			t.Fatalf("failed to insert reviewer %s: %v", r, err)
		}
	}
}

func TestAssignedReviewerIntegration(t *testing.T) {
	ctx := context.Background()
	if err := setupDB(ctx, testDB); err != nil {
		t.Fatalf(err.Error())
	}
	if err := setupTeam(ctx, testDB); err != nil {
		t.Fatalf(err.Error())
	}

	UR := &user.UserRepo{DB: testDB}
	TR := &team.TeamRepo{DB: testDB, UR: UR}
	repo := &pr.PullRequestRepo{DB: testDB, TR: TR, UR: UR}

	insertPR(t, "pr1", "user1", []string{"user1"})

	updatedPR, newReviewer, err := repo.AssignedReviewer(ctx, "pr1", "user1")
	if err != nil {
		t.Fatalf("failed to assign reviewer: %v", err)
	}

	if newReviewer == "user1" {
		t.Fatalf("expected new reviewer to be different from user1")
	}

	found := false
	for _, r := range updatedPR.AssignedReviewers {
		if r == newReviewer {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("new reviewer %s not in updated PR reviewers", newReviewer)
	}
}
