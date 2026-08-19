package handlers

import (
	"encoding/json"
	//"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// internal/handlers/auth_handler.go - UPDATED

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
        Gamename string `json:"gamename"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    if req.Username == "" || req.Password == "" {
        respondError(w, http.StatusBadRequest, "Username and password required")
        return
    }
    
    // Validate username
    if len(req.Username) < 3 || len(req.Username) > 20 {
        respondError(w, http.StatusBadRequest, "Username must be 3-20 characters")
        return
    }
    
    // Validate password
    if len(req.Password) < 4 {
        respondError(w, http.StatusBadRequest, "Password must be at least 4 characters")
        return
    }
    
    var exists bool
    h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)", req.Username).Scan(&exists)
    if exists {
        respondError(w, http.StatusConflict, "Username already taken")
        return
    }
    
    hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    userID := "USER_" + uuid.New().String()[:12]
    
    h.DB.Exec("INSERT INTO users (id, username, password_hash, gamename) VALUES ($1,$2,$3,$4)", 
        userID, req.Username, string(hash), req.Gamename)
    h.DB.Exec("INSERT INTO wallets (user_id, kash, points, coins) VALUES ($1,1000,100,10)", userID)
    
    // Create session with 7 day expiry
    sessionID := uuid.New().String()
    h.DB.Exec(`INSERT INTO sessions (id, user_id, token, expires_at) 
               VALUES ($1,$2,$3, NOW() + INTERVAL '7 days')`, 
        sessionID, userID, sessionID)
    
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "session_id": sessionID,
        "user_id":    userID,
        "username":   req.Username,
        "gamename":   req.Gamename,
        "is_admin":   false,
        "expires_in": "7 days",
    })
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    var user struct {
        ID, Username, PasswordHash, Gamename string
        IsAdmin bool
    }
    
    err := h.DB.QueryRow("SELECT id, username, password_hash, gamename, is_admin FROM users WHERE username=$1", 
        req.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Gamename, &user.IsAdmin)
    
    if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
        respondError(w, http.StatusUnauthorized, "Invalid username or password")
        return
    }
    
    // Create new session with 7 day expiry
    sessionID := uuid.New().String()
    h.DB.Exec(`INSERT INTO sessions (id, user_id, token, expires_at) 
               VALUES ($1,$2,$3, NOW() + INTERVAL '7 days')`, 
        sessionID, user.ID, sessionID)
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "session_id": sessionID,
        "user_id":    user.ID,
        "username":   user.Username,
        "gamename":   user.Gamename,
        "is_admin":   user.IsAdmin,
        "expires_in": "7 days",
    })
}
// internal/handlers/auth.go - FIXED LOGOUT

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
    // Extract token
    token := extractTokenFromRequest(r)
    
    if token == "" {
        respondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
        return
    }
    
    // Find user from token in DB
    var userID string
    err := h.DB.QueryRow("SELECT user_id FROM sessions WHERE token = $1", token).Scan(&userID)
    
    if err != nil {
        respondJSON(w, http.StatusOK, map[string]string{
            "status": "logged_out",
            "message": "Session not found",
        })
        return
    }
    
    // Delete ALL sessions for this user
    h.DB.Exec("DELETE FROM sessions WHERE user_id = $1", userID)
    
    // Clear Redis cache
    if h.Redis != nil && h.Redis.Client != nil {
        h.Redis.DeleteCache("session:" + token)
        // Also delete ALL session keys for this user
        h.Redis.Client.Del(h.Redis.Ctx, "session:"+token)
    }
    
    respondJSON(w, http.StatusOK, map[string]string{
        "status": "logged_out",
        "message": "All sessions invalidated",
    })
}


func extractTokenFromRequest(r *http.Request) string {
    bearer := r.Header.Get("Authorization")
    if strings.HasPrefix(bearer, "Bearer ") {
        return strings.TrimPrefix(bearer, "Bearer ")
    }
    return ""
}




func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    if user == nil {
        respondError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    // Delete user's bets
    h.DB.Exec("DELETE FROM bets WHERE user_id=$1", user.ID)
    h.DB.Exec("DELETE FROM user_admin_bets WHERE user_id=$1", user.ID)
    h.DB.Exec("DELETE FROM user_admin_match_bets WHERE user_id=$1", user.ID)
    h.DB.Exec("DELETE FROM user_fifty_fifty_bets WHERE user_id=$1", user.ID)
    h.DB.Exec("DELETE FROM quick_matches WHERE user_id=$1", user.ID)
    
    // Delete user's leagues
    rows, _ := h.DB.Query("SELECT id FROM leagues WHERE user_id=$1", user.ID)
    for rows.Next() {
        var leagueID string
        rows.Scan(&leagueID)
        h.deleteLeagueData(leagueID)
    }
    rows.Close()
    
    // Delete notifications
    h.DB.Exec("DELETE FROM notifications WHERE user_id=$1", user.ID)
    
    // Delete wallet
    h.DB.Exec("DELETE FROM wallets WHERE user_id=$1", user.ID)
    
    // Delete sessions
    h.DB.Exec("DELETE FROM sessions WHERE user_id=$1", user.ID)
    
    // ===== DELETE THE USER - THIS WAS MISSING =====
    result, err := h.DB.Exec("DELETE FROM users WHERE id=$1", user.ID)
    
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to delete user")
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        respondError(w, http.StatusInternalServerError, "User not found")
        return
    }
    
    respondJSON(w, http.StatusOK, map[string]string{
        "status": "deleted",
        "message": "Account and all data permanently deleted",
    })
}