package team_test

import (
	_ "github.com/lib/pq"
)

// var testDB *sql.DB

// func TestMain(m *testing.M) {
// 	var err error
// 	testDB, err = sql.Open("postgres", "host=localhost port=5429 user=postgres password=123 dbname=tr sslmode=disable")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer testDB.Close()

// 	// Clean DB
// 	if err := cleanDB(testDB); err != nil {
// 		panic(err)
// 	}

// 	os.Exit(m.Run())
// }

// func cleanDB(db *sql.DB) error {
// 	_, err := db.Exec(`
// 		TRUNCATE TABLE userspr, pr, users, teams RESTART IDENTITY CASCADE;
// 	`)
// 	return err
// }

// func TestTeamRouterIntegration_HandleAddTeam(t *testing.T) {
// 	ur := &user.UserRepo{DB: testDB}
// 	tr := &team.TeamRepo{DB: testDB, UR: ur}
// 	router := &team.TeamRouter{TR: tr}

// 	input := team.TeamInput{
// 		TeamName: "IntegrationTeam2",
// 		Members: []team.TeamMember{
// 			{UserID: "u1", Username: "Alice", IsActive: true},
// 			{UserID: "u2", Username: "Bob", IsActive: true},
// 		},
// 	}

// 	body, _ := json.Marshal(input)
// 	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
// 	w := httptest.NewRecorder()

// 	router.HandleAddTeam(w, req)

// 	if w.Code != http.StatusCreated {
// 		t.Fatalf("expected 201 Created, got %d, body: %s", w.Code, w.Body.String())
// 	}

// 	// Validate team exists in DB
// 	rows, err := testDB.QueryContext(context.Background(), "SELECT id FROM teams")
// 	var flag bool
// 	if err != nil {
// 		t.Fatalf("team not found in DB: %v", err)
// 	}
// 	for rows.Next() {
// 		flag = true
// 	}
// 	if !flag {
// 		t.Fatalf("team not found in DB: %v", err)
// 	}

// }

// func TestTeamRouterIntegration_GetTeamWithMembersHandler(t *testing.T) {
// 	ur := &user.UserRepo{DB: testDB}
// 	tr := &team.TeamRepo{DB: testDB, UR: ur}
// 	router := &team.TeamRouter{TR: tr}

// 	req := httptest.NewRequest(http.MethodGet, "/?team_name=IntegrationTeam", nil)
// 	w := httptest.NewRecorder()

// 	router.GetTeamWithMembersHandler(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
// 	}

// 	var resp map[string]interface{}
// 	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
// 		t.Fatalf("failed to parse response: %v", err)
// 	}

// 	teamData := resp["team"].(map[string]interface{})
// 	if teamData["team_name"] != "IntegrationTeam" {
// 		t.Fatalf("expected team_name IntegrationTeam, got %v", teamData["team_name"])
// 	}

// 	members := teamData["members"].([]interface{})
// 	if len(members) != 2 {
// 		t.Fatalf("expected 2 members, got %d", len(members))
// 	}
// }
