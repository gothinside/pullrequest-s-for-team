CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    team_name VARCHAR(128) UNIQUE
);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(256) PRIMARY KEY,
    username VARCHAR(2000) UNIQUE NOT NULL,
    team_id INTEGER REFERENCES teams(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS pr (
    id VARCHAR(256) PRIMARY KEY,
    pr_name VARCHAR(2000),
    author_id VARCHAR(256) NOT NULL REFERENCES users(id),
    pr_status VARCHAR(256),
    created_at TIMESTAMP,
    merged_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS userspr (
    user_id    VARCHAR(256) NOT NULL REFERENCES users(id),
    request_id VARCHAR(256) NOT NULL REFERENCES pr(id), 
    PRIMARY KEY (user_id, request_id) 
);

CREATE TABLE IF NOT EXISTS usershistory (
    user_id VARCHAR(256) NOT NULL REFERENCES users(id),
    pr_count INTEGER,
    PRIMARY KEY (user_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_author_id ON pr(author_id);
CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);
CREATE INDEX IF NOT EXISTS idx_teams_team_name ON teams(team_name);
