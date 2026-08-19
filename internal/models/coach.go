// internal/models/coach.go
package models

type CoachDetail struct {
    CoachID        string     `json:"coach_id" bson:"coach_id"`
    Name           string     `json:"name" bson:"name"`
    Nationality    string     `json:"nationality" bson:"nationality"`
    Age            int        `json:"age" bson:"age"`
    Rating         float64    `json:"rating" bson:"rating"`
    Formation      string     `json:"formation" bson:"formation"`
    PlayStyle      string     `json:"playstyle" bson:"playstyle"`
    SecondaryStyle string     `json:"secondary_style" bson:"secondary_style"`
    Mentality      string     `json:"mentality" bson:"mentality"`
    Stats          CoachStats `json:"stats" bson:"stats"`
    Abilities      []string   `json:"abilities" bson:"abilities"`
    ImageURL       string     `json:"image_url" bson:"image_url"`
}

type CoachStats struct {
    Attacking    int `json:"attacking" bson:"attacking"`
    Defending    int `json:"defending" bson:"defending"`
    Motivation   int `json:"motivation" bson:"motivation"`
    Tactical     int `json:"tactical" bson:"tactical"`
    YouthDev     int `json:"youth_dev" bson:"youth_dev"`
    Discipline   int `json:"discipline" bson:"discipline"`
    Adaptability int `json:"adaptability" bson:"adaptability"`
}