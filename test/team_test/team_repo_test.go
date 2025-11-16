package team_test

import (
	"context"
	"errors"
	"testing"

	"pullreq/internal/errs"
	"pullreq/internal/team"
	"pullreq/internal/user"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestTeamRepo_AddTeam_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ur := &user.UserRepo{DB: db}
	tr := &team.TeamRepo{DB: db, UR: ur}

	ctx := context.Background()

	teamName := "backend"
	members := []team.TeamMember{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
	}

	// Transaction begin
	mock.ExpectBegin()

	// Insert team
	mock.ExpectQuery(`INSERT INTO teams`).
		WithArgs(teamName).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	// Insert users (ON CONFLICT DO UPDATE)
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs("u1", "Alice", 10, true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs("u2", "Bob", 10, true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Commit
	mock.ExpectCommit()

	res, err := tr.AddTeam(ctx, teamName, members)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TeamName != teamName {
		t.Fatalf("expected team name %s, got %s", teamName, res.TeamName)
	}

	if len(res.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(res.Members))
	}
}

func TestTeamRepo_AddTeam_AlreadyExists(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ur := &user.UserRepo{DB: db}
	tr := &team.TeamRepo{DB: db, UR: ur}

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`INSERT INTO teams`).
		WillReturnError(&pq.Error{Code: "23505"}) // unique violation

	mock.ExpectRollback()

	_, err := tr.AddTeam(ctx, "backend", []team.TeamMember{})
	if !errors.Is(err, errs.ExistError) && err == nil {
		t.Fatalf("expected exist error, got: %v", err)
	}
}

func TestTeamRepo_GetTeamWithMembers_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ur := &user.UserRepo{DB: db}
	tr := &team.TeamRepo{DB: db, UR: ur}

	rows := sqlmock.NewRows([]string{"id", "team_name", "user_id", "username", "is_active"}).
		AddRow(10, "backend", "u1", "Alice", true).
		AddRow(10, "backend", "u2", "Bob", false)

	mock.ExpectQuery(`SELECT (.+) FROM teams AS t LEFT JOIN users`).
		WithArgs("backend").
		WillReturnRows(rows)

	res, err := tr.GetTeamWithMembers(context.Background(), "backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TeamName != "backend" {
		t.Fatalf("expected backend, got %s", res.TeamName)
	}

	if len(res.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(res.Members))
	}
}

func TestTeamRepo_GetTeamByUserID_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ur := &user.UserRepo{DB: db}
	tr := &team.TeamRepo{DB: db, UR: ur}

	rows := sqlmock.NewRows([]string{"team_id"}) // no rows
	mock.ExpectQuery(`SELECT team_id FROM users WHERE id =`).
		WithArgs("u100").
		WillReturnRows(rows)

	_, err := tr.GetTeamByUserID(context.Background(), "u100")
	if !errors.Is(err, errs.NotFountError) {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestTeamRepo_GetTeamMember_OK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ur := &user.UserRepo{DB: db}
	tr := &team.TeamRepo{DB: db, UR: ur}

	rows := sqlmock.NewRows([]string{"id", "username", "is_active"}).
		AddRow("u1", "Alice", true).
		AddRow("u2", "Bob", false)

	mock.ExpectQuery(`SELECT id, username, is_active FROM users WHERE team_id`).
		WithArgs(10).
		WillReturnRows(rows)

	list, err := tr.GetTeamMember(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 members, got %d", len(list))
	}
}

type mockTeamRepo struct {
	GetTeamWithMembersFunc func(teamName string) (*team.Team, error)
}

func (m *mockTeamRepo) GetTeams(ctx context.Context) ([]team.Team, error) {
	return []team.Team{}, nil
}

func (m *mockTeamRepo) GetTeamByID(ctx context.Context, id int) (*team.Team, error) {
	return &team.Team{ID: id, TeamName: "Mock"}, nil
}

// тут важно: mock должно прокидывать вызов в GetTeamWithMembersFunc
func (m *mockTeamRepo) GetTeamWithMembers(teamName string) (*team.Team, error) {
	return m.GetTeamWithMembersFunc(teamName)
}
