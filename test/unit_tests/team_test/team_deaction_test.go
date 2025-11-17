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
func TestTeamRepo_Deactivation_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tr := &team.TeamRepo{DB: db}

	ctx := context.Background()
	teamName := "backend"

	// Expect transaction begin
	mock.ExpectBegin()

	// SQL ожидаемый для обновления юзеров
	mock.ExpectExec(`UPDATE users u SET is_active = \$1 FROM teams t WHERE u.team_id = t.id AND t.team_name = \$2`).
		WithArgs(false, teamName).
		WillReturnResult(sqlmock.NewResult(0, 2))

	// Expect commit
	mock.ExpectCommit()

	err = tr.Deactivation(ctx, teamName)
	require.NoError(t, err)

	// Проверка, что все ожидания выполнены
	require.NoError(t, mock.ExpectationsWereMet())
}
