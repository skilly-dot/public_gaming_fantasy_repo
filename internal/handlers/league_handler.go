// internal/handlers/league_handler.go - FULL using UltimateGenerator
package handlers

import (
	//"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/betking/rich-backend/internal/engine"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FINISH LEAGUE - With validation
func (h *Handler) FinishLeague(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    leagueID := chi.URLParam(r, "leagueID")
    
    if user == nil {
        respondError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    var leagueUserID, status string
    var currentWeek, totalWeeks int
    err := h.DB.QueryRow("SELECT user_id, status, day_number, total_weeks FROM leagues WHERE id=$1", leagueID).
        Scan(&leagueUserID, &status, &currentWeek, &totalWeeks)
    
    if err != nil {
        respondError(w, http.StatusNotFound, "League not found")
        return
    }
    
    if leagueUserID != user.ID {
        respondError(w, http.StatusForbidden, "Not your league")
        return
    }
    
    // VALIDATE: Check week
    if currentWeek < totalWeeks {
        respondError(w, http.StatusBadRequest, fmt.Sprintf("League not complete. Week %d of %d", currentWeek, totalWeeks))
        return
    }
    
    // VALIDATE: Check matches
    var pendingMatches int
    h.DB.QueryRow("SELECT COUNT(*) FROM daily_matches WHERE league_id=$1 AND status != 'COMPLETED'", leagueID).Scan(&pendingMatches)
    
    if pendingMatches > 0 {
        respondError(w, http.StatusBadRequest, fmt.Sprintf("%d matches still in progress", pendingMatches))
        return
    }
    
    // Award $1000 prize
    h.DB.Exec("UPDATE wallets SET kash = kash + 1000 WHERE user_id = $1", user.ID)
    
    // Validate winner bet
    h.validateWinnerBet(leagueID, user.ID)
    
    // Delete league
    h.deleteLeagueData(leagueID)
    
    h.WSHub.SendToUser(user.ID, "league_finished", map[string]interface{}{
        "message": "League finished! $1000 prize awarded!",
    })
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status": "finished",
        "prize": 1000,
    })
}



func (h *Handler) GetAvailableLeagues(w http.ResponseWriter, r *http.Request) {
    rows, _ := h.DB.Query(`
        SELECT l.id, l.name, l.type, l.difficulty, l.total_weeks,
               COUNT(ul.user_id) as user_count
        FROM leagues l
        LEFT JOIN user_leagues ul ON l.id = ul.league_id
        WHERE l.status = 'ACTIVE'
        GROUP BY l.id
        ORDER BY user_count DESC, l.name
    `)
    defer rows.Close()
    
    var leagues []map[string]interface{}
    for rows.Next() {
        var id, name, ltype, diff string
        var weeks, count int
        rows.Scan(&id, &name, &ltype, &diff, &weeks, &count)
        leagues = append(leagues, map[string]interface{}{
            "id": id, "name": name, "type": ltype,
            "difficulty": diff, "total_weeks": weeks,
            "user_count": count,
            "is_empty": count == 0,
            "is_popular": count >= 5,
        })
    }
    if leagues == nil { leagues = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, leagues)
}

func (h *Handler) GetMyLeagues(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    rows, _ := h.DB.Query(`
        SELECT l.id, l.name, l.type, l.difficulty, l.total_weeks, l.status, l.day_number
        FROM leagues l
        WHERE l.user_id = $1
        ORDER BY l.created_at DESC
    `, user.ID)
    defer rows.Close()
    
    var leagues []map[string]interface{}
    for rows.Next() {
        var id, name, ltype, diff, status string
        var weeks, day int
        rows.Scan(&id, &name, &ltype, &diff, &weeks, &status, &day)
        leagues = append(leagues, map[string]interface{}{
            "id": id, "name": name, "type": ltype,
            "difficulty": diff, "total_weeks": weeks,
            "status": status, "day_number": day,
        })
    }
    if leagues == nil { leagues = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, leagues)
}
// internal/handlers/league_handler.go

func (h *Handler) CreateLeague(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    var req struct {
        Name       string `json:"name"`
        TeamCount  int    `json:"team_count"`
        Difficulty string `json:"difficulty"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    // ===== VALIDATE NAME =====
    if req.Name == "" {
        req.Name = "My League"
    }
    req.Name = strings.TrimSpace(req.Name)
    if len(req.Name) > 30 {
        req.Name = req.Name[:30]
    }
    
    // ===== VALIDATE TEAM COUNT (5-20) =====
    if req.TeamCount < 5 {
        req.TeamCount = 5
    }
    if req.TeamCount > 20 {
        req.TeamCount = 20
    }
    if req.TeamCount == 0 {
        req.TeamCount = 10 // Default
    }
    
    // ===== VALIDATE DIFFICULTY =====
    validDifficulties := map[string]bool{
        "BEGINNER":     true,
        "EASY":         true,
        "NORMAL":       true,
        "CHALLENGING":  true,
        "HARD":         true,
        "EXPERT":       true,
        "MASTER":       true,
        "GRANDMASTER":  true,
    }
    if !validDifficulties[req.Difficulty] {
        req.Difficulty = "NORMAL"
    }
    
    // ===== CHECK EXISTING ACTIVE LEAGUE =====
    var count int
    h.DB.QueryRow("SELECT COUNT(*) FROM leagues WHERE user_id=$1 AND status='ACTIVE'", user.ID).Scan(&count)
    if count > 0 {
        respondError(w, http.StatusConflict, "You already have an active league. Complete it first.")
        return
    }
    
    // ===== CHECK WALLET =====
    var wallet struct{ Kash, Points, Coins float64 }
    err := h.DB.QueryRow("SELECT kash, points, coins FROM wallets WHERE user_id=$1", user.ID).Scan(&wallet.Kash, &wallet.Points, &wallet.Coins)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Wallet not found")
        return
    }
    
    // ===== CALCULATE LEAGUE COST =====
    leagueCost := calculateLeagueCost(req.TeamCount, req.Difficulty)
    
    if wallet.Kash < leagueCost {
        respondError(w, http.StatusBadRequest, fmt.Sprintf("Need %.0f Kash. You have %.0f", leagueCost, wallet.Kash))
        return
    }
    
    // ===== DEDUCT COST =====
    h.DB.Exec("UPDATE wallets SET kash = kash - $1 WHERE user_id = $2", leagueCost, user.ID)
    
    // ===== DETERMINE LEAGUE TYPE =====
    leagueType := getLeagueType(req.TeamCount)
    
    // ===== CALCULATE TOTAL WEEKS =====
    totalWeeks := (req.TeamCount - 1) * 2
    
    // ===== CREATE LEAGUE RECORD =====
    leagueID := "LEAGUE_" + uuid.New().String()[:12]
    _, err = h.DB.Exec(`INSERT INTO leagues (id, name, type, difficulty, total_weeks, user_id, status, day_number, current_week) 
               VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE', 1, 1)`,
        leagueID, req.Name, leagueType, req.Difficulty, totalWeeks, user.ID)
    if err != nil {
        h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", leagueCost, user.ID)
        respondError(w, http.StatusInternalServerError, "Failed to create league")
        return
    }
    
    // ===== GENERATE LEAGUE DATA =====
    gen := engine.NewUltimateGenerator()
    fullLeague := gen.GenerateFullLeagueCustom(req.Name, leagueType, req.Difficulty, req.TeamCount)
    
    if fullLeague == nil {
        h.rollbackLeagueCreation(leagueID, user.ID, leagueCost)
        respondError(w, http.StatusInternalServerError, "Failed to generate league")
        return
    }
    
    // ===== STORE TEAMS =====
    for _, teamData := range fullLeague.Teams {
        _, err := h.DB.Exec(`INSERT INTO teams (id, league_id, name, rating, formation, attack, midfield, defense, teamwork, experience, set_pieces)
                   VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
            teamData.Team.ID, leagueID, teamData.Team.Name, teamData.Team.Rating, teamData.Team.Formation,
            teamData.Team.Attack, teamData.Team.Midfield, teamData.Team.Defense,
            teamData.Team.Teamwork, teamData.Team.Experience, teamData.Team.SetPieces)
        if err != nil {
            h.rollbackLeagueCreation(leagueID, user.ID, leagueCost)
            respondError(w, http.StatusInternalServerError, "Failed to create teams")
            return
        }
        
        // ===== STORE COACH =====
        _, err = h.DB.Exec(`INSERT INTO coaches (id, league_id, team_id, name, nationality, formation, playstyle, mentality, attacking, defending, tactical, motivation, abilities)
                   VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
            teamData.Coach.ID, leagueID, teamData.Team.ID, teamData.Coach.Name, teamData.Coach.Nationality,
            teamData.Coach.Formation, teamData.Coach.PlayStyle, teamData.Coach.Mentality,
            teamData.Coach.Attacking, teamData.Coach.Defending, teamData.Coach.Tactical, teamData.Coach.Motivation,
            strings.Join(teamData.Coach.Abilities, ","))
        if err != nil {
            h.rollbackLeagueCreation(leagueID, user.ID, leagueCost)
            respondError(w, http.StatusInternalServerError, "Failed to create coaches")
            return
        }
        
        // ===== STORE PLAYERS =====
        for _, player := range teamData.Players {
            _, err = h.DB.Exec(`INSERT INTO players (id, league_id, team_id, name, position, number, rating, pace, shooting, passing, dribbling, defending, physical, stamina, form, traits)
                       VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
                player.ID, leagueID, teamData.Team.ID, player.Name, player.Position, player.Number, player.Rating,
                player.Pace, player.Shooting, player.Passing, player.Dribbling,
                player.Defending, player.Physical, player.Stamina, player.Form,
                strings.Join(player.Traits, ","))
            if err != nil {
                h.rollbackLeagueCreation(leagueID, user.ID, leagueCost)
                respondError(w, http.StatusInternalServerError, "Failed to create players")
                return
            }
        }
    }
    
    // ===== STORE FIXTURES =====
    for _, fix := range fullLeague.Fixtures {
        _, err = h.DB.Exec(`INSERT INTO fixtures (id, league_id, week_number, home_team_id, away_team_id, kickoff_time) 
                   VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
            fix.Fixture.ID, leagueID, fix.Fixture.WeekNumber, fix.Fixture.HomeTeamID, fix.Fixture.AwayTeamID, fix.Fixture.KickoffTime)
        if err != nil {
            h.rollbackLeagueCreation(leagueID, user.ID, leagueCost)
            respondError(w, http.StatusInternalServerError, "Failed to create fixtures")
            return
        }
    }
    
    // ===== GENERATE ODDS =====
    h.generateRandomOdds(leagueID)
    
    // ===== INITIALIZE LEAGUE TABLE =====
    for _, teamData := range fullLeague.Teams {
        _, err = h.DB.Exec(`INSERT INTO league_table (team_id, league_id, team_name) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
            teamData.Team.ID, leagueID, teamData.Team.Name)
        if err != nil {
            h.rollbackLeagueCreation(leagueID, user.ID, leagueCost)
            respondError(w, http.StatusInternalServerError, "Failed to create league table")
            return
        }
    }
    
    // ===== SCHEDULE WEEK 1 MATCHES =====
    h.scheduleDailyMatchesCustom(leagueID, 1, req.TeamCount)
    
    // ===== GET UPDATED WALLET =====
    var updatedWallet struct{ Kash, Points, Coins float64 }
    h.DB.Get(&updatedWallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", user.ID)
    
    // ===== RETURN SUCCESS =====
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "league_id":   leagueID,
        "status":      "created",
        "message":     fmt.Sprintf("League created! %d teams, %d weeks, %s difficulty", req.TeamCount, totalWeeks, req.Difficulty),
        "team_count":  req.TeamCount,
        "total_weeks": totalWeeks,
        "difficulty":  req.Difficulty,
        "league_type": leagueType,
        "cost":        leagueCost,
        "wallet": map[string]float64{
            "kash":   updatedWallet.Kash,
            "points": updatedWallet.Points,
            "coins":  updatedWallet.Coins,
        },
    })
}

// ===== HELPER FUNCTIONS =====

// getLeagueType determines league type from team count
func getLeagueType(teamCount int) string {
    switch {
    case teamCount <= 5:
        return "MINI"
    case teamCount <= 7:
        return "MICRO"
    case teamCount <= 10:
        return "MINOR"
    case teamCount <= 13:
        return "MAJOR"
    case teamCount <= 16:
        return "PREMIER"
    case teamCount <= 20:
        return "ELITE"
    default:
        return "CUSTOM"
    }
}

// calculateLeagueCost calculates total cost based on teams and difficulty
func calculateLeagueCost(teamCount int, difficulty string) float64 {
    baseCost := 500.0
    
    // Team count multiplier
    teamMultiplier := map[int]float64{
        5: 0.8, 6: 0.85, 7: 0.9, 8: 0.95, 9: 1.0, 10: 1.0,
        11: 1.1, 12: 1.2, 13: 1.3, 14: 1.4, 15: 1.5,
        16: 1.6, 17: 1.7, 18: 1.8, 19: 1.9, 20: 2.0,
    }
    
    // Difficulty multiplier
    difficultyMultiplier := map[string]float64{
        "BEGINNER": 0.8, "EASY": 0.9, "NORMAL": 1.0, "CHALLENGING": 1.1,
        "HARD": 1.2, "EXPERT": 1.4, "MASTER": 1.6, "GRANDMASTER": 2.0,
    }
    
    if tm, ok := teamMultiplier[teamCount]; ok {
        baseCost *= tm
    }
    
    if dm, ok := difficultyMultiplier[difficulty]; ok {
        baseCost *= dm
    }
    
    return math.Round(baseCost)
}

// rollbackLeagueCreation cleans up on failure
func (h *Handler) rollbackLeagueCreation(leagueID string, userID string, refundAmount float64) {
    h.DB.Exec("DELETE FROM players WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM coaches WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM teams WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM fixtures WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM match_odds WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM league_table WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM leagues WHERE id=$1", leagueID)
    h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", refundAmount, userID)
}

// scheduleDailyMatchesCustom schedules matches based on team count
func (h *Handler) scheduleDailyMatchesCustom(leagueID string, dayNumber int, teamCount int) {
    now := time.Now()
    firstKickoff := now.Add(5 * time.Minute)
    lastKickoff := now.Add(22 * time.Hour)
    
    // Number of matches per day = teamCount / 2, capped at 10
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
    
    if len(fixtureIDs) == 0 { return }
    
    totalWindow := lastKickoff.Sub(firstKickoff)
    interval := totalWindow / time.Duration(len(fixtureIDs)-1)
    
    for i, fixID := range fixtureIDs {
        kickoff := firstKickoff.Add(time.Duration(i) * interval)
        h.DB.Exec(`INSERT INTO daily_matches (league_id, day_number, fixture_id, kickoff_time, status)
                   VALUES ($1, $2, $3, $4, 'SCHEDULED')`,
            leagueID, dayNumber, fixID, kickoff)
    }
}

// generateRandomOdds replaces calculated odds with random varied ones from the 100-set pool
func (h *Handler) generateRandomOdds(leagueID string) {
    rows, err := h.DB.Query("SELECT id FROM fixtures WHERE league_id=$1", leagueID)
    if err != nil { return }
    defer rows.Close()
    
    // Delete old calculated odds
    h.DB.Exec("DELETE FROM match_odds WHERE league_id=$1", leagueID)
    
    for rows.Next() {
        var fixID string
        rows.Scan(&fixID)
        
        homeProb, drawProb, awayProb := engine.GetRandomProbability()
        
        oddID := "ODD_" + uuid.New().String()[:10]
        h.DB.Exec(`INSERT INTO match_odds (id, league_id, fixture_id, home_win, draw, away_win, home_odds, draw_odds, away_odds)
                   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
            oddID, leagueID, fixID,
            math.Round(homeProb*10000)/10000,
            math.Round(drawProb*10000)/10000,
            math.Round(awayProb*10000)/10000,
            math.Round((1.0/homeProb)*100)/100,
            math.Round((1.0/drawProb)*100)/100,
            math.Round((1.0/awayProb)*100)/100)
    }
}
func (h *Handler) GetFullLeague(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    
    var league struct {
        ID, Name, Type, Difficulty, Status string
        TotalWeeks, DayNumber int
    }
    err := h.DB.Get(&league, "SELECT id, name, type, difficulty, total_weeks, status, day_number FROM leagues WHERE id=$1", leagueID)
    if err != nil {
        respondError(w, http.StatusNotFound, "League not found")
        return
    }
    
    respondJSON(w, http.StatusOK, league)
}

func (h *Handler) DeleteLeague(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    if !user.IsAdmin {
        respondError(w, http.StatusForbidden, "Admin only")
        return
    }
    
    leagueID := chi.URLParam(r, "leagueID")
    
    h.DB.Exec("DELETE FROM daily_matches WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM match_results WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM match_odds WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM league_table WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM players WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM coaches WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM fixtures WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM teams WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM bets WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM leagues WHERE id=$1", leagueID)
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// func (h *Handler) scheduleDailyMatches(leagueID string, dayNumber int) {
//     now := time.Now()
//     firstKickoff := now.Add(5 * time.Minute)
//     lastKickoff := now.Add(22 * time.Hour)
    
//     rows, _ := h.DB.Query(`
//         SELECT f.id FROM fixtures f
//         WHERE f.league_id = $1 AND f.week_number = $2
//         ORDER BY f.id LIMIT 10
//     `, leagueID, dayNumber)
//     defer rows.Close()
    
//     var fixtureIDs []string
//     for rows.Next() {
//         var id string
//         rows.Scan(&id)
//         fixtureIDs = append(fixtureIDs, id)
//     }
    
//     if len(fixtureIDs) == 0 { return }
    
//     totalWindow := lastKickoff.Sub(firstKickoff)
//     interval := totalWindow / time.Duration(len(fixtureIDs)-1)
    
//     for i, fixID := range fixtureIDs {
//         kickoff := firstKickoff.Add(time.Duration(i) * interval)
//         h.DB.Exec(`INSERT INTO daily_matches (league_id, day_number, fixture_id, kickoff_time, status)
//                    VALUES ($1, $2, $3, $4, 'SCHEDULED')`,
//             leagueID, dayNumber, fixID, kickoff)
//     }
// }

func (h *Handler) GetLeagueTable(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    
    rows, _ := h.DB.Query(`
        SELECT team_id, team_name, played, won, drawn, lost, goals_for, goals_against, points
        FROM league_table WHERE league_id = $1
        ORDER BY points DESC, (goals_for - goals_against) DESC
    `, leagueID)
    defer rows.Close()
    
    var table []map[string]interface{}
    pos := 1
    for rows.Next() {
        var teamID, teamName string
        var played, won, drawn, lost, gf, ga, pts int
        rows.Scan(&teamID, &teamName, &played, &won, &drawn, &lost, &gf, &ga, &pts)
        table = append(table, map[string]interface{}{
            "position": pos, "team_id": teamID, "team_name": teamName,
            "played": played, "won": won, "drawn": drawn, "lost": lost,
            "goals_for": gf, "goals_against": ga, "goal_diff": gf - ga, "points": pts,
        })
        pos++
    }
    if table == nil { table = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, table)
}



func (h *Handler) GetWeekResults(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    
    rows, _ := h.DB.Query(`
        SELECT mr.id, mr.week_number, t1.name as home_team, t2.name as away_team,
               mr.home_score, mr.away_score, mr.winner,
               COALESCE(mr.goals::text, '[]') as goals,
               COALESCE(mr.stats::text, '{}') as stats,
               mr.created_at
        FROM match_results mr
        JOIN teams t1 ON mr.home_team_id = t1.id
        JOIN teams t2 ON mr.away_team_id = t2.id
        WHERE mr.league_id = $1
        ORDER BY mr.created_at DESC
    `, leagueID)
    defer rows.Close()
    
    var results []map[string]interface{}
    for rows.Next() {
        var id, home, away, winner, goalsJSON, statsJSON string
        var week, hs, as int
        var createdAt time.Time
        rows.Scan(&id, &week, &home, &away, &hs, &as, &winner, &goalsJSON, &statsJSON, &createdAt)
        
        var goals, stats interface{}
        json.Unmarshal([]byte(goalsJSON), &goals)
        json.Unmarshal([]byte(statsJSON), &stats)
        
        results = append(results, map[string]interface{}{
            "id": id, "week": week, 
            "home_team": home, "away_team": away,
            "home_score": hs, "away_score": as, "winner": winner,
            "goals": goals,
            "stats": stats,
            "created_at": createdAt.Format(time.RFC3339),
        })
    }
    if results == nil { results = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, results)
}

func (h *Handler) GetMatchProbabilities(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    week := r.URL.Query().Get("week")
    
    query := `
        SELECT mo.fixture_id, mo.home_win, mo.draw, mo.away_win, mo.home_odds, mo.draw_odds, mo.away_odds,
               t1.name as home_team, t2.name as away_team
        FROM match_odds mo
        JOIN fixtures f ON mo.fixture_id = f.id
        JOIN teams t1 ON f.home_team_id = t1.id
        JOIN teams t2 ON f.away_team_id = t2.id
        WHERE mo.league_id = $1
    `
    args := []interface{}{leagueID}
    if week != "" {
        query += " AND f.week_number = $2"
        args = append(args, week)
    }
    
    rows, _ := h.DB.Query(query, args...)
    defer rows.Close()
    
    var odds []map[string]interface{}
    for rows.Next() {
        var fixtureID, homeTeam, awayTeam string
        var homeWin, draw, awayWin, homeOdds, drawOdds, awayOdds float64
        rows.Scan(&fixtureID, &homeWin, &draw, &awayWin, &homeOdds, &drawOdds, &awayOdds, &homeTeam, &awayTeam)
        odds = append(odds, map[string]interface{}{
            "fixture_id": fixtureID, "home_team": homeTeam, "away_team": awayTeam,
            "probability": map[string]float64{"home": homeWin, "draw": draw, "away": awayWin},
            "odds": map[string]float64{"home": homeOdds, "draw": drawOdds, "away": awayOdds},
        })
    }
    if odds == nil { odds = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, odds)
}

func (h *Handler) GetMatchResults(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    
    rows, _ := h.DB.Query(`
        SELECT mr.id, mr.week_number, 
               t1.name as home_team, t2.name as away_team,
               mr.home_score, mr.away_score, mr.winner,
               COALESCE(mr.goals::text, '[]') as goals,
               COALESCE(mr.stats::text, '{}') as stats,
               mr.created_at
        FROM match_results mr
        JOIN teams t1 ON mr.home_team_id = t1.id
        JOIN teams t2 ON mr.away_team_id = t2.id
        WHERE mr.league_id = $1
        ORDER BY mr.created_at ASC
    `, leagueID)
    defer rows.Close()
    
    var results []map[string]interface{}
    for rows.Next() {
        var id, home, away, winner, goalsJSON, statsJSON string
        var week, hs, as int
        var createdAt time.Time
        rows.Scan(&id, &week, &home, &away, &hs, &as, &winner, &goalsJSON, &statsJSON, &createdAt)
        
        var goals, stats interface{}
        json.Unmarshal([]byte(goalsJSON), &goals)
        json.Unmarshal([]byte(statsJSON), &stats)
        
        results = append(results, map[string]interface{}{
            "id": id, "week": week,
            "home_team": home, "away_team": away,
            "home_score": hs, "away_score": as, "winner": winner,
            "goals": goals, "stats": stats,
        })
    }
    if results == nil { results = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, results)
}
// internal/handlers/league_handler.go - ADD THESE FUNCTIONS

// ===== LEAGUE WINNER BET =====
func (h *Handler) PlaceLeagueWinnerBet(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    leagueID := chi.URLParam(r, "leagueID")
    
    var req struct {
        TeamID      string `json:"team_id"`
        PointsRange string `json:"points_range"` // e.g., "70-80"
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Verify league belongs to user and is in week 1 (not started)
    var leagueUserID string
    var dayNumber int
    h.DB.QueryRow("SELECT user_id, day_number FROM leagues WHERE id=$1", leagueID).Scan(&leagueUserID, &dayNumber)
    
    if leagueUserID != user.ID {
        respondError(w, http.StatusForbidden, "League not found")
        return
    }
    
    if dayNumber > 1 {
        respondError(w, http.StatusBadRequest, "Winner bet only available before week 1 starts")
        return
    }
    
    // Check if already placed
    var count int
    h.DB.QueryRow("SELECT COUNT(*) FROM bets WHERE league_id=$1 AND user_id=$2 AND is_winner_bet=true", leagueID, user.ID).Scan(&count)
    if count > 0 {
        respondError(w, http.StatusBadRequest, "Already placed winner bet")
        return
    }
    
    // Get team count for prize calculation
    var teamCount int
    h.DB.QueryRow("SELECT COUNT(*) FROM teams WHERE league_id=$1", leagueID).Scan(&teamCount)
    
    // Prize: 5=$10K, 6=$50K, 7=$100K, 8=$150K, 10=$250K, 15=$500K, 20=$1M
    prize := calculateWinnerBetPrize(teamCount)
    
    // Create bet with special flag
    betID := "WBL_" + uuid.New().String()[:12]
    betsJSON, _ := json.Marshal([]map[string]interface{}{
        {
            "fixture_id": leagueID,
            "prediction": req.TeamID,
            "points_range": req.PointsRange,
        },
    })
    
    h.DB.Exec(`INSERT INTO bets (id, user_id, league_id, week, bets, amount, total_odds, is_custom, status, is_winner_bet, placed_at)
               VALUES ($1, $2, $3, 0, $4, 0, 0, false, 'PENDING', true, NOW())`,
        betID, user.ID, leagueID, betsJSON)
    
    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "status": "placed",
        "bet_id": betID,
        "prize": prize,
        "message": fmt.Sprintf("Winner bet placed! Win $%v if correct!", prize),
    })
}

// ===== CALCULATE WINNER BET PRIZE =====
func calculateWinnerBetPrize(teamCount int) float64 {
    switch {
    case teamCount <= 5:
        return 10000
    case teamCount == 6:
        return 50000
    case teamCount == 7:
        return 100000
    case teamCount == 8:
        return 150000
    case teamCount <= 10:
        return 250000
    case teamCount <= 15:
        return 500000
    default:
        return 1000000
    }
}




// ===== VALIDATE WINNER BET =====
func (h *Handler) validateWinnerBet(leagueID string, userID string) {
    // Get winner bet
    var betID, betsStr string
    h.DB.QueryRow("SELECT id, bets::text FROM bets WHERE league_id=$1 AND user_id=$2 AND is_winner_bet=true AND status='PENDING'", 
        leagueID, userID).Scan(&betID, &betsStr)
    
    if betID == "" {
        return // No winner bet
    }
    
    // Get actual league winner from table
    var winnerTeamID string
    h.DB.QueryRow("SELECT team_id FROM league_table WHERE league_id=$1 ORDER BY points DESC LIMIT 1", leagueID).Scan(&winnerTeamID)
    
    var predictions []map[string]interface{}
    json.Unmarshal([]byte(betsStr), &predictions)
    
    if len(predictions) > 0 {
        predictedTeamID := predictions[0]["prediction"].(string)
        
        if predictedTeamID == winnerTeamID {
            // WINNER!
            var teamCount int
            h.DB.QueryRow("SELECT COUNT(*) FROM teams WHERE league_id=$1", leagueID).Scan(&teamCount)
            prize := calculateWinnerBetPrize(teamCount)
            
            h.DB.Exec("UPDATE bets SET status='WON', payout=$1, settled_at=NOW() WHERE id=$2", prize, betID)
            h.DB.Exec("UPDATE wallets SET kash = kash + $1 WHERE user_id = $2", prize, userID)
            
            h.WSHub.SendToUser(userID, "winner_bet_won", map[string]interface{}{
                "prize": prize,
                "message": fmt.Sprintf("Winner bet correct! You won $%.0f!", prize),
            })
        } else {
            h.DB.Exec("UPDATE bets SET status='LOST', settled_at=NOW() WHERE id=$1", betID)
        }
    }
}

// ===== GET LEAGUE COMPLETION DATA =====
func (h *Handler) getLeagueCompletionData(leagueID string) map[string]interface{} {
    // Final table
    rows, _ := h.DB.Query("SELECT team_name, played, won, drawn, lost, points FROM league_table WHERE league_id=$1 ORDER BY points DESC LIMIT 3", leagueID)
    defer rows.Close()
    
    var topThree []map[string]interface{}
    for rows.Next() {
        var name string
        var played, won, drawn, lost, points int
        rows.Scan(&name, &played, &won, &drawn, &lost, &points)
        topThree = append(topThree, map[string]interface{}{
            "team_name": name,
            "played": played,
            "won": won,
            "drawn": drawn,
            "lost": lost,
            "points": points,
        })
    }
    
    // Top scorer
    var scorerName string
    var scorerGoals int
    h.DB.QueryRow("SELECT player_name, goals FROM top_scorers WHERE league_id=$1 ORDER BY goals DESC LIMIT 1", leagueID).Scan(&scorerName, &scorerGoals)
    
    return map[string]interface{}{
        "top_three": topThree,
        "top_scorer": map[string]interface{}{
            "name": scorerName,
            "goals": scorerGoals,
        },
    }
}





// For sqlx transactions
func (h *Handler) deleteLeagueDataTx(tx *sqlx.Tx, leagueID string) {
    tx.Exec("DELETE FROM top_scorers WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM league_table WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM match_odds WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM match_results WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM daily_matches WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM bets WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM quick_matches WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM players WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM coaches WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM fixtures WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM teams WHERE league_id=$1", leagueID)
    tx.Exec("DELETE FROM leagues WHERE id=$1", leagueID)
}

// For non-transaction (using DB directly)
func (h *Handler) deleteLeagueData(leagueID string) {
    h.DB.Exec("DELETE FROM top_scorers WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM league_table WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM match_odds WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM match_results WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM daily_matches WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM bets WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM quick_matches WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM players WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM coaches WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM fixtures WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM teams WHERE league_id=$1", leagueID)
    h.DB.Exec("DELETE FROM leagues WHERE id=$1", leagueID)
}

// Forfeit with transaction
func (h *Handler) ForfeitLeague(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    leagueID := chi.URLParam(r, "leagueID")
    
    if user == nil {
        respondError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    // Check if league exists
    var leagueUserID string
    err := h.DB.QueryRow("SELECT user_id FROM leagues WHERE id=$1", leagueID).Scan(&leagueUserID)
    
    if err != nil {
        respondJSON(w, http.StatusOK, map[string]interface{}{
            "status": "already_deleted",
            "message": "League already forfeited",
        })
        return
    }
    
    if leagueUserID != user.ID {
        respondError(w, http.StatusForbidden, "Not your league")
        return
    }
    
    // Start transaction
    tx, err := h.DB.Beginx()
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Transaction error")
        return
    }
    defer tx.Rollback()
    
    // Check wallet within transaction
    var kash float64
    err = tx.QueryRow("SELECT kash FROM wallets WHERE user_id=$1", user.ID).Scan(&kash)
    if err != nil {
        respondError(w, http.StatusBadRequest, "Wallet not found")
        return
    }
    
    if kash < 100 {
        respondError(w, http.StatusBadRequest, "Need $100 to forfeit")
        return
    }
    
    // Deduct penalty
    tx.Exec("UPDATE wallets SET kash = kash - 100 WHERE user_id = $1", user.ID)
    
    // Delete all data in transaction
    h.deleteLeagueDataTx(tx, leagueID)
    
    // Commit
    if err := tx.Commit(); err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to commit")
        return
    }
    
    h.WSHub.SendToUser(user.ID, "league_forfeited", map[string]interface{}{
        "message": "League forfeited. $100 penalty deducted.",
    })
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status": "forfeited",
        "penalty": 100,
    })
}