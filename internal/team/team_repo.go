package team

import (
	"context"
	"database/sql"
	"fmt"
	"pullreq/internal/errs"
	"pullreq/internal/user"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

type Team struct {
	ID       int          `json:"-"`
	TeamName string       `json:"team_name"`
	Members  []*user.User `json:"members"`
}

type TeamRepoInterface interface {
	AddTeam(ctx context.Context, teamName string, members []TeamMember) (*Team, error)
	GetTeamWithMembers(ctx context.Context, teamName string) (*Team, error)
	GetTeamByUserID(ctx context.Context, userID string) (int, error)
	GetTeamMember(ctx context.Context, teamID int) ([]*user.User, error)
	Deactivation(ctx context.Context, teanName string) error
}

type TeamRepo struct {
	DB *sql.DB
	UR user.UserRepoInterface
}

func (TR *TeamRepo) Deactivation(ctx context.Context, teamName string) error {
	tx, err := TR.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q, args, err := psql.
		Update("users u").
		Set("is_active", false).
		From("teams t").
		Where("u.team_id = t.id AND t.team_name = ?", teamName).
		ToSql()

	if err != nil {
		tx.Rollback()
		return err
	}

	if _, err = tx.ExecContext(ctx, q, args...); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (TR *TeamRepo) GetTeamByUserID(ctx context.Context, userID string) (int, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q, args, err := psql.
		Select("team_id").
		From("users").
		Where(sq.Eq{"id": userID}).
		ToSql()
	if err != nil {
		return -1, err
	}

	var id int = -1
	rows, err := TR.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	var flag bool
	for rows.Next() {
		rows.Scan(&id)
		flag = true
	}
	if !flag {
		return -1, errs.NotFountError
	}
	return id, nil
}

func (TR *TeamRepo) AddTeam(ctx context.Context, teamName string, members []TeamMember) (*Team, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	tx, err := TR.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	sqlTeam, argsTeam, err := psql.
		Insert("teams").
		Columns("team_name").
		Values(teamName).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return nil, err
	}

	var teamID int
	if err := tx.QueryRowContext(ctx, sqlTeam, argsTeam...).Scan(&teamID); err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return nil, errs.ExistError
		}
		return nil, err
	}

	team := &Team{
		ID:       teamID,
		TeamName: teamName,
		Members:  make([]*user.User, 0, len(members)),
	}

	if len(members) > 0 {
		insertBuilder := psql.Insert("users").Columns("id", "username", "team_id", "is_active")
		for _, m := range members {
			insertBuilder = insertBuilder.Values(m.UserID, m.Username, teamID, m.IsActive)
			team.Members = append(team.Members, &user.User{
				Id:       m.UserID,
				Username: m.Username,
				TeamID:   teamID,
				IsActive: m.IsActive,
			})
		}
		sqlUsers, argsUsers, err := insertBuilder.ToSql()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, sqlUsers, argsUsers...); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to insert users: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return team, nil
}

func (TR *TeamRepo) GetTeamMember(ctx context.Context, teamID int) ([]*user.User, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	builder := psql.
		Select(
			"id",
			"username",
			"is_active",
		).
		From("users").
		Where(sq.Eq{"team_id": teamID})

	q, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("invalid data: %w", err)
	}

	rows, err := TR.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teamMembers := make([]*user.User, 0)
	for rows.Next() {
		user := &user.User{}
		err = rows.Scan(
			&user.Id,
			&user.Username,
			&user.IsActive,
		)
		if err != nil {
			return nil, err
		}
		teamMembers = append(teamMembers, user)

	}

	return teamMembers, nil
}

func (TR *TeamRepo) GetTeamWithMembers(ctx context.Context, teamName string) (*Team, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	builder := psql.
		Select(
			"t.id",
			"t.team_name",
			"COALESCE(u.id, '-1') AS user_id",
			"COALESCE(u.username, ' ') AS username",
			"COALESCE(u.is_active, FALSE) AS is_active",
		).
		From("teams AS t").
		LeftJoin("users AS u ON u.team_id = t.id").
		Where(sq.Eq{"t.team_name": teamName})

	q, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("invalid data: %w", err)
	}

	rows, err := TR.DB.QueryContext(ctx, q, args...)
	team := &Team{Members: []*user.User{}}
	foundTeamData := false
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		user := &user.User{}
		err = rows.Scan(
			&team.ID,
			&team.TeamName,
			&user.Id,
			&user.Username,
			&user.IsActive,
		)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		foundTeamData = true

		if user.Id != "-1" {
			team.Members = append(team.Members, user)
		}
	}

	if !foundTeamData {
		return nil, sql.ErrNoRows
	}

	return team, nil
}
