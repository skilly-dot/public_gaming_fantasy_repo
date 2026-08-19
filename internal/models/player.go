// internal/models/player.go
package models

import "time"

// Basic player (Postgres)
type Player struct {
    ID        string    `json:"id" db:"id"`
    TeamID    string    `json:"team_id" db:"team_id"`
    Name      string    `json:"name" db:"name"`
    Position  string    `json:"position" db:"position"`
    Number    int       `json:"number" db:"number"`
    Rating    float64   `json:"rating" db:"rating"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Rich player data (MongoDB)
type PlayerDetail struct {
    PlayerID    string      `json:"player_id" bson:"player_id"`
    Age         int         `json:"age" bson:"age"`
    Nationality string      `json:"nationality" bson:"nationality"`
    Height      int         `json:"height" bson:"height"`
    Weight      int         `json:"weight" bson:"weight"`
    StrongFoot  string      `json:"strong_foot" bson:"strong_foot"`
    Stats       PlayerStats `json:"stats" bson:"stats"`
    Traits      []string    `json:"traits" bson:"traits"`
    Form        float64     `json:"form" bson:"form"`
    Injured     bool        `json:"injured" bson:"injured"`
    Suspended   bool        `json:"suspended" bson:"suspended"`
    ImageURL    string      `json:"image_url" bson:"image_url"`
}

type PlayerStats struct {
    Pace        int `json:"pace" bson:"pace"`
    Shooting    int `json:"shooting" bson:"shooting"`
    Passing     int `json:"passing" bson:"passing"`
    Dribbling   int `json:"dribbling" bson:"dribbling"`
    Defending   int `json:"defending" bson:"defending"`
    Physical    int `json:"physical" bson:"physical"`
    Stamina     int `json:"stamina" bson:"stamina"`
    Aggression  int `json:"aggression" bson:"aggression"`
    Composure   int `json:"composure" bson:"composure"`
    Vision      int `json:"vision" bson:"vision"`
    Crossing    int `json:"crossing" bson:"crossing"`
    Finishing   int `json:"finishing" bson:"finishing"`
    LongShots   int `json:"long_shots" bson:"long_shots"`
    Penalties   int `json:"penalties" bson:"penalties"`
    FreeKicks   int `json:"free_kicks" bson:"free_kicks"`
    Heading     int `json:"heading" bson:"heading"`
    Tackling    int `json:"tackling" bson:"tackling"`
    Positioning int `json:"positioning" bson:"positioning"`
    Reflexes    int `json:"reflexes" bson:"reflexes"`
    Diving      int `json:"diving" bson:"diving"`
    Handling    int `json:"handling" bson:"handling"`
}