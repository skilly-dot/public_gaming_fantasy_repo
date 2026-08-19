// internal/handlers/admin_handler.go - CLEAN VERSION (Users + Leagues + Wallets)
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// ============================================
// USER MANAGEMENT
// ============================================

// GET /api/v1/admin/users
func (h *Handler) AdminGetUsers(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.Query(`
		SELECT u.id, u.username, u.gamename, u.is_admin, u.created_at,
			   w.kash, w.points, w.coins,
			   COALESCE(bs.total_bets, 0) as total_bets,
			   COALESCE(bs.wins, 0) as wins
		FROM users u
		LEFT JOIN wallets w ON u.id = w.user_id
		LEFT JOIN (
			SELECT user_id,
				COUNT(*) as total_bets,
				COUNT(*) FILTER (WHERE status = 'WON') as wins
			FROM bets GROUP BY user_id
		) bs ON u.id = bs.user_id
		ORDER BY u.created_at DESC
	`)
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, username, gamename string
		var isAdmin bool
		var createdAt time.Time
		var kash, points, coins float64
		var totalBets, wins int
		rows.Scan(&id, &username, &gamename, &isAdmin, &createdAt,
			&kash, &points, &coins, &totalBets, &wins)

		users = append(users, map[string]interface{}{
			"id":         id,
			"username":   username,
			"gamename":   gamename,
			"is_admin":   isAdmin,
			"created_at": createdAt,
			"wallet": map[string]float64{
				"kash":   kash,
				"points": points,
				"coins":  coins,
			},
			"stats": map[string]interface{}{
				"total_bets": totalBets,
				"wins":       wins,
				"win_rate": func() float64 {
					if totalBets == 0 {
						return 0
					}
					return float64(wins) / float64(totalBets) * 100
				}(),
			},
		})
	}
	if users == nil {
		users = []map[string]interface{}{}
	}
	respondJSON(w, http.StatusOK, users)
}

// GET /api/v1/admin/users/{userID}
func (h *Handler) AdminGetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")

	var user struct {
		ID, Username, Gamename string
		IsAdmin                bool
		CreatedAt              time.Time
	}
	err := h.DB.Get(&user, "SELECT id, username, gamename, is_admin, created_at FROM users WHERE id=$1", userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// PUT /api/v1/admin/users/{userID}
func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req struct {
		Gamename string `json:"gamename"`
		IsAdmin  bool   `json:"is_admin"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.DB.Exec("UPDATE users SET gamename=$1, is_admin=$2 WHERE id=$3", req.Gamename, req.IsAdmin, userID)
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// PUT /api/v1/admin/users/{userID}/full (Update user + wallet in one call)
func (h *Handler) AdminUpdateUserFull(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req struct {
		Gamename string  `json:"gamename"`
		IsAdmin  bool    `json:"is_admin"`
		Kash     float64 `json:"kash"`
		Points   float64 `json:"points"`
		Coins    float64 `json:"coins"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Gamename != "" {
		h.DB.Exec("UPDATE users SET gamename=$1, is_admin=$2 WHERE id=$3", req.Gamename, req.IsAdmin, userID)
	}
	h.DB.Exec("UPDATE wallets SET kash=$1, points=$2, coins=$3 WHERE user_id=$4",
		req.Kash, req.Points, req.Coins, userID)

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DELETE /api/v1/admin/users/{userID}
func (h *Handler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")

	h.DB.Exec("DELETE FROM user_fifty_fifty_bets WHERE user_id=$1", userID)
	h.DB.Exec("DELETE FROM user_admin_match_bets WHERE user_id=$1", userID)
	h.DB.Exec("DELETE FROM user_admin_bets WHERE user_id=$1", userID)
	h.DB.Exec("DELETE FROM bets WHERE user_id=$1", userID)
	h.DB.Exec("DELETE FROM notifications WHERE user_id=$1", userID)
	h.DB.Exec("DELETE FROM wallets WHERE user_id=$1", userID)
	h.DB.Exec("DELETE FROM sessions WHERE user_id=$1", userID)
	h.DB.Exec("DELETE FROM users WHERE id=$1", userID)

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ============================================
// LEAGUE MANAGEMENT
// ============================================

// GET /api/v1/admin/leagues
func (h *Handler) AdminGetLeagues(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.Query(`
		SELECT l.id, l.name, l.type, l.status, l.day_number, l.total_weeks,
			   u.username, u.gamename,
			   (SELECT COUNT(*) FROM daily_matches WHERE league_id=l.id AND status='COMPLETED') as completed,
			   (SELECT COUNT(*) FROM daily_matches WHERE league_id=l.id) as total_matches
		FROM leagues l
		JOIN users u ON l.user_id = u.id
		ORDER BY l.created_at DESC
	`)
	defer rows.Close()

	var leagues []map[string]interface{}
	for rows.Next() {
		var id, name, ltype, status, username, gamename string
		var day, weeks, completed, total int
		rows.Scan(&id, &name, &ltype, &status, &day, &weeks, &username, &gamename, &completed, &total)
		leagues = append(leagues, map[string]interface{}{
			"id":          id,
			"name":        name,
			"type":        ltype,
			"status":      status,
			"day":         day,
			"total_weeks": weeks,
			"owner": map[string]string{
				"username": username,
				"gamename": gamename,
			},
			"progress": map[string]int{
				"completed": completed,
				"total":     total,
			},
		})
	}
	if leagues == nil {
		leagues = []map[string]interface{}{}
	}
	respondJSON(w, http.StatusOK, leagues)
}

// PUT /api/v1/admin/leagues/{leagueID}
func (h *Handler) AdminUpdateLeague(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	var req struct {
		Status    string `json:"status"`
		DayNumber int    `json:"day_number"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Status != "" {
		h.DB.Exec("UPDATE leagues SET status=$1 WHERE id=$2", req.Status, leagueID)
	}
	if req.DayNumber > 0 {
		h.DB.Exec("UPDATE leagues SET day_number=$1 WHERE id=$2", req.DayNumber, leagueID)
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DELETE /api/v1/admin/leagues/{leagueID}
func (h *Handler) AdminDeleteLeague(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")

	h.DB.Exec("DELETE FROM daily_matches WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM match_results WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM match_odds WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM league_table WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM bets WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM players WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM coaches WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM fixtures WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM teams WHERE league_id=$1", leagueID)
	h.DB.Exec("DELETE FROM leagues WHERE id=$1", leagueID)

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ============================================
// WALLET MANAGEMENT
// ============================================

// PUT /api/v1/admin/wallets/{userID}
func (h *Handler) AdminUpdateWallet(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req struct {
		Kash   float64 `json:"kash"`
		Points float64 `json:"points"`
		Coins  float64 `json:"coins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate amounts
	if req.Kash < 0 || req.Points < 0 || req.Coins < 0 {
		respondError(w, http.StatusBadRequest, "Amounts cannot be negative")
		return
	}

	result, err := h.DB.Exec("UPDATE wallets SET kash=$1, points=$2, coins=$3 WHERE user_id=$4",
		req.Kash, req.Points, req.Coins, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update wallet")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "User wallet not found")
		return
	}

	// Return updated wallet
	var wallet struct {
		Kash   float64 `db:"kash"`
		Points float64 `db:"points"`
		Coins  float64 `db:"coins"`
	}
	h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", userID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "updated",
		"wallet": wallet,
		"message": fmt.Sprintf("Wallet updated: KSh %.0f, %.0f pts, %.0f coins",
			wallet.Kash, wallet.Points, wallet.Coins),
	})
}

// POST /api/v1/admin/wallets/{userID}/add (Add to existing balance)
func (h *Handler) AdminAddToWallet(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req struct {
		Kash   float64 `json:"kash"`
		Points float64 `json:"points"`
		Coins  float64 `json:"coins"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.DB.Exec("UPDATE wallets SET kash=kash+$1, points=points+$2, coins=coins+$3 WHERE user_id=$4",
		req.Kash, req.Points, req.Coins, userID)

	var wallet struct {
		Kash   float64 `db:"kash"`
		Points float64 `db:"points"`
		Coins  float64 `db:"coins"`
	}
	h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", userID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "added",
		"wallet":  wallet,
		"message": fmt.Sprintf("Added KSh %.0f to wallet", req.Kash),
	})
}