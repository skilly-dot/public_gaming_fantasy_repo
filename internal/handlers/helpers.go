// internal/handlers/helpers.go
package handlers

// internal/handlers/handler.go (or helpers.go)

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/betking/rich-backend/internal/middleware" // ← Import middleware
	"github.com/betking/rich-backend/internal/models"
)

//type contextKey string

//const userContextKey contextKey = "user"

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
    respondJSON(w, status, map[string]string{"error": message})
}

// Use the SAME key as middleware
func getUserFromContext(ctx context.Context) *models.User {
    user, ok := ctx.Value(middleware.UserContextKey).(*models.User)  // ← Use middleware.UserContextKey
    if !ok {
        return nil
    }
    return user
}

func sanitizeError(err error) string {
    msg := err.Error()
    if strings.Contains(msg, "no rows") || strings.Contains(msg, "sql: no rows") {
        return "Not found"
    }
    if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") {
        return "Already exists"
    }
    // Log real error but return generic message
    log.Printf("Internal error: %v", err)
    return "Something went wrong"
}
