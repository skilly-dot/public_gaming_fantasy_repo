// internal/handlers/fifty_fifty_handler.go - WITH FULL DATA PUSH

package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminCreateFiftyFifty - Creates a new 50-50 bet
func (h *Handler) AdminCreateFiftyFifty(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    var req struct {
        Title       string  `json:"title"`
        Description string  `json:"description"`
        ExpiresAt   string  `json:"expires_at"`
        YesOdds     float64 `json:"yes_odds"`
        NoOdds      float64 `json:"no_odds"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request")
        return
    }
    
    if req.Title == "" {
        respondError(w, http.StatusBadRequest, "Title is required")
        return
    }
    
    // Default odds if not provided
    if req.YesOdds <= 0 {
        req.YesOdds = 1.85
    }
    if req.NoOdds <= 0 {
        req.NoOdds = 1.85
    }
    
    betID := "FF_" + uuid.New().String()[:12]
    
    var err error
    if req.ExpiresAt != "" {
        parsedTime, parseErr := time.Parse(time.RFC3339, req.ExpiresAt)
        if parseErr != nil {
            // Default 24 hours
            _, err = h.DB.Exec(`INSERT INTO fifty_fifty_bets (id, admin_id, title, description, yes_odds, no_odds, expires_at)
                              VALUES ($1, $2, $3, $4, $5, $6, NOW() + INTERVAL '24 hours')`,
                betID, user.ID, req.Title, req.Description, req.YesOdds, req.NoOdds)
        } else {
            _, err = h.DB.Exec(`INSERT INTO fifty_fifty_bets (id, admin_id, title, description, yes_odds, no_odds, expires_at)
                              VALUES ($1, $2, $3, $4, $5, $6, $7)`,
                betID, user.ID, req.Title, req.Description, req.YesOdds, req.NoOdds, parsedTime)
        }
    } else {
        _, err = h.DB.Exec(`INSERT INTO fifty_fifty_bets (id, admin_id, title, description, yes_odds, no_odds, expires_at)
                          VALUES ($1, $2, $3, $4, $5, $6, NOW() + INTERVAL '24 hours')`,
            betID, user.ID, req.Title, req.Description, req.YesOdds, req.NoOdds)
    }
    
    if err != nil {
        log.Printf("Failed to create 50-50 bet: %v", err)
        respondError(w, http.StatusInternalServerError, "Failed to create bet")
        return
    }
    
    // ===== NOTIFY ALL USERS - New 50-50 bet available =====
    go h.WSHub.SendToAll("fifty_fifty_created", map[string]interface{}{
        "bet_id":      betID,
        "title":       req.Title,
        "description": req.Description,
        "yes_odds":    req.YesOdds,
        "no_odds":     req.NoOdds,
        "message":     fmt.Sprintf("New 50-50 bet: %s", req.Title),
    })
    
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "status":   "created",
        "bet_id":   betID,
        "yes_odds": req.YesOdds,
        "no_odds":  req.NoOdds,
    })
}

// PUT /api/v1/admin/fifty-fifty/{betID}/lock
func (h *Handler) AdminLockFiftyFifty(w http.ResponseWriter, r *http.Request) {
    betID := chi.URLParam(r, "betID")
    
    var status string
    err := h.DB.QueryRow("SELECT status FROM fifty_fifty_bets WHERE id=$1", betID).Scan(&status)
    if err != nil {
        respondError(w, http.StatusNotFound, "Bet not found")
        return
    }
    if status != "OPEN" {
        respondError(w, http.StatusBadRequest, "Bet is " + status)
        return
    }
    
    h.DB.Exec("UPDATE fifty_fifty_bets SET status='LOCKED' WHERE id=$1", betID)
    
    // ===== NOTIFY ALL USERS - Bet locked =====
    go h.WSHub.SendToAll("fifty_fifty_locked", map[string]interface{}{
        "bet_id":  betID,
        "message": "A 50-50 bet has been locked",
    })
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "locked"})
}

// PUT /api/v1/admin/fifty-fifty/{betID}/settle
func (h *Handler) AdminSettleFiftyFifty(w http.ResponseWriter, r *http.Request) {
    betID := chi.URLParam(r, "betID")
    
    var req struct {
        Result string `json:"result"` // YES or NO
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    if req.Result != "YES" && req.Result != "NO" {
        respondError(w, http.StatusBadRequest, "Result must be YES or NO")
        return
    }
    
    // Update the proposition
    h.DB.Exec(`UPDATE fifty_fifty_bets 
               SET status='SETTLED', result=$1, settled_at=NOW() 
               WHERE id=$2`, req.Result, betID)
    
    // Settle all user bets - ONE click settles everything
    h.settleFiftyFiftyBets(betID, req.Result)
    
    // ===== NOTIFY ALL USERS - Bet settled =====
    go h.WSHub.SendToAll("fifty_fifty_settled", map[string]interface{}{
        "bet_id":  betID,
        "result":  req.Result,
        "message": fmt.Sprintf("50-50 bet settled: %s", req.Result),
    })
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "settled"})
}

// DELETE /api/v1/admin/fifty-fifty/{betID}
func (h *Handler) AdminDeleteFiftyFifty(w http.ResponseWriter, r *http.Request) {
    betID := chi.URLParam(r, "betID")
    
    h.DB.Exec(`UPDATE fifty_fifty_bets 
               SET status='DELETED', deleted_at=NOW() 
               WHERE id=$1 AND status != 'DELETED'`, betID)
    
    // ===== NOTIFY ALL USERS - Bet deleted =====
    go h.WSHub.SendToAll("fifty_fifty_deleted", map[string]interface{}{
        "bet_id":  betID,
        "message": "A 50-50 bet has been removed",
    })
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GET /api/v1/admin/fifty-fifty (Admin sees all their 50-50 bets)
func (h *Handler) AdminGetFiftyFifty(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT id, title, description, status, result, created_at, expires_at, settled_at
        FROM fifty_fifty_bets
        WHERE admin_id = $1 AND deleted_at IS NULL
        ORDER BY created_at DESC
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, title, desc, status, result string
        var createdAt, settledAt time.Time
        var expiresAt *time.Time
        rows.Scan(&id, &title, &desc, &status, &result, &createdAt, &expiresAt, &settledAt)
        
        bets = append(bets, map[string]interface{}{
            "id":          id,
            "title":       title,
            "description": desc,
            "status":      status,
            "result":      result,
            "created_at":  createdAt,
            "expires_at":  expiresAt,
            "settled_at":  settledAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// ===== USER ENDPOINTS =====

// GET /api/v1/bets/fifty-fifty-available (Users see available 50-50 bets)
func (h *Handler) GetFiftyFiftyForBetting(w http.ResponseWriter, r *http.Request) {
    rows, _ := h.DB.Query(`
        SELECT id, title, description, yes_odds, no_odds, status, expires_at
        FROM fifty_fifty_bets
        WHERE status = 'OPEN' AND deleted_at IS NULL
        ORDER BY created_at DESC
    `)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, title, desc, status string
        var yesOdds, noOdds float64
        var expiresAt *time.Time
        rows.Scan(&id, &title, &desc, &yesOdds, &noOdds, &status, &expiresAt)
        
        bets = append(bets, map[string]interface{}{
            "id":          id,
            "title":       title,
            "description": desc,
            "yes_odds":    yesOdds,
            "no_odds":     noOdds,
            "status":      status,
            "expires_at":  expiresAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// POST /api/v1/bets/fifty-fifty-place
func (h *Handler) PlaceFiftyFiftyBet(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    var req struct {
        FiftyFiftyID string  `json:"fifty_fifty_id"`
        Prediction   string  `json:"prediction"` // YES or NO
        Amount       float64 `json:"amount"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request")
        return
    }
    
    if req.Prediction != "YES" && req.Prediction != "NO" {
        respondError(w, http.StatusBadRequest, "Prediction must be YES or NO")
        return
    }
    
    // Check bet is OPEN
    var status string
    var yesOdds, noOdds float64
    err := h.DB.QueryRow(`SELECT status, yes_odds, no_odds FROM fifty_fifty_bets WHERE id=$1`, 
        req.FiftyFiftyID).Scan(&status, &yesOdds, &noOdds)
    if err != nil || status != "OPEN" {
        respondError(w, http.StatusBadRequest, "Bet is not open")
        return
    }
    
    // Get odds based on prediction
    odds := yesOdds
    if req.Prediction == "NO" {
        odds = noOdds
    }
    
    // Check wallet
    var kash float64
    h.DB.QueryRow("SELECT kash FROM wallets WHERE user_id=$1", user.ID).Scan(&kash)
    if kash < req.Amount {
        respondError(w, http.StatusBadRequest, fmt.Sprintf("Insufficient balance. You have KSh %.0f", kash))
        return
    }
    
    // Calculate potential win (12% tax)
    potentialWin := req.Amount * odds * 0.88
    
    // Deduct wallet
    h.DB.Exec("UPDATE wallets SET kash = kash - $1 WHERE user_id = $2", req.Amount, user.ID)
    
    // Place bet
    betID := "UFF_" + uuid.New().String()[:12]
    _, err = h.DB.Exec(`INSERT INTO user_fifty_fifty_bets 
        (id, user_id, fifty_fifty_id, prediction, odds, amount, potential_win)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
        betID, user.ID, req.FiftyFiftyID, req.Prediction, odds, req.Amount, potentialWin)
    
    if err != nil {
        // Refund
        h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", req.Amount, user.ID)
        respondError(w, http.StatusInternalServerError, "Failed to place bet")
        return
    }
    
    // ===== PUSH FULL USER STATE - 50-50 bet placed =====
    go h.PushUserState(user.ID, "fifty_fifty_bet_placed", map[string]interface{}{
        "bet_id":        betID,
        "fifty_fifty_id": req.FiftyFiftyID,
        "prediction":    req.Prediction,
        "odds":          odds,
        "potential_win": potentialWin,
        "message":       "50-50 bet placed successfully!",
    })
    
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "status":        "placed",
        "bet_id":        betID,
        "prediction":    req.Prediction,
        "odds":          odds,
        "potential_win": potentialWin,
    })
}

// GET /api/v1/bets/fifty-fifty-active
func (h *Handler) GetActiveFiftyFiftyBets(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT uff.id, ff.title, uff.prediction, uff.odds, uff.amount, uff.potential_win, uff.placed_at
        FROM user_fifty_fifty_bets uff
        JOIN fifty_fifty_bets ff ON uff.fifty_fifty_id = ff.id
        WHERE uff.user_id = $1 AND uff.status = 'PENDING'
        ORDER BY uff.placed_at DESC
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, title, prediction string
        var odds, amount, potentialWin float64
        var placedAt time.Time
        rows.Scan(&id, &title, &prediction, &odds, &amount, &potentialWin, &placedAt)
        
        bets = append(bets, map[string]interface{}{
            "id":            id,
            "title":         title,
            "prediction":    prediction,
            "odds":          odds,
            "amount":        amount,
            "potential_win": potentialWin,
            "placed_at":     placedAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// GET /api/v1/bets/fifty-fifty-history
func (h *Handler) GetFiftyFiftyHistory(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT uff.id, ff.title, uff.prediction, uff.odds, uff.amount, 
               uff.status, uff.payout, uff.settled_at
        FROM user_fifty_fifty_bets uff
        JOIN fifty_fifty_bets ff ON uff.fifty_fifty_id = ff.id
        WHERE uff.user_id = $1 AND uff.status != 'PENDING'
        ORDER BY uff.settled_at DESC
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, title, prediction, status string
        var odds, amount, payout float64
        var settledAt time.Time
        rows.Scan(&id, &title, &prediction, &odds, &amount, &status, &payout, &settledAt)
        
        bets = append(bets, map[string]interface{}{
            "id":          id,
            "title":       title,
            "prediction":  prediction,
            "odds":        odds,
            "amount":      amount,
            "status":      status,
            "payout":      payout,
            "settled_at":  settledAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// settleFiftyFiftyBets - Settles all bets on a 50-50 proposition
func (h *Handler) settleFiftyFiftyBets(betID string, result string) {
    // ONE query settles ALL bets on this proposition
    rows, _ := h.DB.Query(`
        SELECT id, user_id, prediction, amount, potential_win 
        FROM user_fifty_fifty_bets 
        WHERE fifty_fifty_id = $1 AND status = 'PENDING'
    `, betID)
    
    if rows == nil {
        return
    }
    defer rows.Close()
    
    for rows.Next() {
        var userBetID, userID, prediction string
        var amount, potentialWin float64
        rows.Scan(&userBetID, &userID, &prediction, &amount, &potentialWin)
        
        if prediction == result {
            // Winner - pay out the pre-calculated potential win
            h.DB.Exec(`UPDATE user_fifty_fifty_bets 
                       SET status='WON', payout=$1, settled_at=NOW() 
                       WHERE id=$2`, potentialWin, userBetID)
            h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", potentialWin, userID)
            
            // ===== PUSH FULL USER STATE - 50-50 bet won =====
            go h.PushUserState(userID, "fifty_fifty_won", map[string]interface{}{
                "bet_id":  userBetID,
                "payout":  potentialWin,
                "message": fmt.Sprintf("50-50 bet won! +$%.2f", potentialWin),
            })
        } else {
            // Loser
            h.DB.Exec(`UPDATE user_fifty_fifty_bets 
                       SET status='LOST', settled_at=NOW() 
                       WHERE id=$1`, userBetID)
            
            // ===== PUSH FULL USER STATE - 50-50 bet lost =====
            go h.PushUserState(userID, "fifty_fifty_lost", map[string]interface{}{
                "bet_id":  userBetID,
                "message": "50-50 bet lost",
            })
        }
    }
}