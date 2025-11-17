package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pullreq/internal/pr"
	"pullreq/internal/team"
	"pullreq/internal/user"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

type TestEnv struct {
	DB     *sql.DB
	Server *httptest.Server
}

func SetupTestEnv() (*TestEnv, error) {
	var err error
	db, err := sql.Open("postgres", "host=localhost port=5424 user=postgres password=123 dbname=tr sslmode=disable")
	if err != nil {
		panic(err)
	}

	userRepo := &user.UserRepo{DB: db}
	teamRepo := &team.TeamRepo{DB: db, UR: userRepo}
	prRepo := &pr.PullRequestRepo{DB: db, UR: userRepo, TR: teamRepo}

	teamRouter := &team.TeamRouter{TR: teamRepo}
	userRouter := &user.UserRouter{UR: userRepo}
	prRouter := &pr.PrRouter{PR: prRepo}

	r := chi.NewRouter()

	r.Route("/team", func(r chi.Router) {
		r.Post("/add", teamRouter.HandleAddTeam)
		r.Get("/get", teamRouter.GetTeamWithMembersHandler)
		r.Post("/deactivation", teamRouter.DeactivateTeam)
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", userRouter.RouterSetActiviry)
		r.Get("/getReview", userRouter.GetUserReviewsHandler)
		r.Get("/getStat", userRouter.GetStat)
	})

	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", prRouter.CreatePullRequest)
		r.Post("/merge", prRouter.Merge)
		r.Post("/reassign", prRouter.AssignedReviewer)
	})

	return &TestEnv{
		DB:     db,
		Server: httptest.NewServer(r),
	}, nil
}

func Test_CreateTeam_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()
	defer env.DB.Close()

	body := `{
           "team_name": "payments2",
           "members": [
             {
               "user_id": "u1",
               "username": "Alice",
               "is_active": true
             },
             {
               "user_id": "u2",
               "username": "Bob",
               "is_active": true
             },
			 {
               "user_id": "u3",
               "username": "Vlad",
               "is_active": true
             }
           ]
         }`
	resp, err := http.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)
}

func Test_GetTeam_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()

	resp, err := http.Get(env.Server.URL + "/team/get?team_name=payments2")
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}

func Test_DeactivateTeam_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()

	body := `{"team_id": 1}`

	resp, err := http.Post(env.Server.URL+"/team/deactivation", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}

func Test_CreatePullRequest_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()

	body := `{
		"pull_request_id": "pr1",
		"pull_request_name": "Fixcrash",
		"author_id": "u1"
	}`

	resp, err := http.Post(env.Server.URL+"/pullRequest/create", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	out := make(map[string]pr.PullRequest)
	err = json.Unmarshal(respBody, &out)
	require.NoError(t, err)

	require.Equal(t, "pr1", out["pr"].ID)
}

func Test_AssignReviewer_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()
	defer env.DB.Close()

	body := `{
		"pull_request_id": "pr1",
		"old_user_id": "u2"
	}`

	resp, err := http.Post(env.Server.URL+"/pullRequest/reassign", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	data, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &out))
}
