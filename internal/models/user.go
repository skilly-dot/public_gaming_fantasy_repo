package models

import "time"

type User struct {
    ID           string    `json:"id" db:"id"`
    Username     string    `json:"username" db:"username"`
    PasswordHash string    `json:"-" db:"password_hash"`
    Gamename     string    `json:"gamename" db:"gamename"`
    IsAdmin      bool      `json:"is_admin" db:"is_admin"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
