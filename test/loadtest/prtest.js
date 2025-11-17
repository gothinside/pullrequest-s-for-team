import http from "k6/http";
import { check, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

export const options = {
    stages: [
        { duration: "10s", target: 10 },
        { duration: "20s", target: 50 },
        { duration: "2s", target: 200 },
    ],
    thresholds: {
        http_req_duration: ["p(95)<300"],
        http_req_failed: ["rate<0.01"],
    },
};

const BASE = "http://host.docker.internal:8080";

const predefinedTeams = [
    {
        team_name: "team_alpha",
        members: [
            { user_id: "uuuuuuuuuuuuuu1", username: "alice", is_active: true },
            { user_id: "u02", username: "bob", is_active: true },
        ],
    },
    {
        team_name: "team_beta",
        members: [
            { user_id: "u03", username: "carol", is_active: true },
            { user_id: "u04", username: "dave", is_active: true },
        ],
    },
];

export function setup() {
    for (const team of predefinedTeams) {
        const payloadTeam = JSON.stringify(team);
        let res = http.post(`${BASE}/team/add`, payloadTeam, {
            headers: { "Content-Type": "application/json" },
        });
        check(res, { [`Team Add ${team.team_name}: 201`]: r => r.status === 201 });
    }
    return { teams: predefinedTeams };
}

export default function () {
    // используем первую команду для PR
    const teamName = "team_" + randomString(5);
    const u1 = "u1" + randomString(10)
    const payloadTeam = JSON.stringify({
        team_name: teamName,
        members: [
            { user_id: u1, username: randomString(5), is_active: true },
            { user_id: randomString(10), username: randomString(5), is_active: true },
            { user_id: randomString(10), username: randomString(5), is_active: true }
        ],
    });

    let res1 = http.post(`${BASE}/team/add`, payloadTeam, {
        headers: { "Content-Type": "application/json" },
    });
    check(res1, { "Team Add: 201": r => r.status === 201 });
    sleep(0.2);
    
    // --- 2. Get team ---
    let res2 = http.get(`${BASE}/team/get?team_name=${teamName}`);
    check(res2, { "Team Get: 200": r => r.status === 200 });
    sleep(0.2);
    const payloadPR = JSON.stringify({
        pull_request_id: "pr_" + randomString(5),
        pull_request_name: "FixBug_" + randomString(5),
        author_id: u1,
    });

    let res3 = http.post(`${BASE}/pullRequest/create`, payloadPR, {
        headers: { "Content-Type": "application/json" },
    });
    check(res3, { "PR Create: 201": r => r.status === 201 });
    sleep(0.2);

    const team= JSON.stringify({
        team_name: teamName,
    });
    let res5 = http.post(`${BASE}/team/deactivation`, team, {
        headers: { "Content-Type": "application/json" },
    });
    check(res5, { "Team deactivation: 200": r => r.status === 200 });
}
