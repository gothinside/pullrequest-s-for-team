package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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

type Test struct {
	Body           string
	Method         string
	Url            string
	IsOut          bool
	ExpectedOutput string
	ExpectedStatus int
}

func (t *Test) Run(server *httptest.Server) error {
	url := t.Url
	var resp *http.Response
	var err error

	if t.Method == "GET" {
		resp, err = http.Get(url)
	} else {
		resp, err = http.Post(url, "application/json", bytes.NewBufferString(t.Body))
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != t.ExpectedStatus {
		return fmt.Errorf("wrong status: got=%d expected=%d", resp.StatusCode, t.ExpectedStatus)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var actual interface{}
	var expected interface{}

	if err := json.Unmarshal(respBody, &actual); err != nil {
		return fmt.Errorf("invalid actual JSON: %w", err)
	}

	if err := json.Unmarshal([]byte(t.ExpectedOutput), &expected); err != nil {
		return fmt.Errorf("invalid expected JSON: %w", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf(
			"json mismatch\nactual:   %s\nexpected: %s",
			string(respBody),
			t.ExpectedOutput,
		)
	}

	return nil
}

func Test_E2EFULL(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()
	defer env.DB.Close()
	//...No comments why i stoped
	tests := []*Test{
		&Test{
			Body: `{
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
         }`,
			Url:            env.Server.URL + "/team/add",
			ExpectedStatus: 201,
			ExpectedOutput: `{"team":{"team_name":"payments2","members":[{"user_id":"u1","username":"Alice","is_active":true},{"user_id":"u2","username":"Bob","is_active":true},{"user_id":"u3","username":"Vlad","is_active":true}]}}`,
		},
		&Test{
			Body:           "",
			Method:         "GET",
			Url:            env.Server.URL + "/team/get?team_name=payments2",
			ExpectedStatus: 200,
			ExpectedOutput: `{"team":{"team_name":"payments2","members":[{"id":"u1","username":"Alice","team_name":"payments2","is_active":true},{"id":"u2","username":"Bob","team_name":"payments2","is_active":true},{"id":"u3","username":"Vlad","team_name":"payments2","is_active":true}]}}`,
		},
	}
	for _, test := range tests {
		err = test.Run(env.Server)
		require.NoError(t, err)
	}
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

// func Test_CreateTeam_E2E(t *testing.T) {
// 	env, err := SetupTestEnv()
// 	require.NoError(t, err)
// 	defer env.Server.Close()
// 	defer env.DB.Close()

// 	body := `{
//            "team_name": "payments2",
//            "members": [
//              {
//                "user_id": "u1",
//                "username": "Alice",
//                "is_active": true
//              },
//              {
//                "user_id": "u2",
//                "username": "Bob",
//                "is_active": true
//              },
// 			 {
//                "user_id": "u3",
//                "username": "Vlad",
//                "is_active": true
//              }
//            ]
//          }`
// 	resp, err := http.Post(env.Server.URL+"/team/add", "application/json", bytes.NewBufferString(body))
// 	require.NoError(t, err)
// 	require.Equal(t, 201, resp.StatusCode)
// }

func Test_GetTeam_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()

	resp, err := http.Get(env.Server.URL + "/team/get?team_name=payments2")
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
	// require.Equal(t, "[]", out["pr"].AssignedReviewers)
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
	status := 0
	resp, err := http.Post(env.Server.URL+"/pullRequest/reassign", "application/json", bytes.NewBufferString(body))
	for i := 0; i < 3; i++ {
		require.NoError(t, err)
		if resp.StatusCode == 200 {
			status = 200
		}
	}
	require.NoError(t, err)
	require.Equal(t, status, resp.StatusCode)
}

func Test_MERGE_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()

	body := `{
		 "pull_request_id": "pr1"
	}`

	resp, err := http.Post(env.Server.URL+"/pullRequest/merge", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	out := make(map[string]pr.PullRequest)
	err = json.Unmarshal(respBody, &out)
	require.NoError(t, err)

	require.Equal(t, "pr1", out["pr"].ID)
}

func Test_MERGE_AGAIN_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()

	body := `{
		 "pull_request_id": "pr1"
	}`

	resp, err := http.Post(env.Server.URL+"/pullRequest/merge", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	fmt.Println(string(respBody))
	out := make(map[string]pr.PullRequest)
	err = json.Unmarshal(respBody, &out)
	require.NoError(t, err)
	require.Equal(t, "pr1", out["pr"].ID)
	require.Equal(t, "MERGED", out["pr"].Status)
}

func Test_DeactivateTeam_E2E(t *testing.T) {
	env, err := SetupTestEnv()
	require.NoError(t, err)
	defer env.Server.Close()

	body := `{"team_name": "payments2"}`

	resp, err := http.Post(env.Server.URL+"/team/deactivation", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}
