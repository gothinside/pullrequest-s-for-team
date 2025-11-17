package user_test

import (
	"context"
	"testing"

	"pullreq/internal/user"

	"github.com/DATA-DOG/go-sqlmock"
)

// --- Helper to setup UserRepo with sqlmock ---
func setupUserRepo(t *testing.T) (*user.UserRepo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	repo := &user.UserRepo{DB: db}
	return repo, mock, func() { db.Close() }
}

// --- Test UpdateUserActivity ---
func TestUserRepo_UpdateUserActivity(t *testing.T) {
	repo, mock, teardown := setupUserRepo(t)
	defer teardown()

	userID := "u1"
	isActive := true

	mock.ExpectQuery(`UPDATE users SET is_active = \$1 WHERE id = \$2 RETURNING id, username, team_id, is_active`).
		WithArgs(true, "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "team_id", "is_active"}).AddRow("u1", "Alice", 1, true))

	updated, err := repo.UpdateUserActivity(context.Background(), userID, isActive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Id != userID || !updated.IsActive {
		t.Errorf("expected updated user active, got %+v", updated)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Test GetUsersPrShort ---
func TestUserRepo_GetUsersPrShort(t *testing.T) {
	repo, mock, teardown := setupUserRepo(t)
	defer teardown()

	userID := "u1"
	rows := sqlmock.NewRows([]string{"id", "pr_name", "author_id", "pr_status"}).
		AddRow("pr-1", "Add feature", "u1", "OPEN").
		AddRow("pr-2", "Fix bug", "u2", "MERGED")

	mock.ExpectQuery(`SELECT pr.id, pr.pr_name, pr.author_id, pr.pr_status FROM userspr`).
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := repo.GetUsersPrShort(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 PRs, got %d", len(result))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
