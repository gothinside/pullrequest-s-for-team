package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pullreq/internal/errs"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
)

type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type User struct {
	Id       string
	Username string
	TeamID   int
	IsActive bool
}

type UserRepo struct {
	DB *sql.DB
}

type UserRepoInterface interface {
	AddUser(ctx context.Context, tx *sql.Tx, user *User) error //Better use transaction manager from avito:)
	UpdateUser(ctx context.Context, user User) error
	UpdateUserActivity(ctx context.Context, userID string, isActive bool) (*User, error)
	GetUsersPrShort(ctx context.Context, userID string) ([]PullRequestShort, error)
	GetStatAboutUser(ctx context.Context, userID string) (int, error)
}

func (UR *UserRepo) GetStatAboutUser(ctx context.Context, userID string) (int, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Select("pr_count").
		From("usershistory").
		Where(sq.Eq{"user_id": userID}).
		ToSql()

	var res int
	if err != nil {
		return -1, err
	}

	rows, err := UR.DB.QueryContext(ctx, sql, args...)
	if err != nil {
		return -1, err
	}
	defer rows.Close()
	var flag bool
	for rows.Next() {
		flag = true
		err = rows.Scan(&res)
		if err != nil {
			return -1, err
		}
	}
	if !flag {
		return -1, errs.NotFountError
	}
	return res, nil

}

func (UR *UserRepo) GetUsersPrShort(ctx context.Context, userID string) ([]PullRequestShort, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql, args, err := psql.
		Select(
			"pr.id",
			"pr.pr_name",
			"pr.author_id",
			"pr.pr_status",
		).
		From("userspr").
		Join("pr ON pr.id = userspr.request_id").
		Where(sq.Eq{"userspr.user_id": userID}).
		ToSql()

	if err != nil {
		return nil, err
	}

	rows, err := UR.DB.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]PullRequestShort, 0)

	for rows.Next() {
		var item PullRequestShort
		if err := rows.Scan(
			&item.PullRequestID,
			&item.PullRequestName,
			&item.AuthorID,
			&item.Status,
		); err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (UR *UserRepo) AddUser(ctx context.Context, tx *sql.Tx, user *User) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql, args, err := psql.Insert("users").
		Columns("id", "username", "team_id", "is_active").
		Values(user.Id, user.Username, user.TeamID, user.IsActive).
		Suffix(`
ON CONFLICT (id) DO UPDATE 
SET username = EXCLUDED.username,
    is_active = EXCLUDED.is_active,
    team_id = EXCLUDED.team_id
`).ToSql()
	if err != nil {
		return err
	}
	// Use a transaction or DB to execute
	_, err = tx.Exec(sql, args...)
	fmt.Println(err)
	if err != nil {
		return err
	}

	return nil
}

func (UR *UserRepo) UpdateUser(ctx context.Context, user User) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Update("users").
		Set("username", user.Username).
		Set("is_active", user.IsActive).
		Where(sq.Eq{"id": user.Id}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = UR.DB.Exec(sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func (UR *UserRepo) UpdateUserActivity(ctx context.Context, userID string, isActive bool) (*User, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q, args, err := psql.Update("users").
		Set("is_active", isActive).
		Where(sq.Eq{"id": userID}).
		Suffix("RETURNING id, username, team_id, is_active").
		ToSql()
	if err != nil {
		return nil, err
	}

	updatedUser := &User{}

	err = UR.DB.QueryRowContext(ctx, q, args...).Scan(
		&updatedUser.Id,
		&updatedUser.Username,
		&updatedUser.TeamID,
		&updatedUser.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return updatedUser, nil
}
