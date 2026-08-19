// internal/models/team.go
package models

type TeamDetail struct {
    TeamID        string    `json:"team_id" bson:"team_id"`
    Formation     string    `json:"formation" bson:"formation"`
    CaptainID     string    `json:"captain_id" bson:"captain_id"`
    ViceCaptain   string    `json:"vice_captain" bson:"vice_captain"`
    PenaltyTaker  string    `json:"penalty_taker" bson:"penalty_taker"`
    FreeKickTaker string    `json:"free_kick_taker" bson:"free_kick_taker"`
    CornerTaker   string    `json:"corner_taker" bson:"corner_taker"`
    Stats         TeamStats `json:"stats" bson:"stats"`
    ImageURL      string    `json:"image_url" bson:"image_url"`
}

type TeamStats struct {
    Attack     int `json:"attack" bson:"attack"`
    Midfield   int `json:"midfield" bson:"midfield"`
    Defense    int `json:"defense" bson:"defense"`
    Teamwork   int `json:"teamwork" bson:"teamwork"`
    Experience int `json:"experience" bson:"experience"`
    SetPieces  int `json:"set_pieces" bson:"set_pieces"`
}