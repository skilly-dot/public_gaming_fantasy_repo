// internal/handlers/quick_match_handler.go - WITH FULL DATA PUSH

package handlers

import (
    "encoding/json"
    "fmt"
    "math"
    "math/rand"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
)

// GenerateQuickMatch - Generates random matchup with smart odds
func (h *Handler) GenerateQuickMatch(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    leagueID := chi.URLParam(r, "leagueID")
    
    if user == nil {
        respondError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    var leagueUserID string
    err := h.DB.QueryRow("SELECT user_id FROM leagues WHERE id=$1", leagueID).Scan(&leagueUserID)
    if err != nil || leagueUserID != user.ID {
        respondError(w, http.StatusForbidden, "League not found")
        return
    }
    
    rows, err := h.DB.Query(`
        SELECT id, name, rating, attack, defense 
        FROM teams 
        WHERE league_id=$1 
        ORDER BY RANDOM() 
        LIMIT 2
    `, leagueID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to get teams")
        return
    }
    defer rows.Close()
    
    type TeamInfo struct {
        ID      string  `json:"id"`
        Name    string  `json:"name"`
        Rating  float64 `json:"rating"`
        Attack  int     `json:"attack"`
        Defense int     `json:"defense"`
    }
    
    var teams []TeamInfo
    for rows.Next() {
        var t TeamInfo
        rows.Scan(&t.ID, &t.Name, &t.Rating, &t.Attack, &t.Defense)
        teams = append(teams, t)
    }
    
    if len(teams) < 2 {
        respondError(w, http.StatusBadRequest, "Not enough teams")
        return
    }
    
    homeTeam := teams[0]
    awayTeam := teams[1]
    
    homeOdds, drawOdds, awayOdds := calculateSmartOdds(
        homeTeam.Rating, float64(homeTeam.Attack), float64(homeTeam.Defense),
        awayTeam.Rating, float64(awayTeam.Attack), float64(awayTeam.Defense),
    )
    
    quickMatchID := "QM_" + uuid.New().String()[:12]
    
    h.DB.Exec(`INSERT INTO quick_matches (id, league_id, user_id, home_team_id, away_team_id, home_odds, draw_odds, away_odds, status)
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PENDING')`,
        quickMatchID, leagueID, user.ID, homeTeam.ID, awayTeam.ID, homeOdds, drawOdds, awayOdds)
    
    // ===== PUSH USER STATE - Quick match generated =====
    go h.PushUserState(user.ID, "quick_match_generated", map[string]interface{}{
        "quick_match_id": quickMatchID,
        "home_team":      homeTeam,
        "away_team":      awayTeam,
        "odds": map[string]float64{
            "home": homeOdds,
            "draw": drawOdds,
            "away": awayOdds,
        },
    })
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "quick_match_id": quickMatchID,
        "home_team": homeTeam,
        "away_team": awayTeam,
        "odds": map[string]float64{
            "home": homeOdds,
            "draw": drawOdds,
            "away": awayOdds,
        },
    })
}

// BetOnQuickMatch - User places bet
func (h *Handler) BetOnQuickMatch(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    leagueID := chi.URLParam(r, "leagueID")
    quickMatchID := chi.URLParam(r, "quickMatchID")
    
    if user == nil {
        respondError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    var req struct {
        Prediction string  `json:"prediction"`
        Amount     float64 `json:"amount"`
        Odds       float64 `json:"odds"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // GET ODDS FROM DB if not provided
    if req.Odds <= 0 {
        var homeOdds, drawOdds, awayOdds float64
        h.DB.QueryRow("SELECT home_odds, draw_odds, away_odds FROM quick_matches WHERE id=$1", quickMatchID).
            Scan(&homeOdds, &drawOdds, &awayOdds)
        
        switch req.Prediction {
        case "HOME_WIN":
            req.Odds = homeOdds
        case "DRAW":
            req.Odds = drawOdds
        case "AWAY_WIN":
            req.Odds = awayOdds
        }
    }
    
    // Verify quick match
    var status string
    var homeID, awayID string
    err := h.DB.QueryRow("SELECT status, home_team_id, away_team_id FROM quick_matches WHERE id=$1 AND user_id=$2", 
        quickMatchID, user.ID).Scan(&status, &homeID, &awayID)
    
    if err != nil || status != "PENDING" {
        respondError(w, http.StatusBadRequest, "Quick match not available")
        return
    }
    
    // Check wallet
    var kash float64
    h.DB.QueryRow("SELECT kash FROM wallets WHERE user_id=$1", user.ID).Scan(&kash)
    if kash < req.Amount {
        respondError(w, http.StatusBadRequest, "Insufficient balance")
        return
    }
    
    // Deduct stake
    h.DB.Exec("UPDATE wallets SET kash = kash - $1 WHERE user_id = $2", req.Amount, user.ID)
    
    // Create bet
    betID := "QMB_" + uuid.New().String()[:12]
    betsJSON, _ := json.Marshal([]map[string]interface{}{
        {
            "fixture_id": quickMatchID,
            "prediction": req.Prediction,
            "home_team_id": homeID,
            "away_team_id": awayID,
        },
    })
    
    h.DB.Exec(`INSERT INTO bets (id, user_id, league_id, week, bets, amount, total_odds, is_custom, status, is_quick_match, quick_match_id, placed_at)
               VALUES ($1, $2, $3, 0, $4, $5, $6, true, 'PENDING', true, $7, NOW())`,
        betID, user.ID, leagueID, betsJSON, req.Amount, req.Odds, quickMatchID)
    
    h.DB.Exec("UPDATE quick_matches SET status='BET_PLACED' WHERE id=$1", quickMatchID)
    
    // ===== PUSH FULL USER STATE - Quick match bet placed =====
    go h.PushUserState(user.ID, "quick_match_bet_placed", map[string]interface{}{
        "bet_id":        betID,
        "quick_match_id": quickMatchID,
        "odds":          req.Odds,
        "potential_win": req.Amount * req.Odds * 0.88,
    })
    
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "status": "bet_placed",
        "bet_id": betID,
        "odds": req.Odds,
        "potential_win": req.Amount * req.Odds * 0.88,
    })
}

// StartQuickMatch - Simulates match and settles bet
func (h *Handler) StartQuickMatch(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    quickMatchID := chi.URLParam(r, "quickMatchID")
    
    if user == nil {
        respondError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    var homeID, awayID, status string
    err := h.DB.QueryRow("SELECT home_team_id, away_team_id, status FROM quick_matches WHERE id=$1 AND user_id=$2", 
        quickMatchID, user.ID).Scan(&homeID, &awayID, &status)
    
    if err != nil || status != "BET_PLACED" {
        respondError(w, http.StatusBadRequest, "Place bet first")
        return
    }
    
    // SMART SIMULATION
    homeScore, awayScore, winner := h.simulateQuickMatchSmart(homeID, awayID)
    
    var homeName, awayName string
    h.DB.QueryRow("SELECT name FROM teams WHERE id=$1", homeID).Scan(&homeName)
    h.DB.QueryRow("SELECT name FROM teams WHERE id=$1", awayID).Scan(&awayName)
    
    // Settle bet
    var betID, betsStr string
    var amount, totalOdds float64
    err = h.DB.QueryRow("SELECT id, bets::text, amount, total_odds FROM bets WHERE quick_match_id=$1 AND status='PENDING'", 
        quickMatchID).Scan(&betID, &betsStr, &amount, &totalOdds)
    
    if err == nil {
        var predictions []map[string]interface{}
        json.Unmarshal([]byte(betsStr), &predictions)
        
        if len(predictions) > 0 {
            prediction := predictions[0]["prediction"].(string)
            
            // CORRECT winner check
            won := false
            if prediction == "HOME_WIN" && winner == "HOME" {
                won = true
            } else if prediction == "DRAW" && winner == "DRAW" {
                won = true
            } else if prediction == "AWAY_WIN" && winner == "AWAY" {
                won = true
            }
            
            if won {
                payout := amount * totalOdds * 0.88
                h.DB.Exec("UPDATE bets SET status='WON', payout=$1, settled_at=NOW() WHERE id=$2", payout, betID)
                h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", payout, user.ID)
                
                // ===== PUSH FULL USER STATE - Quick match won =====
                go h.PushUserState(user.ID, "quick_match_won", map[string]interface{}{
                    "bet_id":         betID,
                    "quick_match_id": quickMatchID,
                    "payout":         payout,
                    "message":        fmt.Sprintf("Quick match WON! +$%.2f", payout),
                })
            } else {
                h.DB.Exec("UPDATE bets SET status='LOST', settled_at=NOW() WHERE id=$1", betID)
                
                // ===== PUSH FULL USER STATE - Quick match lost =====
                go h.PushUserState(user.ID, "quick_match_lost", map[string]interface{}{
                    "bet_id":         betID,
                    "quick_match_id": quickMatchID,
                    "message":        "Quick match lost",
                })
            }
        }
    }
    
    h.DB.Exec("UPDATE quick_matches SET status='COMPLETED', home_score=$1, away_score=$2, winner=$3 WHERE id=$4", 
        homeScore, awayScore, winner, quickMatchID)
    
    // ===== PUSH FULL STATE - Quick match completed =====
    go h.PushUserState(user.ID, "quick_match_completed", map[string]interface{}{
        "quick_match_id": quickMatchID,
        "home_team":      homeName,
        "away_team":      awayName,
        "home_score":     homeScore,
        "away_score":     awayScore,
        "winner":         winner,
    })
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status": "completed",
        "result": map[string]interface{}{
            "home_team": homeName,
            "away_team": awayName,
            "home_score": homeScore,
            "away_score": awayScore,
            "winner": winner,
        },
    })
}

// calculateSmartOdds - Team strength based odds
func calculateSmartOdds(homeRating, homeAttack, homeDefense, awayRating, awayAttack, awayDefense float64) (float64, float64, float64) {
    homeStrength := homeRating*0.6 + homeAttack*0.3 + homeDefense*0.1
    awayStrength := awayRating*0.6 + awayAttack*0.3 + awayDefense*0.1
    
    homeStrength *= 1.15
    
    totalStrength := homeStrength + awayStrength
    homeProb := homeStrength / totalStrength
    awayProb := awayStrength / totalStrength
    
    drawProb := 0.25
    
    homeProb = homeProb * (1 - drawProb)
    awayProb = awayProb * (1 - drawProb)
    
    if homeProb < 0.15 { homeProb = 0.15 }
    if homeProb > 0.70 { homeProb = 0.70 }
    if awayProb < 0.10 { awayProb = 0.10 }
    if awayProb > 0.60 { awayProb = 0.60 }
    
    total := homeProb + drawProb + awayProb
    homeProb /= total
    drawProb /= total
    awayProb /= total
    
    homeOdds := math.Round((1.06/homeProb)*100) / 100
    drawOdds := math.Round((1.06/drawProb)*100) / 100
    awayOdds := math.Round((1.06/awayProb)*100) / 100
    
    return homeOdds, drawOdds, awayOdds
}

// simulateQuickMatchSmart - Poisson based on team strength
func (h *Handler) simulateQuickMatchSmart(homeID, awayID string) (int, int, string) {
    var homeRating, awayRating float64
    var homeAttack, homeDefense, awayAttack, awayDefense int
    
    h.DB.QueryRow("SELECT rating, attack, defense FROM teams WHERE id=$1", homeID).Scan(&homeRating, &homeAttack, &homeDefense)
    h.DB.QueryRow("SELECT rating, attack, defense FROM teams WHERE id=$1", awayID).Scan(&awayRating, &awayAttack, &awayDefense)
    
    homeLambda := 1.2 + (homeRating-awayRating)*0.5 + float64(homeAttack-awayDefense)*0.02
    awayLambda := 1.0 + (awayRating-homeRating)*0.4 + float64(awayAttack-homeDefense)*0.02
    
    homeLambda *= 1.15
    
    if homeLambda < 0.3 { homeLambda = 0.3 }
    if homeLambda > 3.5 { homeLambda = 3.5 }
    if awayLambda < 0.3 { awayLambda = 0.3 }
    if awayLambda > 3.5 { awayLambda = 3.5 }
    
    homeScore := poissonRandom(homeLambda)
    awayScore := poissonRandom(awayLambda)
    
    if homeScore > 5 { homeScore = 5 }
    if awayScore > 5 { awayScore = 5 }
    
    winner := "DRAW"
    if homeScore > awayScore { winner = "HOME" }
    if awayScore > homeScore { winner = "AWAY" }
    
    return homeScore, awayScore, winner
}

// poissonRandom - Poisson distribution
func poissonRandom(lambda float64) int {
    L := math.Exp(-lambda)
    k := 0
    p := 1.0
    for p > L {
        k++
        p *= rand.Float64()
    }
    return k - 1
}