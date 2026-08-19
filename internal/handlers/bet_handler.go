package handlers

import (
	"encoding/json"
	"fmt"
	//"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) PlaceBet(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    var req struct {
        LeagueID string `json:"league_id"`
        Week     int    `json:"week"`
        Bets     []struct {
            FixtureID  string `json:"fixture_id"`
            Prediction string `json:"prediction"`
        } `json:"bets"`
        Amount   float64 `json:"amount"`
        IsCustom bool    `json:"is_custom"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // ===== CHECK IF WEEK HAS STARTED =====
    var weekStarted bool
    h.DB.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM daily_matches 
            WHERE league_id = $1 
            AND week_number = $2 
            AND status != 'SCHEDULED'
        )
    `, req.LeagueID, req.Week).Scan(&weekStarted)
    
    if weekStarted {
        respondError(w, http.StatusBadRequest, "Betting locked. Week already in progress.")
        return
    }

    if len(req.Bets) < 1 {
        respondError(w, http.StatusBadRequest, "Select at least one match")
        return
    }
    if req.Amount <= 0 {
        respondError(w, http.StatusBadRequest, "Invalid amount")
        return
    }

    tx, err := h.DB.Begin()
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Server error")
        return
    }
    defer tx.Rollback()

    var kash float64
    err = tx.QueryRow("SELECT kash FROM wallets WHERE user_id=$1", user.ID).Scan(&kash)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Wallet error")
        return
    }
    if kash < req.Amount {
        respondError(w, http.StatusBadRequest, "Insufficient balance")
        return
    }

    var totalOdds float64 = 1.0
    for _, bet := range req.Bets {
        var homeOdds, drawOdds, awayOdds float64
        err := tx.QueryRow("SELECT home_odds, draw_odds, away_odds FROM match_odds WHERE fixture_id=$1 AND league_id=$2",
            bet.FixtureID, req.LeagueID).Scan(&homeOdds, &drawOdds, &awayOdds)
        if err != nil {
            homeOdds, drawOdds, awayOdds = 1.85, 3.40, 4.20
        }
        switch bet.Prediction {
        case "HOME_WIN":
            totalOdds *= homeOdds
        case "DRAW":
            totalOdds *= drawOdds
        case "AWAY_WIN":
            totalOdds *= awayOdds
        }
    }

    _, err = tx.Exec("UPDATE wallets SET kash = kash - $1 WHERE user_id = $2", req.Amount, user.ID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to update wallet")
        return
    }

    betID := "BET_" + uuid.New().String()[:12]
    betsJSON, _ := json.Marshal(req.Bets)
    _, err = tx.Exec(`INSERT INTO bets (id, user_id, league_id, week, bets, amount, total_odds, is_custom, status, placed_at) 
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PENDING', $9)`,
        betID, user.ID, req.LeagueID, req.Week, string(betsJSON), req.Amount, totalOdds, req.IsCustom, time.Now())
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to save bet")
        return
    }

    tx.Exec("UPDATE wallets SET points = points + 5 WHERE user_id = $1", user.ID)

    err = tx.Commit()
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to place bet")
        return
    }

    var wallet struct{ Kash, Points, Coins float64 }
    h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", user.ID)

    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "bet_id": betID, "status": "placed",
        "total_odds": totalOdds, "potential": req.Amount * totalOdds,
        "wallet": wallet,
    })
}

// GetActiveBets - Returns PENDING and LOCKED bets
func (h *Handler) GetActiveBets(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT id, week, bets::text, amount, total_odds, is_custom, status, placed_at 
        FROM bets 
        WHERE user_id=$1 
        AND status IN ('PENDING', 'LOCKED')
        ORDER BY placed_at DESC
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, betsStr, status string
        var week int
        var amount, totalOdds float64
        var isCustom bool
        var placedAt time.Time
        rows.Scan(&id, &week, &betsStr, &amount, &totalOdds, &isCustom, &status, &placedAt)
        
        var betsData interface{}
        json.Unmarshal([]byte(betsStr), &betsData)
        
        bets = append(bets, map[string]interface{}{
            "id": id, "week": week, "bets": betsData, "amount": amount,
            "total_odds": totalOdds, "is_custom": isCustom, 
            "status": status, "placed_at": placedAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, bets)
}

// GetBetHistory - Returns WON and LOST bets
func (h *Handler) GetBetHistory(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT id, week, bets::text, amount, is_custom, status, payout, tax, points, coins, placed_at, settled_at
        FROM bets 
        WHERE user_id=$1 
        AND status IN ('WON', 'LOST')
        ORDER BY COALESCE(settled_at, placed_at) DESC 
        LIMIT 50
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, betsStr, status string
        var week int
        var amount, payout, tax float64
        var isCustom bool
        var points, coins int
        var placedAt time.Time
        var settledAt *time.Time
        
        rows.Scan(&id, &week, &betsStr, &amount, &isCustom, &status, &payout, &tax, &points, &coins, &placedAt, &settledAt)
        
        var betsData interface{}
        json.Unmarshal([]byte(betsStr), &betsData)
        
        bets = append(bets, map[string]interface{}{
            "id": id, "week": week, "bets": betsData, "amount": amount,
            "is_custom": isCustom, "status": status, "payout": payout, "tax": tax,
            "points": points, "coins": coins, "placed_at": placedAt, "settled_at": settledAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// SettleBets - Settles LOCKED bets
func (h *Handler) SettleBets(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    week := r.URL.Query().Get("week")

    rows, _ := h.DB.Query("SELECT id, user_id, bets::text, amount, is_custom FROM bets WHERE league_id=$1 AND week=$2 AND status='LOCKED'", leagueID, week)
    defer rows.Close()

    settled := 0
    for rows.Next() {
        var betID, userID, betsStr string
        var amount float64
        var isCustom bool
        rows.Scan(&betID, &userID, &betsStr, &amount, &isCustom)

        var bets []struct {
            FixtureID  string `json:"fixture_id"`
            Prediction string `json:"prediction"`
        }
        json.Unmarshal([]byte(betsStr), &bets)

        correct := 0
        total := len(bets)
        for _, bet := range bets {
            var winner string
            err := h.DB.QueryRow("SELECT winner FROM match_results WHERE league_id=$1 AND week_number=$2 AND fixture_id=$3",
                leagueID, week, bet.FixtureID).Scan(&winner)
            if err == nil {
                if (bet.Prediction == "HOME_WIN" && winner == "HOME") ||
                    (bet.Prediction == "DRAW" && winner == "DRAW") ||
                    (bet.Prediction == "AWAY_WIN" && winner == "AWAY") {
                    correct++
                }
            }
        }

        var status string
        var grossPayout, tax, netPayout float64
        var points, coins int
        const TAX_RATE = 0.12

        if isCustom {
            if correct == total {
                status = "WON"
                grossPayout = amount * 2.0
                tax = grossPayout * TAX_RATE
                netPayout = grossPayout - tax
                points = int(amount * 2)
                coins = int(amount * 0.5)
            } else {
                status = "LOST"
                points = 5
            }
        } else {
            losses := total - correct
            if losses <= 3 {
                status = "WON"
                grossPayout = amount * 1.5
                tax = grossPayout * TAX_RATE
                netPayout = grossPayout - tax
                points = int(amount * 1.5)
                coins = int(amount * 0.3)
            } else {
                status = "LOST"
                points = 10
            }
        }

        h.DB.Exec("UPDATE bets SET status=$1, payout=$2, tax=$3, points=$4, coins=$5, settled_at=$6 WHERE id=$7",
            status, netPayout, tax, points, coins, time.Now(), betID)

        if status == "WON" {
            h.DB.Exec("UPDATE wallets SET kash=kash+$1, points=points+$2, coins=coins+$3 WHERE user_id=$4",
                netPayout, points, coins, userID)
            
            h.WSHub.SendToUser(userID, "bet_won", map[string]interface{}{
                "bet_id": betID,
                "payout": netPayout,
                "message": fmt.Sprintf("You won KSh %.0f!", netPayout),
            })
        } else {
            h.DB.Exec("UPDATE wallets SET points=points+$1 WHERE user_id=$2", points, userID)
            
            h.WSHub.SendToUser(userID, "bet_lost", map[string]interface{}{
                "bet_id": betID,
                "message": "Better luck next time!",
            })
        }
        
        h.WSHub.SendToUser(userID, "wallet_update", map[string]interface{}{})
        settled++
    }

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "settled": settled, "week": week,
        "message": fmt.Sprintf("Settled %d bets for week %s", settled, week),
    })
}

func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    rows, _ := h.DB.Query("SELECT id, title, message, type, is_read, created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT 20", user.ID)
    defer rows.Close()
    var notifs []map[string]interface{}
    for rows.Next() {
        var id int
        var title, message, ntype string
        var isRead bool
        var createdAt time.Time
        rows.Scan(&id, &title, &message, &ntype, &isRead, &createdAt)
        notifs = append(notifs, map[string]interface{}{
            "id": id, "title": title, "message": message,
            "type": ntype, "is_read": isRead, "created_at": createdAt,
        })
    }
    if notifs == nil { notifs = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, notifs)
}