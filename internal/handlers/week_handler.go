// internal/handlers/simulate_handler.go - WITH FULL DATA PUSH

package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// POST /api/v1/leagues/{leagueID}/start-week
func (h *Handler) StartWeek(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    user := getUserFromContext(r.Context())

    // Verify league belongs to user
    var leagueUserID string
    err := h.DB.QueryRow("SELECT user_id FROM leagues WHERE id=$1", leagueID).Scan(&leagueUserID)
    if err != nil || leagueUserID != user.ID {
        respondError(w, http.StatusForbidden, "League not found")
        return
    }

    // Check if week already started
    var started bool
    h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM daily_matches WHERE league_id=$1 AND status != 'SCHEDULED')", leagueID).Scan(&started)
    if started {
        respondError(w, http.StatusBadRequest, "Week already in progress")
        return
    }

    // ===== LOCK ALL PENDING BETS FOR THIS LEAGUE =====
    result, err := h.DB.Exec(`
        UPDATE bets 
        SET status = 'LOCKED' 
        WHERE league_id = $1 
        AND status = 'PENDING'
    `, leagueID)
    
    if err != nil {
        log.Printf("Error locking bets: %v", err)
    } else {
        lockedCount, _ := result.RowsAffected()
        log.Printf("Locked %d bets for league %s", lockedCount, leagueID)
    }

    // Start week simulation in background
    go h.simulateWeek(leagueID, user.ID)

    // ===== PUSH FULL STATE - Week started =====
    go h.PushFullState(user.ID, leagueID, "week_started", map[string]interface{}{
        "league_id": leagueID,
        "message":   "Week started! Bets locked.",
    })

    respondJSON(w, http.StatusOK, map[string]string{
        "status":  "started",
        "message": "Week started! Bets locked.",
    })
}

// simulateWeek simulates all matches for a week
func (h *Handler) simulateWeek(leagueID string, userID string) {
    for {
        var match struct {
            ID        int
            FixtureID string
            HomeID    string
            AwayID    string
            WeekNum   int
            HomeName  string
            AwayName  string
        }

        err := h.DB.QueryRow(`
            SELECT dm.id, dm.fixture_id, f.home_team_id, f.away_team_id, f.week_number,
                   t1.name, t2.name
            FROM daily_matches dm
            JOIN fixtures f ON dm.fixture_id = f.id
            JOIN teams t1 ON f.home_team_id = t1.id
            JOIN teams t2 ON f.away_team_id = t2.id
            WHERE dm.league_id = $1 AND dm.status = 'SCHEDULED'
            ORDER BY dm.kickoff_time LIMIT 1
        `, leagueID).Scan(&match.ID, &match.FixtureID, &match.HomeID, &match.AwayID, &match.WeekNum, &match.HomeName, &match.AwayName)

        if err != nil {
            // All matches done - settle bets
            h.settleWeekBets(leagueID, userID)
            return
        }

        // Simulate immediately
        h.simulateMatch(leagueID, userID, match)
        time.Sleep(45 * time.Second) // Match duration
    }
}

// updateLeagueTable updates standings after a match
func (h *Handler) updateLeagueTable(leagueID string, homeID string, awayID string, homeScore int, awayScore int, winner string) {
    // Update home team
    h.DB.Exec(`UPDATE league_table SET 
        played=played+1, 
        goals_for=goals_for+$1, 
        goals_against=goals_against+$2,
        won=won+CASE WHEN $3='HOME' THEN 1 ELSE 0 END, 
        drawn=drawn+CASE WHEN $3='DRAW' THEN 1 ELSE 0 END, 
        lost=lost+CASE WHEN $3='AWAY' THEN 1 ELSE 0 END,
        points=points+CASE WHEN $3='HOME' THEN 3 WHEN $3='DRAW' THEN 1 ELSE 0 END 
        WHERE team_id=$4 AND league_id=$5`,
        homeScore, awayScore, winner, homeID, leagueID)

    // Update away team
    h.DB.Exec(`UPDATE league_table SET 
        played=played+1, 
        goals_for=goals_for+$1, 
        goals_against=goals_against+$2,
        won=won+CASE WHEN $3='AWAY' THEN 1 ELSE 0 END, 
        drawn=drawn+CASE WHEN $3='DRAW' THEN 1 ELSE 0 END, 
        lost=lost+CASE WHEN $3='HOME' THEN 1 ELSE 0 END,
        points=points+CASE WHEN $3='AWAY' THEN 3 WHEN $3='DRAW' THEN 1 ELSE 0 END 
        WHERE team_id=$4 AND league_id=$5`,
        awayScore, homeScore, winner, awayID, leagueID)
}

// settleWeekBets settles all LOCKED bets for a league week
func (h *Handler) settleWeekBets(leagueID string, userID string) {
    rows, _ := h.DB.Query(`
        SELECT b.id, b.user_id, b.bets::text, b.amount, b.is_custom
        FROM bets b 
        WHERE b.league_id = $1 
        AND b.status = 'LOCKED'
    `, leagueID)
    if rows == nil {
        // ===== PUSH FULL STATE - Week completed (no bets) =====
        go h.PushFullState(userID, leagueID, "week_completed", map[string]interface{}{
            "league_id": leagueID,
            "message":   "Week complete!",
        })
        return
    }
    defer rows.Close()

    settledCount := 0
    for rows.Next() {
        var betID, betUserID, betsStr string
        var amount float64
        var isCustom bool
        rows.Scan(&betID, &betUserID, &betsStr, &amount, &isCustom)

        var predictions []map[string]interface{}
        json.Unmarshal([]byte(betsStr), &predictions)

        allCorrect := true
        for _, p := range predictions {
            fixtureID := p["fixture_id"].(string)
            prediction := p["prediction"].(string)
            var matchWinner string
            h.DB.QueryRow("SELECT winner FROM daily_matches WHERE fixture_id=$1 AND status='COMPLETED'", fixtureID).Scan(&matchWinner)
            if (prediction == "HOME_WIN" && matchWinner != "HOME") ||
                (prediction == "DRAW" && matchWinner != "DRAW") ||
                (prediction == "AWAY_WIN" && matchWinner != "AWAY") {
                allCorrect = false
            }
        }

        if allCorrect {
            payout := amount * 2.0
            tax := payout * 0.12
            netPayout := payout - tax
            h.DB.Exec(`UPDATE bets SET status='WON', payout=$1, tax=$2, points=$3, coins=$4, settled_at=NOW() WHERE id=$5`,
                netPayout, tax, int(amount*2), int(amount*0.5), betID)
            h.DB.Exec(`UPDATE wallets SET kash=kash+$1, points=points+$2, coins=coins+$3 WHERE user_id=$4`,
                netPayout, int(amount*2), int(amount*0.5), betUserID)

            // ===== PUSH FULL USER STATE - Bet won =====
            go h.PushUserState(betUserID, "bet_won", map[string]interface{}{
                "bet_id":  betID,
                "payout":  netPayout,
                "message": fmt.Sprintf("You won $%.0f!", netPayout),
            })
        } else {
            h.DB.Exec(`UPDATE bets SET status='LOST', points=5, settled_at=NOW() WHERE id=$1`, betID)
            h.DB.Exec(`UPDATE wallets SET points=points+5 WHERE user_id=$1`, betUserID)

            // ===== PUSH FULL USER STATE - Bet lost =====
            go h.PushUserState(betUserID, "bet_lost", map[string]interface{}{
                "bet_id":  betID,
                "message": "Better luck next time!",
            })
        }
        
        settledCount++
    }

    // ===== PUSH FULL LEAGUE STATE - Week completed =====
    go h.PushFullState(userID, leagueID, "week_completed", map[string]interface{}{
        "league_id": leagueID,
        "settled":   settledCount,
        "message":   fmt.Sprintf("Week complete! %d bets settled", settledCount),
    })
}

// getTeamPlayers returns player names for a team
func (h *Handler) getTeamPlayers(teamID string) []string {
    rows, _ := h.DB.Query("SELECT name FROM players WHERE team_id=$1 ORDER BY RANDOM() LIMIT 18", teamID)
    defer rows.Close()
    var players []string
    for rows.Next() {
        var name string
        rows.Scan(&name)
        players = append(players, name)
    }
    return players
}

// POST /api/v1/leagues/{leagueID}/next-week
func (h *Handler) NextWeek(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    user := getUserFromContext(r.Context())

    var leagueUserID string
    err := h.DB.QueryRow("SELECT user_id FROM leagues WHERE id=$1", leagueID).Scan(&leagueUserID)
    if err != nil || leagueUserID != user.ID {
        respondError(w, http.StatusForbidden, "League not found")
        return
    }

    var pending int
    h.DB.QueryRow("SELECT COUNT(*) FROM daily_matches WHERE league_id=$1 AND status != 'COMPLETED'", leagueID).Scan(&pending)
    if pending > 0 {
        respondError(w, http.StatusBadRequest, "Not all matches completed")
        return
    }

    var dayNumber, totalWeeks int
    h.DB.QueryRow("SELECT day_number, total_weeks FROM leagues WHERE id=$1", leagueID).Scan(&dayNumber, &totalWeeks)

    if dayNumber >= totalWeeks {
        h.DB.Exec("UPDATE leagues SET status='COMPLETED' WHERE id=$1", leagueID)
        
        // ===== PUSH FULL STATE - League completed =====
        go h.PushFullState(user.ID, leagueID, "league_completed", map[string]interface{}{
            "league_id": leagueID,
            "message":   "League completed!",
        })
        
        respondJSON(w, http.StatusOK, map[string]string{"status": "league_completed"})
        return
    }

    nextWeek := dayNumber + 1

    // Delete last week's match_results
    h.DB.Exec("DELETE FROM match_results WHERE league_id=$1", leagueID)

    // Delete daily schedule
    h.DB.Exec("DELETE FROM daily_matches WHERE league_id=$1", leagueID)

    // Schedule new week
    h.scheduleDailyMatches(leagueID, nextWeek)

    // Update both day_number and current_week
    h.DB.Exec("UPDATE leagues SET day_number=$1, current_week=$1 WHERE id=$2", nextWeek, leagueID)

    // ===== PUSH FULL STATE - Week advanced =====
    go h.PushFullState(user.ID, leagueID, "week_advanced", map[string]interface{}{
        "league_id": leagueID,
        "week":      nextWeek,
        "message":   fmt.Sprintf("Week %d ready! Place your bets.", nextWeek),
    })

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status":  "advanced",
        "week":    nextWeek,
        "message": fmt.Sprintf("Week %d ready! Place your bets.", nextWeek),
    })
}

// scheduleDailyMatches schedules matches for a day
func (h *Handler) scheduleDailyMatches(leagueID string, dayNumber int) {
    now := time.Now()
    firstKickoff := now.Add(5 * time.Minute)
    lastKickoff := now.Add(22 * time.Hour)

    var teamCount int
    h.DB.QueryRow("SELECT COUNT(*) FROM teams WHERE league_id=$1", leagueID).Scan(&teamCount)

    matchesPerDay := teamCount / 2
    if matchesPerDay > 10 {
        matchesPerDay = 10
    }
    if matchesPerDay < 2 {
        matchesPerDay = 2
    }

    rows, _ := h.DB.Query(`
        SELECT f.id FROM fixtures f
        WHERE f.league_id = $1 AND f.week_number = $2
        ORDER BY f.id LIMIT $3
    `, leagueID, dayNumber, matchesPerDay)
    defer rows.Close()

    var fixtureIDs []string
    for rows.Next() {
        var id string
        rows.Scan(&id)
        fixtureIDs = append(fixtureIDs, id)
    }

    if len(fixtureIDs) == 0 {
        return
    }

    totalWindow := lastKickoff.Sub(firstKickoff)
    interval := totalWindow / time.Duration(len(fixtureIDs)-1)

    for i, fixID := range fixtureIDs {
        kickoff := firstKickoff.Add(time.Duration(i) * interval)
        h.DB.Exec(`INSERT INTO daily_matches (league_id, day_number, fixture_id, kickoff_time, status)
                   VALUES ($1, $2, $3, $4, 'SCHEDULED')`,
            leagueID, dayNumber, fixID, kickoff)
    }
}

// getCurrentWeek returns the current week number
func (h *Handler) getCurrentWeek(leagueID string) int {
    var week int
    err := h.DB.QueryRow("SELECT COALESCE(current_week, day_number, 1) FROM leagues WHERE id = $1", leagueID).Scan(&week)
    if err != nil {
        return 1
    }
    return week
}

// GetTopScorers returns top scorers for a league
func (h *Handler) GetTopScorers(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")

    rows, _ := h.DB.Query(`
        SELECT player_name, goals
        FROM top_scorers
        WHERE league_id = $1
        ORDER BY goals DESC
        LIMIT 20
    `, leagueID)
    defer rows.Close()

    var scorers []map[string]interface{}
    pos := 1
    for rows.Next() {
        var name string
        var goals int
        rows.Scan(&name, &goals)
        scorers = append(scorers, map[string]interface{}{
            "position": pos,
            "name":     name,
            "goals":    goals,
        })
        pos++
    }
    if scorers == nil {
        scorers = []map[string]interface{}{}
    }
    respondJSON(w, http.StatusOK, scorers)
}

func (h *Handler) simulateMatch(leagueID string, userID string, match struct {
    ID        int
    FixtureID string
    HomeID    string
    AwayID    string
    WeekNum   int
    HomeName  string
    AwayName  string
}) {
    // NORMALIZED REALISTIC SCORE - Max 5 goals total
    homeScore, awayScore, winner := h.generateRealisticScore()

    // Rich stats
    possession := 40 + rand.Intn(30)
    stats := map[string]interface{}{
        "possession":      map[string]int{"home": possession, "away": 100 - possession},
        "shots":           map[string]int{"home": homeScore + rand.Intn(8) + 2, "away": awayScore + rand.Intn(6) + 1},
        "shots_on_target": map[string]int{"home": homeScore + rand.Intn(3), "away": awayScore + rand.Intn(2)},
        "corners":         map[string]int{"home": 2 + rand.Intn(8), "away": 1 + rand.Intn(6)},
        "cards":           map[string]int{"home": rand.Intn(3), "away": rand.Intn(3)},
        "fouls":           map[string]int{"home": rand.Intn(12) + 3, "away": rand.Intn(12) + 3},
    }

    // Get players
    homePlayers := h.getTeamPlayers(match.HomeID)
    awayPlayers := h.getTeamPlayers(match.AwayID)

    // Generate goals with UNIQUE minutes
    goals := []map[string]interface{}{}
    usedMinutes := make(map[int]bool)
    
    generateUniqueMinute := func() int {
        for {
            minute := rand.Intn(90) + 1
            if !usedMinutes[minute] {
                usedMinutes[minute] = true
                return minute
            }
        }
    }
    
    // Home goals
    for i := 0; i < homeScore; i++ {
        minute := generateUniqueMinute()
        playerName := "Unknown"
        if len(homePlayers) > 0 {
            playerName = homePlayers[rand.Intn(len(homePlayers))]
        }
        goals = append(goals, map[string]interface{}{
            "minute": minute, "team": "HOME", "player": playerName, "type": randomGoalType(),
        })
    }
    
    // Away goals
    for i := 0; i < awayScore; i++ {
        minute := generateUniqueMinute()
        playerName := "Unknown"
        if len(awayPlayers) > 0 {
            playerName = awayPlayers[rand.Intn(len(awayPlayers))]
        }
        goals = append(goals, map[string]interface{}{
            "minute": minute, "team": "AWAY", "player": playerName, "type": randomGoalType(),
        })
    }

    goalsJSON, _ := json.Marshal(goals)
    statsJSON, _ := json.Marshal(stats)

    // Update daily_matches
    h.DB.Exec(`UPDATE daily_matches SET status='COMPLETED', home_score=$1, away_score=$2, winner=$3 WHERE id=$4`,
        homeScore, awayScore, winner, match.ID)

    // Update fixtures
    h.DB.Exec(`UPDATE fixtures SET home_score=$1, away_score=$2, winner=$3, status='COMPLETED' WHERE id=$4`,
        homeScore, awayScore, winner, match.FixtureID)

    // Save to match_results
    resultID := "RES_" + uuid.New().String()[:12]
    h.DB.Exec(`INSERT INTO match_results (id, league_id, fixture_id, week_number, home_team_id, away_team_id, home_score, away_score, winner, goals, stats)
               VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
        resultID, leagueID, match.FixtureID, match.WeekNum, match.HomeID, match.AwayID, homeScore, awayScore, winner, goalsJSON, statsJSON)

    // Update top scorers
    for _, goal := range goals {
        playerName, _ := goal["player"].(string)
        if playerName != "" && playerName != "Unknown" {
            h.DB.Exec(`
                INSERT INTO top_scorers (league_id, player_name, goals) 
                VALUES ($1, $2, 1)
                ON CONFLICT (league_id, player_name) 
                DO UPDATE SET goals = top_scorers.goals + 1
            `, leagueID, playerName)
        }
    }

    // Update league table
    h.updateLeagueTable(leagueID, match.HomeID, match.AwayID, homeScore, awayScore, winner)

    // ===== PUSH FULL LEAGUE STATE - Match completed =====
    go h.PushFullState(userID, leagueID, "match_completed", map[string]interface{}{
        "match_id":   match.FixtureID,
        "home_team":  match.HomeName,
        "away_team":  match.AwayName,
        "home_score": homeScore,
        "away_score": awayScore,
        "winner":     winner,
        "goals":      goals,
        "stats":      stats,
    })

    fmt.Printf("Match: %s %d-%d %s\n", match.HomeName, homeScore, awayScore, match.AwayName)
}

// generateRealisticScore - Weighted common scores
func (h *Handler) generateRealisticScore() (int, int, string) {
    scores := []struct{ home, away int }{
        {1, 0}, {2, 1}, {1, 1}, {2, 0}, {0, 0},
        {3, 1}, {3, 2}, {2, 2}, {0, 1}, {1, 2},
        {0, 2}, {3, 0}, {0, 3}, {4, 1}, {1, 3},
        {4, 0}, {0, 4}, {5, 0}, {0, 5}, {4, 2},
    }
    
    weights := []int{15, 12, 10, 10, 8, 8, 6, 6, 8, 8, 6, 5, 5, 3, 3, 2, 2, 1, 1, 1}
    
    totalWeight := 0
    for _, w := range weights {
        totalWeight += w
    }
    
    roll := rand.Intn(totalWeight)
    cumulative := 0
    for i, w := range weights {
        cumulative += w
        if roll < cumulative {
            home := scores[i].home
            away := scores[i].away
            winner := "DRAW"
            if home > away { winner = "HOME" } else if away > home { winner = "AWAY" }
            return home, away, winner
        }
    }
    
    return 1, 0, "HOME"
}

// randomGoalType - NO penalties
func randomGoalType() string {
    types := []string{"open_play", "header", "free_kick", "long_range", "counter_attack"}
    return types[rand.Intn(len(types))]
}