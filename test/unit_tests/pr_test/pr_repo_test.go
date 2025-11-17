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
	schema := `
DROP TABLE IF EXISTS userspr CASCADE;
DROP TABLE IF EXISTS usershistory CASCADE;
DROP TABLE IF EXISTS pr CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS teams CASCADE;

CREATE TABLE teams(
    id SERIAL PRIMARY KEY,
    team_name VARCHAR(128) UNIQUE
);

CREATE TABLE users (
    id VARCHAR(256) PRIMARY KEY,
    username VARCHAR(2000) UNIQUE NOT NULL,
    team_id INTEGER REFERENCES teams(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL
);

CREATE TABLE pr (
    id VARCHAR(256) PRIMARY KEY,
    pr_name varchar(2000),
    author_id VARCHAR(256) NOT NULL REFERENCES users(id),
    pr_status VARCHAR(256),
    created_ad TIMESTAMP,
    mergerd_at TIMESTAMP
);

CREATE TABLE userspr (
    user_id    VARCHAR(256) NOT NULL REFERENCES users(id),
    request_id VARCHAR(256) NOT NULL REFERENCES pr(id),
    PRIMARY KEY (user_id, request_id)
);

CREATE TABLE usershistory(
    user_id VARCHAR(256) NOT NULL REFERENCES users(id),
    pr_count INTEGER,
    PRIMARY KEY (user_id)
);

CREATE INDEX idx_pr_author_id ON pr(author_id);
CREATE INDEX idx_users_team_id ON users(team_id);
CREATE INDEX idx_teams_team_name ON teams(team_name);
`

	_, err := db.Exec(schema)
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
		panic(err)
	}

	UR := &user.UserRepo{DB: testDB}
	TR := &team.TeamRepo{DB: testDB, UR: UR}
	repo := &pr.PullRequestRepo{DB: testDB, TR: TR, UR: UR}

	if err := setupTeam(ctx, testDB); err != nil {
		t.Fatalf("failed to setup team: %v", err)
	}
	ID := "prmrg"
	prReq := pr.CreatePullRequestRequest{ID: "prmrg", PullRequestName: "My First PR", AuthorID: "u"}
	createdPR, err := repo.Create(ctx, prReq)
	if err != nil {
		t.Fatalf("failed to create PR: %v", err)
	}
	if len(createdPR.AssignedReviewers) == 0 {
		t.Fatalf("expected assigned reviewers, got none")
	}

	gotPR, err := repo.GetPr(ctx, ID)
	if err != nil {
		t.Fatalf("failed to get PR: %v", err)
	}
	if gotPR.ID != prReq.ID {
		t.Fatalf("expected PR ID %s, got %s", prReq.ID, gotPR.ID)
	}

	mergedPR, err := repo.Merged(ctx, ID)
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
		t.Fatalf("Erorr setup")
	}
	if err := setupTeam(ctx, testDB); err != nil {
		t.Fatalf("Error setup")
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
