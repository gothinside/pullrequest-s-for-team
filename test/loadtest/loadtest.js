import http from "k6/http";
import { check, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

export const options = {
    stages: [
        { duration: "10s", target: 10 },   // ramp-up
        { duration: "30s", target: 50 },   // peak load
        { duration: "20s", target: 0 },    // ramp-down
    ],
    thresholds: {
        http_req_duration: ["p(95)<300"], // 95% requests < 300ms
        http_req_failed: ["rate<0.01"],   // <1% errors
    },
};

const BASE = "http://host.docker.internal:8080";

export default function () {
    // --- 1. Create team ---
    const teamName = "team_" + randomString(5);
    const payloadTeam = JSON.stringify({
        team_name: teamName,
        members: [
            { user_id: randomString(10), username: randomString(5), is_active: true },
            { user_id: randomString(10), username: randomString(5), is_active: true },
            { user_id: randomString(10), username: randomString(5), is_active: true }
        ],
    });

    let res = http.post(`${BASE}/team/add`, payloadTeam, {
        headers: { "Content-Type": "application/json" },
    });
    check(res, { "Team Add: 201": r => r.status === 201 });
    sleep(0.2);

    // --- 2. Get team ---
    res = http.get(`${BASE}/team/get?team_name=${teamName}`);
    check(res, { "Team Get: 200": r => r.status === 200 });
    sleep(0.2);


}
