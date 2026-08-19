package handlers

import (
	"log"
	"net/http"
	// "time"
)

// internal/handlers/rankings_handler.go - WITH WINS/LOSSES

func (h *Handler) GetRankings(w http.ResponseWriter, r *http.Request) {
    
    rows, err := h.DB.Query(`
        SELECT 
            u.gamename,
            u.username,
            COALESCE(w.kash, 0) as kash,
            COALESCE(w.points, 0) as points,
            COALESCE(SUM(CASE WHEN b.status = 'WON' THEN 1 ELSE 0 END), 0) as wins,
            COALESCE(SUM(CASE WHEN b.status = 'LOST' THEN 1 ELSE 0 END), 0) as losses,
            COALESCE(SUM(b.payout), 0) as total_winnings,
            CASE WHEN COUNT(b.id) > 0 
                THEN ROUND(100.0 * SUM(CASE WHEN b.status = 'WON' THEN 1 ELSE 0 END) / COUNT(b.id), 1)
                ELSE 0 END as win_rate,
            ROW_NUMBER() OVER (ORDER BY COALESCE(w.kash, 0) DESC) as rank
        FROM users u
        LEFT JOIN wallets w ON u.id = w.user_id
        LEFT JOIN bets b ON u.id = b.user_id
        GROUP BY u.id, u.gamename, u.username, w.kash, w.points
        ORDER BY COALESCE(w.kash, 0) DESC
        LIMIT 50
    `)
    
    if err != nil {
        log.Printf("Rankings query error: %v", err)
        respondJSON(w, http.StatusOK, []interface{}{})
        return
    }
    defer rows.Close()
    
    var rankings []map[string]interface{}
    for rows.Next() {
        var gamename, username string
        var kash, points, totalWinnings float64
        var wins, losses, rank int
        var winRate float64
        
        if err := rows.Scan(&gamename, &username, &kash, &points, &wins, &losses, &totalWinnings, &winRate, &rank); err != nil {
            log.Printf("Scan error: %v", err)
            continue
        }
        
        rankings = append(rankings, map[string]interface{}{
            "gamename":       gamename,
            "username":       username,
            "kash":           kash,
            "points":         points,
            "wins":           wins,
            "losses":         losses,
            "win_rate":       winRate,
            "total_winnings": totalWinnings,
            "rank":           rank,
        })
    }
    
    if rankings == nil {
        rankings = []map[string]interface{}{}
    }
    
    h.WSHub.SendToAll("rankings_update", rankings)
    
    respondJSON(w, http.StatusOK, rankings)
}
// GetUserRank - User's global rank
func (h *Handler) GetUserRank(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    type RankInfo struct {
        Rank          int     `json:"rank"`
        TotalWinnings float64 `json:"total_winnings"`
        Wins          int     `json:"wins"`
        Losses        int     `json:"losses"`
        WinRate       float64 `json:"win_rate"`
    }
    
    var rank RankInfo
    
    h.DB.Get(&rank, `
        SELECT rank, total_winnings, wins, losses, win_rate FROM (
            SELECT u.id,
                   w.kash as total_winnings,
                   COUNT(CASE WHEN b.status='WON' THEN 1 END) as wins,
                   COUNT(CASE WHEN b.status='LOST' THEN 1 END) as losses,
                   CASE WHEN COUNT(b.id) > 0 
                        THEN ROUND(100.0 * COUNT(CASE WHEN b.status='WON' THEN 1 END) / COUNT(b.id), 1)
                        ELSE 0 END as win_rate,
                   ROW_NUMBER() OVER (ORDER BY w.kash DESC) as rank
            FROM wallets w 
            JOIN users u ON w.user_id = u.id
            LEFT JOIN bets b ON u.id = b.user_id
            GROUP BY u.id, w.kash
        ) sub WHERE id = $1
    `, user.ID)
    
    respondJSON(w, http.StatusOK, rank)
}