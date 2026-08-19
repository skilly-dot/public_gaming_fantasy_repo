// internal/handlers/admin_match_handler.go
package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
)

// ===== ADMIN ENDPOINTS =====

// POST /api/v1/admin/match-bets/create
func (h *Handler) AdminCreateMatchBet(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    var req struct {
        Title       string `json:"title"`
        Description string `json:"description"`
        Matches     []struct {
            HomeTeam string  `json:"home_team"`
            AwayTeam string  `json:"away_team"`
            HomeOdds float64 `json:"home_odds"`
            DrawOdds float64 `json:"draw_odds"`
            AwayOdds float64 `json:"away_odds"`
        } `json:"matches"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    if len(req.Matches) == 0 {
        respondError(w, http.StatusBadRequest, "At least 1 match required")
        return
    }
    
    if len(req.Matches) > 30 {
        respondError(w, http.StatusBadRequest, "Maximum 30 matches allowed")
        return
    }
    
    // Create parent bet
    betID := "ADMINBET_" + uuid.New().String()[:12]
    _, err := h.DB.Exec(`INSERT INTO admin_match_bets (id, admin_id, title, description, status)
                          VALUES ($1, $2, $3, $4, 'OPEN')`,
        betID, user.ID, req.Title, req.Description)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to create bet")
        return
    }
    
    // Create matches
    var matchIDs []string
    for i, m := range req.Matches {
        matchID := "ADMINMATCH_" + uuid.New().String()[:12]
        _, err := h.DB.Exec(`INSERT INTO admin_matches 
            (id, admin_bet_id, match_index, home_team, away_team, home_odds, draw_odds, away_odds)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
            matchID, betID, i+1, m.HomeTeam, m.AwayTeam, m.HomeOdds, m.DrawOdds, m.AwayOdds)
        if err != nil {
            respondError(w, http.StatusInternalServerError, "Failed to create matches")
            return
        }
        matchIDs = append(matchIDs, matchID)
    }
    
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "status":    "created",
        "bet_id":    betID,
        "matches":   len(req.Matches),
        "match_ids": matchIDs,
    })
    h.WSHub.SendToAll("admin_bet_created", map[string]interface{}{
    "bet_id": betID,
    "title":  req.Title,
})
}

// DELETE /api/v1/admin/match-bets/{betID}
func (h *Handler) AdminDeleteMatchBet(w http.ResponseWriter, r *http.Request) {
    betID := chi.URLParam(r, "betID")
    
    // Soft delete - just mark as deleted
    _, err := h.DB.Exec(`UPDATE admin_match_bets 
                          SET status='DELETED', deleted_at=NOW() 
                          WHERE id=$1 AND status != 'DELETED'`, betID)
    if err != nil {
        respondError(w, http.StatusNotFound, "Bet not found")
        return
    }
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// PUT /api/v1/admin/match-bets/{betID}/lock
func (h *Handler) AdminLockMatchBet(w http.ResponseWriter, r *http.Request) {
    betID := chi.URLParam(r, "betID")
    
    var status string
    err := h.DB.QueryRow("SELECT status FROM admin_match_bets WHERE id=$1", betID).Scan(&status)
    if err != nil {
        respondError(w, http.StatusNotFound, "Bet not found")
        return
    }
    if status != "OPEN" {
        respondError(w, http.StatusBadRequest, "Bet is " + status)
        return
    }
    
    // Lock the parent bet and all matches
    h.DB.Exec("UPDATE admin_match_bets SET status='LOCKED', updated_at=NOW() WHERE id=$1", betID)
    h.DB.Exec("UPDATE admin_matches SET status='LOCKED' WHERE admin_bet_id=$1 AND status='SCHEDULED'", betID)
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "locked"})
    h.WSHub.SendToAll("admin_bet_locked", map[string]interface{}{
    "bet_id": betID,
})
}

// POST /api/v1/admin/matches/{matchID}/settle
func (h *Handler) AdminSettleMatch(w http.ResponseWriter, r *http.Request) {
    matchID := chi.URLParam(r, "matchID")
    
    var req struct {
        HomeScore int `json:"home_score"`
        AwayScore int `json:"away_score"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Calculate result
    var result string
    if req.HomeScore > req.AwayScore {
        result = "HOME_WIN"
    } else if req.HomeScore < req.AwayScore {
        result = "AWAY_WIN"
    } else {
        result = "DRAW"
    }
    
    // Update match
    _, err := h.DB.Exec(`UPDATE admin_matches 
                          SET home_score=$1, away_score=$2, result=$3, status='COMPLETED' 
                          WHERE id=$4`,
        req.HomeScore, req.AwayScore, result, matchID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to settle match")
        return
    }
    
    // Settle all user bets containing this match
    go h.settleAdminMatchBets(matchID, result)
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status": "settled",
        "result": result,
    })
    h.WSHub.SendToAll("admin_bet_settled", map[string]interface{}{
    "match_id": matchID,
    "result":   result,
  })

 rows, _ := h.DB.Query("SELECT DISTINCT user_id FROM user_admin_match_bets WHERE match_picks::text LIKE $1", "%"+matchID+"%")
 for rows.Next() {
    var userID string
    rows.Scan(&userID)
    h.WSHub.SendToUser(userID, "wallet_update", map[string]interface{}{
        "message": "Admin bet settled",
    })
}
}

// GET /api/v1/admin/match-bets (Admin sees all their bets)
func (h *Handler) AdminGetMatchBets(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT amb.id, amb.title, amb.description, amb.status, amb.created_at,
               COUNT(am.id) as match_count,
               COUNT(CASE WHEN am.status = 'COMPLETED' THEN 1 END) as settled_count
        FROM admin_match_bets amb
        LEFT JOIN admin_matches am ON amb.id = am.admin_bet_id
        WHERE amb.admin_id = $1 AND amb.deleted_at IS NULL
        GROUP BY amb.id
        ORDER BY amb.created_at DESC
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, title, desc, status string
        var createdAt time.Time
        var matchCount, settledCount int
        rows.Scan(&id, &title, &desc, &status, &createdAt, &matchCount, &settledCount)
        
        bets = append(bets, map[string]interface{}{
            "id":            id,
            "title":         title,
            "description":   desc,
            "status":        status,
            "created_at":    createdAt,
            "total_matches": matchCount,
            "settled":       settledCount,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// ===== USER ENDPOINTS =====

// GET /api/v1/bets/admin-matches (Users see available admin matches for betting)
func (h *Handler) GetAdminMatchesForBetting(w http.ResponseWriter, r *http.Request) {
    rows, _ := h.DB.Query(`
        SELECT am.id, am.admin_bet_id, am.match_index, 
               am.home_team, am.away_team,
               am.home_odds, am.draw_odds, am.away_odds,
               am.status,
               amb.title as bet_title
        FROM admin_matches am
        JOIN admin_match_bets amb ON am.admin_bet_id = amb.id
        WHERE am.status IN ('SCHEDULED') 
        AND amb.status IN ('OPEN')
        AND amb.deleted_at IS NULL
        ORDER BY amb.created_at DESC, am.match_index ASC
    `)
    defer rows.Close()
    
    var matches []map[string]interface{}
    for rows.Next() {
        var id, betID, homeTeam, awayTeam, status, betTitle string
        var matchIndex int
        var homeOdds, drawOdds, awayOdds float64
        rows.Scan(&id, &betID, &matchIndex, &homeTeam, &awayTeam,
            &homeOdds, &drawOdds, &awayOdds, &status, &betTitle)
        
        matches = append(matches, map[string]interface{}{
            "fixture_id":  id,
            "admin_bet_id": betID,
            "bet_title":   betTitle,
            "match_index": matchIndex,
            "home_team":   homeTeam,
            "away_team":   awayTeam,
            "odds": map[string]float64{
                "home": homeOdds,
                "draw": drawOdds,
                "away": awayOdds,
            },
            "status": status,
        })
    }
    if matches == nil { matches = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, matches)
}

// POST /api/v1/bets/admin-place (User places bet on admin matches)
func (h *Handler) PlaceAdminMatchBet(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    var req struct {
        AdminBetID string `json:"admin_bet_id"`
        MatchPicks []struct {
            MatchID    string  `json:"match_id"`
            Prediction string  `json:"prediction"`
            Odds       float64 `json:"odds"`
        } `json:"match_picks"`
        Amount float64 `json:"amount"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request")
        return
    }
    
    // Validate admin bet is OPEN
    var betStatus string
    err := h.DB.QueryRow("SELECT status FROM admin_match_bets WHERE id=$1", req.AdminBetID).Scan(&betStatus)
    if err != nil || betStatus != "OPEN" {
        respondError(w, http.StatusBadRequest, "Bet is not open")
        return
    }
    
    // Validate all matches are SCHEDULED and get details
    totalOdds := 1.0
    var matchPicksJSON []map[string]interface{}
    
    for _, pick := range req.MatchPicks {
        var homeTeam, awayTeam, matchStatus string
        var homeOdds, drawOdds, awayOdds float64
        
        err := h.DB.QueryRow(`SELECT home_team, away_team, home_odds, draw_odds, away_odds, status 
                               FROM admin_matches WHERE id=$1`, pick.MatchID).
            Scan(&homeTeam, &awayTeam, &homeOdds, &drawOdds, &awayOdds, &matchStatus)
        if err != nil || matchStatus != "SCHEDULED" {
            respondError(w, http.StatusBadRequest, "Invalid match: "+pick.MatchID)
            return
        }
        
        // Get correct odds based on prediction
        var odds float64
        switch pick.Prediction {
        case "HOME_WIN":
            odds = homeOdds
        case "DRAW":
            odds = drawOdds
        case "AWAY_WIN":
            odds = awayOdds
        default:
            respondError(w, http.StatusBadRequest, "Invalid prediction")
            return
        }
        
        totalOdds *= odds
        
        matchPicksJSON = append(matchPicksJSON, map[string]interface{}{
            "match_id":    pick.MatchID,
            "home_team":   homeTeam,
            "away_team":   awayTeam,
            "prediction":  pick.Prediction,
            "odds":        odds,
        })
    }
    
    // Check wallet
    var kash float64
    h.DB.QueryRow("SELECT kash FROM wallets WHERE user_id=$1", user.ID).Scan(&kash)
    if kash < req.Amount {
        respondError(w, http.StatusBadRequest, fmt.Sprintf("Insufficient balance. You have KSh %.0f", kash))
        return
    }
    
    // Calculate potential win (with 12% tax)
    potentialWin := req.Amount * totalOdds * 0.88
    
    // Deduct wallet
    h.DB.Exec("UPDATE wallets SET kash = kash - $1 WHERE user_id = $2", req.Amount, user.ID)
    
    // Place bet
    picksBytes, _ := json.Marshal(matchPicksJSON)
    betID := "UAMB_" + uuid.New().String()[:12]
    _, err = h.DB.Exec(`INSERT INTO user_admin_match_bets 
        (id, user_id, admin_bet_id, match_picks, total_odds, amount, potential_win)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
        betID, user.ID, req.AdminBetID, string(picksBytes), totalOdds, req.Amount, potentialWin)
    
    if err != nil {
        // Refund
        h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", req.Amount, user.ID)
        respondError(w, http.StatusInternalServerError, "Failed to place bet")
        return
    }
    
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "status":        "placed",
        "bet_id":        betID,
        "total_odds":    totalOdds,
        "potential_win": potentialWin,
        "stake":         req.Amount,
    })
}

// GET /api/v1/bets/admin-active (User's active admin bets)
func (h *Handler) GetActiveAdminBets(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT uamb.id, uamb.admin_bet_id, amb.title, uamb.match_picks::text,
               uamb.total_odds, uamb.amount, uamb.potential_win, uamb.placed_at
        FROM user_admin_match_bets uamb
        JOIN admin_match_bets amb ON uamb.admin_bet_id = amb.id
        WHERE uamb.user_id = $1 AND uamb.status = 'PENDING'
        ORDER BY uamb.placed_at DESC
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, betID, title, picksJSON string
        var totalOdds, amount, potentialWin float64
        var placedAt time.Time
        rows.Scan(&id, &betID, &title, &picksJSON, &totalOdds, &amount, &potentialWin, &placedAt)
        
        var picks interface{}
        json.Unmarshal([]byte(picksJSON), &picks)
        
        bets = append(bets, map[string]interface{}{
            "id":            id,
            "admin_bet_id":  betID,
            "title":         title,
            "match_picks":   picks,
            "total_odds":    totalOdds,
            "amount":        amount,
            "potential_win": potentialWin,
            "placed_at":     placedAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// GET /api/v1/bets/admin-history (User's admin bet history)
func (h *Handler) GetAdminBetHistory(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    rows, _ := h.DB.Query(`
        SELECT uamb.id, amb.title, uamb.match_picks::text,
               uamb.total_odds, uamb.amount, uamb.status, uamb.payout, uamb.settled_at
        FROM user_admin_match_bets uamb
        JOIN admin_match_bets amb ON uamb.admin_bet_id = amb.id
        WHERE uamb.user_id = $1 AND uamb.status != 'PENDING'
        ORDER BY uamb.settled_at DESC
    `, user.ID)
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, title, picksJSON, status string
        var totalOdds, amount, payout float64
        var settledAt time.Time
        rows.Scan(&id, &title, &picksJSON, &totalOdds, &amount, &status, &payout, &settledAt)
        
        var picks interface{}
        json.Unmarshal([]byte(picksJSON), &picks)
        
        bets = append(bets, map[string]interface{}{
            "id":          id,
            "title":       title,
            "match_picks": picks,
            "total_odds":  totalOdds,
            "amount":      amount,
            "status":      status,
            "payout":      payout,
            "settled_at":  settledAt,
        })
    }
    if bets == nil { bets = []map[string]interface{}{} }
    
    respondJSON(w, http.StatusOK, bets)
}

// ===== SETTLEMENT LOGIC (Separate from league) =====

func (h *Handler) settleAdminMatchBets(matchID string, result string) {
    // Find all pending bets containing this match
    rows, _ := h.DB.Query(`
        SELECT id, user_id, match_picks::text, total_odds, amount, potential_win
        FROM user_admin_match_bets 
        WHERE status = 'PENDING' 
        AND match_picks::text LIKE $1
    `, "%"+matchID+"%")
    
    if rows == nil {
        return
    }
    defer rows.Close()
    
    for rows.Next() {
        var betID, userID, picksJSON string
        var totalOdds, amount, potentialWin float64
        rows.Scan(&betID, &userID, &picksJSON, &totalOdds, &amount, &potentialWin)
        
        var picks []map[string]interface{}
        json.Unmarshal([]byte(picksJSON), &picks)
        
        // Check if all matches in this bet are settled
        allSettled := true
        allCorrect := true
        
        for _, pick := range picks {
            pickMatchID := pick["match_id"].(string)
            prediction := pick["prediction"].(string)
            
            var matchResult string
            err := h.DB.QueryRow("SELECT result FROM admin_matches WHERE id=$1", pickMatchID).Scan(&matchResult)
            if err != nil || matchResult == "" {
                allSettled = false
                break
            }
            
            if matchResult != prediction {
                allCorrect = false
            }
        }
        
        if allSettled {
            if allCorrect {
                // Calculate payout (already had 12% tax deducted in potential_win)
                h.DB.Exec(`UPDATE user_admin_match_bets 
                           SET status='WON', payout=$1, settled_at=NOW() 
                           WHERE id=$2`, potentialWin, betID)
                h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", potentialWin, userID)
            } else {
                h.DB.Exec(`UPDATE user_admin_match_bets 
                           SET status='LOST', settled_at=NOW() 
                           WHERE id=$1`, betID)
            }
        }
    }
}
// GET /api/v1/admin/match-bets/{betID}/matches
// Returns ALL matches for a specific bet (admin only - for settlement)
func (h *Handler) AdminGetBetMatches(w http.ResponseWriter, r *http.Request) {
    betID := chi.URLParam(r, "betID")
    
    rows, err := h.DB.Query(`
        SELECT id, match_index, home_team, away_team,
               home_odds, draw_odds, away_odds,
               COALESCE(home_score, -1) as home_score,
               COALESCE(away_score, -1) as away_score,
               COALESCE(result, '') as result,
               status
        FROM admin_matches 
        WHERE admin_bet_id = $1
        ORDER BY match_index ASC
    `, betID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Database error")
        return
    }
    defer rows.Close()
    
    var matches []map[string]interface{}
    for rows.Next() {
        var id, result, status string
        var matchIndex, homeScore, awayScore int
        var homeOdds, drawOdds, awayOdds float64
        var homeTeam, awayTeam string
        
        rows.Scan(&id, &matchIndex, &homeTeam, &awayTeam,
            &homeOdds, &drawOdds, &awayOdds,
            &homeScore, &awayScore, &result, &status)
        
        m := map[string]interface{}{
            "fixture_id":  id,
            "match_index": matchIndex,
            "home_team":   homeTeam,
            "away_team":   awayTeam,
            "odds": map[string]float64{
                "home": homeOdds,
                "draw": drawOdds,
                "away": awayOdds,
            },
            "status": status,
            "result": result,
        }
        
        if homeScore >= 0 && awayScore >= 0 {
            m["home_score"] = homeScore
            m["away_score"] = awayScore
        }
        
        matches = append(matches, m)
    }
    
    if matches == nil {
        matches = []map[string]interface{}{}
    }
    
    respondJSON(w, http.StatusOK, matches)
}