package middleware

import (
	"context"
	"net/http"
	//"time"

	// "strings"
	"github.com/betking/rich-backend/internal/database"
	"github.com/betking/rich-backend/internal/models"
)

type contextKey string
const UserContextKey contextKey = "user"

// internal/middleware/security.go

func AuthMiddleware(db *database.PostgresDB, redis *database.RedisDB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractToken(r)
            if token == "" {
                http.Error(w, `{"error":"Login required"}`, http.StatusUnauthorized)
                return
            }

            var userData struct {
                UserID   string `json:"user_id"`
                Username string `json:"username"`
                Gamename string `json:"gamename"`
                IsAdmin  bool   `json:"is_admin"`
            }

            // CHECK DATABASE FIRST - Always
            err := db.QueryRow(`
                SELECT u.id, u.username, u.gamename, u.is_admin 
                FROM sessions s 
                JOIN users u ON s.user_id = u.id 
                WHERE s.token = $1
            `, token).Scan(&userData.UserID, &userData.Username, &userData.Gamename, &userData.IsAdmin)
            
            if err != nil {
                http.Error(w, `{"error":"Invalid session"}`, http.StatusUnauthorized)
                return
            }
            
            // Cache for next time
            redis.CacheData("session:"+token, userData, 300)

            user := &models.User{
                ID:       userData.UserID,
                Username: userData.Username,
                Gamename: userData.Gamename,
                IsAdmin:  userData.IsAdmin,
            }
            ctx := context.WithValue(r.Context(), UserContextKey, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func RequireAdmin(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, ok := r.Context().Value(UserContextKey).(*models.User)
        if !ok || user == nil || !user.IsAdmin {
            http.Error(w, `{"error":"Admin access required"}`, http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}







