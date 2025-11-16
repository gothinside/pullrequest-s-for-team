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
) // Postgres driver
var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = sql.Open("postgres", "host=localhost port=5429 user=postgres password=123 dbname=tr sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer testDB.Close()

	// // Clean DB
	// if err := cleanDB(testDB); err != nil {
	// 	panic(err)
	// }

	os.Exit(m.Run())
}

func cleanDB(db *sql.DB) error {
	_, err := db.Exec(`
			TRUNCATE TABLE userspr, pr, users, teams RESTART IDENTITY CASCADE;
		`)
	return err
}

// Mock team repo
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
	err := cleanDB(db)
	if err != nil {
		return err
	}
	return nil
}

func setupTeam(ctx context.Context, db *sql.DB) error {
	//
	_, err := db.Exec(`INSERT INTO teams (team_name) VALUES ($1)`, "Awesome Team")
	if err != nil {
		return err
	}

	// Insert users
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

	// // Assign users to the team
	// for _, u := range users {
	// 	if _, err := db.Exec(`INSERT INTO team_members (team_id, user_id) VALUES ($1, $2)`, "team1", u.id); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func TestPullRequestRepo_Create_GetMerge(t *testing.T) {
	ctx := context.Background()
	err := setupDB(ctx, testDB)
	if err != nil {
		t.Fatalf(err.Error())
	}
	UR := &user.UserRepo{DB: testDB}
	TR := &team.TeamRepo{DB: testDB, UR: UR}
	repo := &pr.PullRequestRepo{
		DB: testDB,
		TR: TR,
		UR: UR,
	}
	// Setup team and users in DB
	if err := setupTeam(ctx, testDB); err != nil {
		t.Fatalf("failed to setup team: %v", err)
	}

	// 1. Create PR
	prReq := pr.CreatePullRequestRequest{
		ID:              "pr1",
		PullRequestName: "My First PR",
		AuthorID:        "u",
	}
	createdPR, err := repo.Create(ctx, prReq)
	if err != nil {
		t.Fatalf("failed to create PR: %v", err)
	}
	if len(createdPR.AssignedReviewers) == 0 {
		t.Fatalf("expected assigned reviewers, got none")
	}

	// 2. Get PR
	gotPR, err := repo.GetPr(ctx, "pr1")
	if err != nil {
		t.Fatalf("failed to get PR: %v", err)
	}
	if gotPR.ID != prReq.ID {
		t.Fatalf("expected PR ID %s, got %s", prReq.ID, gotPR.ID)
	}

	// 3. Merge PR
	mergedPR, err := repo.Merged(ctx, "pr1")
	if err != nil {
		t.Fatalf("failed to merge PR: %v", err)
	}
	if mergedPR.Status != "MERGED" {
		t.Fatalf("expected PR status MERGED, got %s", mergedPR.Status)
	}
}

func insertPR(t *testing.T, prID string, authorID string, reviewers []string) {
	// Insert PR
	_, err := testDB.Exec(`INSERT INTO pr (id, pr_name, author_id, pr_status) VALUES ($1, $2, $3, $4)`,
		prID, "My PR", authorID, "OPEN")
	if err != nil {
		t.Fatalf("failed to insert PR: %v", err)
	}

	// Insert assigned reviewers
	for _, r := range reviewers {
		_, err := testDB.Exec(`INSERT INTO userspr (user_id, request_id) VALUES ($1, $2)`, r, prID)
		if err != nil {
			t.Fatalf("failed to insert reviewer %s: %v", r, err)
		}
	}
}

func TestAssignedReviewerIntegration(t *testing.T) {
	err := setupDB(context.Background(), testDB)
	err = setupTeam(context.Background(), testDB)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	UR := &user.UserRepo{DB: testDB}
	TR := &team.TeamRepo{DB: testDB, UR: UR}
	repo := &pr.PullRequestRepo{
		DB: testDB,
		TR: TR,
		UR: UR,
	}

	// Insert PR with one reviewer
	insertPR(t, "pr1", "user1", []string{"user1"})

	// Call AssignedReviewer to replace user1
	updatedPR, newReviewer, err := repo.AssignedReviewer(ctx, "pr1", "user1")
	if err != nil {
		t.Fatalf("failed to assign reviewer: %v", err)
	}

	// Assertions
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

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"math/rand"
// 	"pullreq/internal/errs"
// 	"pullreq/internal/pr"
// 	"pullreq/internal/team"
// 	"pullreq/internal/user"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// )

// // --- Mock TeamRepo ---
// type mockTeamRepo struct {
// 	team.TeamRepoInterface
// 	GetTeamByUserIDFunc func(ctx context.Context, userID string) (int, error)
// 	GetTeamMemberFunc   func(ctx context.Context, teamID int) ([]*user.User, error)
// }

// func (m *mockTeamRepo) GetTeamByUserID(ctx context.Context, userID string) (int, error) {
// 	return m.GetTeamByUserIDFunc(ctx, userID)
// }

// func (m *mockTeamRepo) GetTeamMember(ctx context.Context, teamID int) ([]*user.User, error) {
// 	return m.GetTeamMemberFunc(ctx, teamID)
// }

// // --- Mock PullRequestRepo to override Check and GetPr ---
// type mockPRRepo struct {
// 	pr.PullRequestRepo
// 	CheckFunc func(ctx context.Context, ID string) error
// 	GetPrFunc func(ctx context.Context, ID string) (*pr.PullRequest, error)
// }

// func (m *mockPRRepo) Check(ctx context.Context, ID string) error {
// 	if m.CheckFunc != nil {
// 		return m.CheckFunc(ctx, ID)
// 	}
// 	return nil
// }

// func (m *mockPRRepo) GetPr(ctx context.Context, ID string) (*pr.PullRequest, error) {
// 	if m.GetPrFunc != nil {
// 		return m.GetPrFunc(ctx, ID)
// 	}
// 	return nil, errors.New("not implemented")
// }

// // --- Test Create Pull Request ---
// func TestCreatePullRequest_LogicOnly(t *testing.T) {
// 	ctx := context.Background()
// 	rand.Seed(time.Now().UnixNano())

// 	mockTeam := &mockTeamRepo{
// 		GetTeamByUserIDFunc: func(ctx context.Context, userID string) (int, error) {
// 			return 1, nil
// 		},
// 		GetTeamMemberFunc: func(ctx context.Context, teamID int) ([]*user.User, error) {
// 			return []*user.User{
// 				{Id: "u1", Username: "user1"},
// 				{Id: "u2", Username: "user2"},
// 			}, nil
// 		},
// 	}

// 	prRepo := &mockPRRepo{
// 		PullRequestRepo: pr.PullRequestRepo{
// 			DB: nil, // not used
// 			TR: mockTeam,
// 		},
// 		CheckFunc: func(ctx context.Context, ID string) error {
// 			return nil // PR does not exist
// 		},
// 	}

// 	req := pr.CreatePullRequestRequest{
// 		ID:              "pr1",
// 		PullRequestName: "My PR",
// 		AuthorID:        "u1",
// 	}

// 	result, err := prRepo.Create(ctx, req)
// 	assert.NoError(t, err)
// 	assert.Equal(t, "pr1", result.ID)
// 	assert.Equal(t, "OPEN", result.Status)
// 	assert.Len(t, result.AssignedReviewers, 2)
// 	assert.Contains(t, []string{"u1", "u2"}, result.AssignedReviewers[0])
// }

// // --- Test AssignedReviewer Success ---
// func TestAssignedReviewer_Success(t *testing.T) {
// 	ctx := context.Background()

// 	mockTeam := &mockTeamRepo{
// 		GetTeamByUserIDFunc: func(ctx context.Context, userID string) (int, error) {
// 			return 1, nil
// 		},
// 		GetTeamMemberFunc: func(ctx context.Context, teamID int) ([]*user.User, error) {
// 			return []*user.User{
// 				{Id: "u1"}, {Id: "u2"}, {Id: "u3"},
// 			}, nil
// 		},
// 	}
// 	mockPR := &mockPRRepo{
// 		PullRequestRepo: pr.PullRequestRepo{
// 			DB: nil,
// 			TR: mockTeam,
// 		},
// 		GetPrFunc: func(ctx context.Context, ID string) (*pr.PullRequest, error) {
// 			return &pr.PullRequest{
// 				ID:                "pr1",
// 				Status:            "OPEN",
// 				AssignedReviewers: []string{"u1"},
// 			}, nil
// 		},
// 	}
// 	fmt.Println(111)

// 	newPR, newReviewer, err := mockPR.AssignedReviewer(ctx, "pr1", "u1")
// 	assert.NoError(t, err)
// 	assert.NotEqual(t, "u1", newReviewer)
// 	assert.Contains(t, []string{"u2", "u3"}, newReviewer)
// 	assert.Equal(t, "pr1", newPR.ID)
// }

// // --- Test AssignedReviewer No Candidates ---
// func TestAssignedReviewer_NoCandidates(t *testing.T) {
// 	ctx := context.Background()

// 	mockTeam := &mockTeamRepo{
// 		GetTeamByUserIDFunc: func(ctx context.Context, userID string) (int, error) {
// 			return 1, nil
// 		},
// 		GetTeamMemberFunc: func(ctx context.Context, teamID int) ([]*user.User, error) {
// 			return []*user.User{
// 				{Id: "u1"}, // Only the current reviewer
// 			}, nil
// 		},
// 	}

// 	mockPR := &mockPRRepo{
// 		PullRequestRepo: pr.PullRequestRepo{
// 			DB: nil,
// 			TR: mockTeam,
// 		},
// 		GetPrFunc: func(ctx context.Context, ID string) (*pr.PullRequest, error) {
// 			return &pr.PullRequest{
// 				ID:                "pr1",
// 				Status:            "OPEN",
// 				AssignedReviewers: []string{"u1"},
// 			}, nil
// 		},
// 	}

// 	_, _, err := mockPR.AssignedReviewer(ctx, "pr1", "u1")
// 	assert.Error(t, err)
// 	assert.Equal(t, errs.NotFountError, err)
// }

// // --- Test Merged Pull Request ---
// func TestMergedPullRequest(t *testing.T) {
// 	ctx := context.Background()

// 	mockPR := &mockPRRepo{
// 		PullRequestRepo: pr.PullRequestRepo{
// 			DB: nil,
// 			TR: &mockTeamRepo{},
// 		},
// 		GetPrFunc: func(ctx context.Context, ID string) (*pr.PullRequest, error) {
// 			return &pr.PullRequest{
// 				ID:     "pr1",
// 				Status: "OPEN",
// 			}, nil
// 		},
// 	}

// 	pr, err := mockPR.Merged(ctx, "pr1")
// 	assert.NoError(t, err)
// 	assert.Equal(t, "MERGED", pr.Status)
// }
