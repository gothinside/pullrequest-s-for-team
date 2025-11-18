package pr

import (
	"context"
	"database/sql"
	"math/rand/v2"
	"pullreq/internal/errs"
	"pullreq/internal/team"
	"pullreq/internal/user"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type PullReqestInput struct {
	PrID     string
	PrName   string
	AuthorID string
}

type PullRequestRepo struct {
	DB *sql.DB
	TR team.TeamRepoInterface
	UR user.UserRepoInterface
}

type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type PullRequestRepoInterface interface {
	AssignedReviewer(ctx context.Context, PrID, UserID string) (*PullRequest, string, error)
	Check(ctx context.Context, ID string) error
	GetPr(ctx context.Context, ID string) (*PullRequest, error)
	Merged(ctx context.Context, ID string) (*PullRequest, error)
	Create(ctx context.Context, req CreatePullRequestRequest) (*PullRequest, error)
}

func (PR *PullRequestRepo) AssignedReviewer(ctx context.Context, prID, userID string) (*PullRequest, string, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	tx, err := PR.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	var status string
	lockQuery, args, _ := psql.Select("pr_status").
		From("pr").
		Where(sq.Eq{"id": prID}).
		Suffix("FOR UPDATE").
		ToSql()

	err = tx.QueryRowContext(ctx, lockQuery, args...).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", errs.NotFountError
		}
		return nil, "", err
	}

	if status == "MERGED" {
		return nil, "", errs.PRMergedError
	}

	teamID, err := PR.TR.GetTeamByUserID(ctx, userID)
	if err != nil {
		return nil, "", err
	}

	newReviewerQuery, args, _ := psql.
		Select("id").
		From("users").
		Where(sq.Eq{"team_id": teamID, "is_active": true}).
		Where(sq.NotEq{"id": userID}).
		Limit(1).
		ToSql()

	var newReviewer string
	err = tx.QueryRowContext(ctx, newReviewerQuery, args...).Scan(&newReviewer)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", errs.NoCandidateError
		}
		return nil, "", err
	}
	updateQuery, args, _ := psql.Update("userspr").
		Set("user_id", newReviewer).
		Where(sq.Eq{"user_id": userID, "request_id": prID}).
		ToSql()

	_, err = tx.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		return nil, "", err
	}

	historyQuery, args, _ := psql.Insert("usershistory").
		Columns("user_id", "pr_count").
		Values(newReviewer, 1).
		Suffix("ON CONFLICT (user_id) DO UPDATE SET pr_count = usershistory.pr_count + 1").
		ToSql()

	_, err = tx.ExecContext(ctx, historyQuery, args...)
	if err != nil {
		return nil, "", err
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}

	pr, err := PR.GetPr(ctx, prID)
	return pr, newReviewer, err
}

func (PR *PullRequestRepo) Check(ctx context.Context, ID string) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	checkQuery, checkArgs, err := psql.Select("TRUE").From("pr").Where(sq.Eq{"id": ID}).ToSql()
	if err != nil {
		return err
	}

	rows, err := PR.DB.QueryContext(ctx, checkQuery, checkArgs...)
	if err != nil {
		return err
	}

	for rows.Next() {
		return errs.ExistError
	}

	return nil
}

func (PR *PullRequestRepo) GetPr(ctx context.Context, ID string) (*PullRequest, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q, args, err := psql.
		Select("pr.id, pr.pr_name, pr.author_id, pr.pr_status, ur.user_id").
		From("pr").
		Join("userspr ur on ur.request_id = pr.id").
		Where(sq.Eq{"ID": ID}).
		ToSql()

	rows, err := PR.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}

	res := &PullRequest{}
	var exist bool
	for rows.Next() {
		exist = true
		var userID string
		rows.Scan(&res.ID, &res.PullRequestName, &res.AuthorID, &res.Status, &userID)
		res.AssignedReviewers = append(res.AssignedReviewers, userID)
	}

	if !exist {
		return nil, errs.NotFountError
	}

	return res, nil
}

func (PR *PullRequestRepo) Merged(ctx context.Context, ID string) (*PullRequest, error) {
	pr, err := PR.GetPr(ctx, ID)
	if err != nil {
		return nil, err
	}

	if pr.Status == "MERGED" {
		return pr, nil
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	q, args, err := psql.Update("pr").
		Set("pr_status", "MERGED").
		Set("mergerd_at", time.Now()).
		Where(sq.Eq{"ID": ID}).
		ToSql()

	_, err = PR.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}

	pr.Status = "MERGED"
	return pr, nil
}

func (PR *PullRequestRepo) Create(ctx context.Context, req CreatePullRequestRequest) (*PullRequest, error) {
	if err := PR.Check(ctx, req.ID); err != nil {
		return nil, err
	}

	teamID, err := PR.TR.GetTeamByUserID(ctx, req.AuthorID)
	if err != nil {
		return nil, err
	}

	users, err := PR.TR.GetTeamMember(ctx, teamID)
	if err != nil {
		return nil, err
	}

	activeUsers := make([]*user.User, 0)
	for _, u := range users {
		if u.IsActive {
			activeUsers = append(activeUsers, u)
		}
	}

	reviews := selectReviewers(activeUsers)

	pr := &PullRequest{
		ID:                req.ID,
		PullRequestName:   req.PullRequestName,
		AuthorID:          req.AuthorID,
		Status:            "OPEN",
		AssignedReviewers: reviews,
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	tx, err := PR.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	insertPR, args, err := psql.Insert("pr").
		Columns("id", "pr_name", "author_id", "pr_status", "created_ad").
		Values(req.ID, req.PullRequestName, req.AuthorID, "OPEN", time.Now()).
		ToSql()
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, insertPR, args...); err != nil {
		return nil, err
	}

	if len(reviews) > 0 {
		reviewBuilder := psql.Insert("userspr").Columns("user_id", "request_id")
		for _, reviewerID := range reviews {
			reviewBuilder = reviewBuilder.Values(reviewerID, req.ID)
		}
		q, args, err := reviewBuilder.ToSql()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return nil, err
		}
	}

	if len(reviews) > 0 {
		userIDs := make([]string, len(reviews))
		for i, id := range reviews {
			userIDs[i] = id
		}

		updateSQL, args, err := psql.Update("usershistory").
			Set("pr_count", sq.Expr("pr_count + 1")).
			Where(sq.Eq{"user_id": userIDs}).
			ToSql()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, updateSQL, args...); err != nil {
			return nil, err
		}

		insertBuilder := psql.Insert("usershistory").
			Columns("user_id", "pr_count")
		for _, id := range reviews {
			insertBuilder = insertBuilder.Values(id, 1)
		}
		insertBuilder = insertBuilder.Suffix("ON CONFLICT DO NOTHING")

		insertSQL, args, err := insertBuilder.ToSql()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, insertSQL, args...); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return pr, nil
}

func selectReviewers(users []*user.User) []string {
	if len(users) == 0 {
		return []string{}
	}

	v1 := rand.IntN(len(users))

	v2 := rand.IntN(len(users))
	if len(users) == 1 {
		return []string{users[v1].Id}
	}
	if v1 == v2 {
		v2 = (v1 + 1) % len(users)
	}

	return []string{users[v1].Id, users[v2].Id}
}
