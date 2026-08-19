// internal/models/bet.go
package models

import "time"

type Bet struct {
    ID        string    `json:"id" db:"id"`
    UserID    string    `json:"user_id" db:"user_id"`
    MatchID   string    `json:"match_id" db:"match_id"`
    BetType   string    `json:"bet_type" db:"bet_type"`
    Amount    float64   `json:"amount" db:"amount"`
    Odds      float64   `json:"odds" db:"odds"`
    Status    string    `json:"status" db:"status"`
    Payout    float64   `json:"payout" db:"payout"`
    Points    int       `json:"points" db:"points"`
    Week      int       `json:"week" db:"week"`
    IsCustom  bool      `json:"is_custom" db:"is_custom"`
    PlacedAt  time.Time `json:"placed_at" db:"placed_at"`
}

// Full league data response
type FullLeagueData struct {
    League    League        `json:"league"`
    Teams     []Team        `json:"teams"`
    Players   []Player      `json:"players"`
    Fixtures  []Fixture     `json:"fixtures"`
    Details   LeagueDetails `json:"details"`
}

type LeagueDetails struct {
    Coaches      []CoachDetail  `json:"coaches"`
    PlayerStats  []PlayerDetail `json:"player_stats"`
    TeamDetails  []TeamDetail   `json:"team_details"`
}