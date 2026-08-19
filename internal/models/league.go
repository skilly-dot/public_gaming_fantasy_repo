// internal/models/league.go
package models

import "time"

type League struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Type        string    `json:"type" db:"type"`
    Status      string    `json:"status" db:"status"`
    CurrentWeek int       `json:"current_week" db:"current_week"`
    TotalWeeks  int       `json:"total_weeks" db:"total_weeks"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UserLeague struct {
    UserID   string    `json:"user_id" db:"user_id"`
    LeagueID string    `json:"league_id" db:"league_id"`
    JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}

type Fixture struct {
    ID          int    `json:"id" db:"id"`
    LeagueID    string `json:"league_id" db:"league_id"`
    WeekNumber  int    `json:"week_number" db:"week_number"`
    HomeTeamID  string `json:"home_team_id" db:"home_team_id"`
    AwayTeamID  string `json:"away_team_id" db:"away_team_id"`
}

type Team struct {
    ID       string    `json:"id" db:"id"`
    LeagueID string    `json:"league_id" db:"league_id"`
    Name     string    `json:"name" db:"name"`
    Rating   float64   `json:"rating" db:"rating"`
    CoachID  string    `json:"coach_id" db:"coach_id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}