-- internal/database/migrations/00002_top_scorers.sql
-- +goose Up
CREATE TABLE IF NOT EXISTS top_scorers (
    league_id TEXT NOT NULL,
    player_name TEXT NOT NULL,
    goals INTEGER DEFAULT 1,
    PRIMARY KEY (league_id, player_name)
);

CREATE INDEX IF NOT EXISTS idx_top_scorers_league ON top_scorers(league_id);
CREATE INDEX IF NOT EXISTS idx_top_scorers_goals ON top_scorers(league_id, goals DESC);

-- +goose Down
DROP TABLE IF EXISTS top_scorers;