package team_test

import (
	"context"
	"database/sql"
	"pullreq/internal/team"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	connStr := "postgresql://postgres:123@localhost:5429/tr?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("cannot connect DB: %v", err)
	}

	db.Exec("DROP TABLE IF EXISTS users")
	db.Exec(`
		CREATE TABLE users (
			id VARCHAR(64) PRIMARY KEY,
			username VARCHAR(128),
			team_id INTEGER,
			is_active BOOLEAN DEFAULT TRUE
		)
	`)

	return db
}
func TestAddUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &team.TeamRepo{DB: db}

	mock.ExpectBegin()

	mock.ExpectExec(`UPDATE users`).
		WithArgs(false, 10). // пример деактивации
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	ctx := context.Background()

	// вызов метода
	err = repo.Deactivation(ctx, 10)
	require.NoError(t, err)
}
