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

func (PR *PullRequestRepo) AssignedReviewer(ctx context.Context, PrID, UserID string) (*PullRequest, string, error) {
	teamID, err := PR.TR.GetTeamByUserID(ctx, UserID)
	if err != nil {
		return nil, "", err
	}

	pr, err := PR.GetPr(ctx, PrID)
	if err != nil {
		return nil, "", err
	}

	if pr.Status == "MERGED" {
		return nil, "", errs.PRMergedError
	}
<<<<<<< HEAD
=======

>>>>>>> dbe791d (-a)
	users, err := PR.TR.GetTeamMember(ctx, teamID)
	if err != nil {
		return nil, "", err
	}

	thisUsers := make(map[string]bool)
	for _, u := range pr.AssignedReviewers {
		thisUsers[u] = true
	}

	if !thisUsers[UserID] {
		return nil, "", errs.NotAssignedError
	}

	candidates := make([]string, 0)
	for _, u := range users {
		if !thisUsers[u.Id] && u.IsActive {
			candidates = append(candidates, u.Id)
		}
	}

	if len(candidates) == 0 {
		return nil, "", errs.NoCandidateError
	}

	newReviewer := candidates[rand.IntN(len(candidates))]

	tx, err := PR.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	updateQuery, args, err := psql.Update("userspr").
		Set("user_id", newReviewer).
		Where(sq.Eq{"user_id": UserID, "request_id": PrID}).
		ToSql()
	if err != nil {
		return nil, "", err
	}

	if _, err := tx.ExecContext(ctx, updateQuery, args...); err != nil {
		return nil, "", err
	}

	statBuilder := psql.Insert("usershistory").Columns("user_id", "pr_count")
	q, args, err := statBuilder.
		Values(newReviewer, 1).
		Suffix("ON CONFLICT (user_id) DO UPDATE SET pr_count = usershistory.pr_count + 1").
		ToSql()
	if err != nil {
		return nil, "", err
	}

	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return nil, "", err
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}

	updatedPR, err := PR.GetPr(ctx, PrID)
	return updatedPR, newReviewer, err
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

	activeusers := make([]*user.User, 0)
	for _, u := range users {
		if u.IsActive {
			activeusers = append(activeusers, u)
		}
	}

	reviews := selectReviewers(activeusers)

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
		Columns("id", "pr_name", "author_id", "pr_status").
		Values(req.ID, req.PullRequestName, req.AuthorID, "OPEN").
		ToSql()
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, insertPR, args...); err != nil {
		return nil, err
	}

	reviewBuilder := psql.Insert("userspr").Columns("user_id", "request_id")
	statBuilder := psql.Insert("usershistory").Columns("user_id", "pr_count")

	for _, reviewerID := range reviews {
		q, args, err := reviewBuilder.Values(reviewerID, req.ID).ToSql()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return nil, err
		}

		q, args, err = statBuilder.
			Values(reviewerID, 1).
			Suffix("ON CONFLICT (user_id) DO UPDATE SET pr_count = usershistory.pr_count + 1").
			ToSql()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return nil, err
		}
	}
<<<<<<< HEAD
	
=======

>>>>>>> dbe791d (-a)
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return pr, nil
}

func selectReviewers(users []*user.User) []string {
	reviews := make([]string, 0)

	if len(users) == 1 {
		reviews = append(reviews, users[0].Id)
	} else {
		v1 := rand.IntN(len(users))
		v2 := rand.IntN(len(users))
		if v1 == v2 {
			if v1+1 < len(users) {
				reviews = append(reviews, users[v1].Id, users[v1+1].Id)
			} else {
				reviews = append(reviews, users[v1].Id, users[v1-1].Id)
			}
		} else {
			reviews = append(reviews, users[v1].Id, users[v2].Id)
		}
	}

	return reviews
}
