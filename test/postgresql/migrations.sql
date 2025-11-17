DROP TABLE IF EXISTS userspr CASCADE;
DROP TABLE IF EXISTS usershistory CASCADE;
DROP TABLE IF EXISTS pr CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS teams CASCADE;

CREATE TABLE teams(
    id SERIAL PRIMARY KEY,
    team_name VARCHAR(128) UNIQUE
);

CREATE TABLE users (
    id VARCHAR(256) PRIMARY KEY,
    username VARCHAR(2000) UNIQUE NOT NULL,
    team_id INTEGER REFERENCES teams(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL
);

CREATE TABLE pr (
    id VARCHAR(256) PRIMARY KEY,
    pr_name varchar(2000),
    author_id VARCHAR(256) NOT NULL REFERENCES users(id),
    pr_status VARCHAR(256),
    created_ad TIMESTAMP,
    mergerd_at TIMESTAMP
);

CREATE TABLE userspr (
    user_id    VARCHAR(256) NOT NULL REFERENCES users(id),
    request_id VARCHAR(256) NOT NULL REFERENCES pr(id),
    PRIMARY KEY (user_id, request_id)
);

CREATE TABLE usershistory(
    user_id VARCHAR(256) NOT NULL REFERENCES users(id),
    pr_count INTEGER,
    PRIMARY KEY (user_id)
);

CREATE INDEX idx_pr_author_id ON pr(author_id);
CREATE INDEX idx_users_team_id ON users(team_id);
CREATE INDEX idx_teams_team_name ON teams(team_name);