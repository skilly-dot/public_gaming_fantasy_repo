// internal/handlers/daily_handler.go
package handlers

import (
	//"encoding/json"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5" 
	"github.com/google/uuid"
)

// GET /api/v1/leagues/{leagueID}/daily
// Returns all data needed for the current day - directly from DB, no Redis
func (h *Handler) GetDailyData(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    currentWeek := h.getCurrentWeek(leagueID)
    
    // Get TODAY'S matches from daily_matches table
    matches := h.getTodayMatches(leagueID)
    
    // Get current league table
    table := h.getCurrentTable(leagueID)
    
    // Get odds for current week
    odds := h.getWeekOdds(leagueID, currentWeek)
    
    // Get teams and players
    teams := h.getTeamsForLeague(leagueID)
    players := h.getPlayersForLeague(leagueID)
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "matches": matches,
        "table":   table,
        "odds":    odds,
        "teams":   teams,
        "players": players,
    })
}

// NEW - Get today's matches from daily_matches
func (h *Handler) getTodayMatches(leagueID string) []map[string]interface{} {
    rows, _ := h.DB.Query(`
        SELECT dm.id, dm.fixture_id, dm.kickoff_time, dm.status, 
               dm.home_score, dm.away_score, dm.winner,
               t1.name as home_team, t2.name as away_team
        FROM daily_matches dm
        JOIN fixtures f ON dm.fixture_id = f.id
        JOIN teams t1 ON f.home_team_id = t1.id
        JOIN teams t2 ON f.away_team_id = t2.id
        WHERE dm.league_id = $1
        ORDER BY dm.kickoff_time
    `, leagueID)
    defer rows.Close()
    
    var matches []map[string]interface{}
    for rows.Next() {
        var id int
        var fixtureID, status, homeTeam, awayTeam, winner string
        var homeScore, awayScore int
        var kickoff time.Time
        rows.Scan(&id, &fixtureID, &kickoff, &status, &homeScore, &awayScore, &winner, &homeTeam, &awayTeam)
        
        if status != "COMPLETED" {
            homeScore = -1
            awayScore = -1
            winner = ""
        }
        
        matches = append(matches, map[string]interface{}{
            "id": id, "fixture_id": fixtureID,
            "home_team": homeTeam, "away_team": awayTeam,
            "home_score": homeScore, "away_score": awayScore,
            "winner": winner, "status": status,
            "kickoff_time": kickoff.Format(time.RFC3339),
        })
    }
    if matches == nil { matches = []map[string]interface{}{} }
    return matches
}
// getDailyFixtures returns fixtures for the week with scores hidden for non-completed matches
func (h *Handler) getDailyFixtures(leagueID string, week int) []map[string]interface{} {
    rows, _ := h.DB.Query(`
        SELECT f.id, f.week_number, 
               t1.name as home_team, t2.name as away_team,
               t1.id as home_team_id, t2.id as away_team_id,
               COALESCE(f.home_score, -1) as home_score,
               COALESCE(f.away_score, -1) as away_score,
               COALESCE(f.winner, '') as winner,
               COALESCE(f.status, 'SCHEDULED') as status,
               COALESCE(f.kickoff_time, 0) as kickoff_time
        FROM fixtures f
        JOIN teams t1 ON f.home_team_id = t1.id
        JOIN teams t2 ON f.away_team_id = t2.id
        WHERE f.league_id = $1 AND f.week_number = $2
        ORDER BY f.id
    `, leagueID, week)
    defer rows.Close()
    
    var fixtures []map[string]interface{}
    for rows.Next() {
        var id, homeTeam, awayTeam, homeID, awayID, winner, status string
        var weekNum, homeScore, awayScore int
        var kickoffTime int64
        rows.Scan(&id, &weekNum, &homeTeam, &awayTeam, &homeID, &awayID, &homeScore, &awayScore, &winner, &status, &kickoffTime)
        
        // Hide scores for non-completed matches
        if status != "COMPLETED" {
            homeScore = -1
            awayScore = -1
            winner = ""
        }
        
        fixtures = append(fixtures, map[string]interface{}{
            "id": id, "week": weekNum,
            "home_team": homeTeam, "away_team": awayTeam,
            "home_team_id": homeID, "away_team_id": awayID,
            "home_score": homeScore, "away_score": awayScore,
            "winner": winner, "status": status,
            "kickoff_time": kickoffTime,
        })
    }
    if fixtures == nil { fixtures = []map[string]interface{}{} }
    return fixtures
}

// getCurrentTable returns league standings sorted by points
func (h *Handler) getCurrentTable(leagueID string) []map[string]interface{} {
    rows, _ := h.DB.Query(`
        SELECT lt.team_id, lt.team_name, lt.played, lt.won, lt.drawn, lt.lost,
               lt.goals_for, lt.goals_against, lt.points
        FROM league_table lt
        WHERE lt.league_id = $1
        ORDER BY lt.points DESC, (lt.goals_for - lt.goals_against) DESC
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
    return table
}

// In daily_handler.go - Update getWeekOdds

func (h *Handler) getWeekOdds(leagueID string, week int) []map[string]interface{} {
    rows, err := h.DB.Query(`
        SELECT 
            f.id as fixture_id,
            f.week_number,
            f.kickoff_time,
            t1.name as home_team,
            t2.name as away_team,
            mo.home_win,
            mo.draw,
            mo.away_win,
            mo.home_odds,
            mo.draw_odds,
            mo.away_odds
        FROM fixtures f
        JOIN teams t1 ON f.home_team_id = t1.id
        JOIN teams t2 ON f.away_team_id = t2.id
        LEFT JOIN match_odds mo ON f.id = mo.fixture_id
        WHERE f.league_id = $1 AND f.week_number = $2
        ORDER BY f.id
    `, leagueID, week)
    
    if err != nil {
        return []map[string]interface{}{}
    }
    defer rows.Close()
    
    var odds []map[string]interface{}
    for rows.Next() {
        var fixtureID string
        var weekNum int
        var kickoffTime int64
        var homeTeam, awayTeam string
        var homeWin, draw, awayWin, homeOdds, drawOdds, awayOdds float64
        
        err := rows.Scan(&fixtureID, &weekNum, &kickoffTime, &homeTeam, &awayTeam,
            &homeWin, &draw, &awayWin, &homeOdds, &drawOdds, &awayOdds)
        
        if err != nil {
            continue
        }
        
        odds = append(odds, map[string]interface{}{
            "fixture_id":   fixtureID,
            "week":         weekNum,
            "kickoff_time": kickoffTime,
            "home_team":    homeTeam,
            "away_team":    awayTeam,
            "probability": map[string]float64{
                "home": homeWin,
                "draw": draw,
                "away": awayWin,
            },
            "odds": map[string]float64{
                "home": homeOdds,
                "draw": drawOdds,
                "away": awayOdds,
            },
        })
    }
    
    if odds == nil {
        odds = []map[string]interface{}{}
    }
    return odds
}

// getTeamsForLeague returns all teams with coach data
func (h *Handler) getTeamsForLeague(leagueID string) []map[string]interface{} {
    rows, _ := h.DB.Query(`
        SELECT t.id, t.name, t.rating, t.formation,
               t.attack, t.midfield, t.defense, t.teamwork, t.experience, t.set_pieces,
               c.name as coach_name, c.playstyle, c.mentality
        FROM teams t
        LEFT JOIN coaches c ON t.id = c.team_id
        WHERE t.league_id = $1
        ORDER BY t.rating DESC
    `, leagueID)
    defer rows.Close()
    
    var teams []map[string]interface{}
    for rows.Next() {
        var id, name, formation, coachName, playstyle, mentality string
        var rating float64
        var attack, midfield, defense, teamwork, experience, setPieces int
        rows.Scan(&id, &name, &rating, &formation, &attack, &midfield, &defense, &teamwork, &experience, &setPieces, &coachName, &playstyle, &mentality)
        teams = append(teams, map[string]interface{}{
            "id": id, "name": name, "rating": rating, "formation": formation,
            "stats": map[string]int{
                "attack": attack, "midfield": midfield, "defense": defense,
                "teamwork": teamwork, "experience": experience, "set_pieces": setPieces,
            },
            "coach": map[string]string{
                "name": coachName, "playstyle": playstyle, "mentality": mentality,
            },
        })
    }
    if teams == nil { teams = []map[string]interface{}{} }
    return teams
}

// getPlayersForLeague returns all players with stats
func (h *Handler) getPlayersForLeague(leagueID string) []map[string]interface{} {
    rows, _ := h.DB.Query(`
        SELECT p.id, p.name, p.position, p.number, p.rating,
               p.pace, p.shooting, p.passing, p.dribbling,
               p.defending, p.physical, p.stamina, p.form, p.traits,
               t.name as team_name
        FROM players p
        JOIN teams t ON p.team_id = t.id
        WHERE p.league_id = $1
        ORDER BY p.rating DESC
    `, leagueID)
    defer rows.Close()
    
    var players []map[string]interface{}
    for rows.Next() {
        var id, name, position, teamName, traits string
        var number int
        var rating, form float64
        var pace, shooting, passing, dribbling, defending, physical, stamina int
        rows.Scan(&id, &name, &position, &number, &rating, &pace, &shooting, &passing, &dribbling, &defending, &physical, &stamina, &form, &traits, &teamName)
        players = append(players, map[string]interface{}{
            "id": id, "name": name, "position": position, "number": number,
            "rating": rating, "team_name": teamName,
            "stats": map[string]int{
                "pace": pace, "shooting": shooting, "passing": passing,
                "dribbling": dribbling, "defending": defending,
                "physical": physical, "stamina": stamina,
            },
            "form": form, "traits": traits,
        })
    }
    if players == nil { players = []map[string]interface{}{} }
    return players
}

func (h *Handler) simulateOneMatch(leagueID string, match struct {
    ID int
    FixtureID, HomeID, AwayID string
    WeekNum int
}) {
    homeScore := rand.Intn(4) + 1
    awayScore := rand.Intn(3)
    winner := "HOME"
    if awayScore > homeScore { winner = "AWAY" }
    if homeScore == awayScore { winner = "DRAW" }
    
    stats := map[string]interface{}{
        "possession": map[string]int{"home": 40+rand.Intn(30), "away": 0},
        "shots": map[string]int{"home": homeScore+rand.Intn(8), "away": awayScore+rand.Intn(6)},
        "cards": map[string]int{"home": rand.Intn(3), "away": rand.Intn(3)},
        "corners": map[string]int{"home": 2+rand.Intn(10), "away": 1+rand.Intn(8)},
    }
    stats["possession"].(map[string]int)["away"] = 100 - stats["possession"].(map[string]int)["home"]
    
    goals := []map[string]interface{}{}
    minute := 0
    for i := 0; i < homeScore; i++ {
        minute += rand.Intn(90/(homeScore+awayScore+1)) + 5
        if minute > 90 { minute = 90 }
        goals = append(goals, map[string]interface{}{"minute": minute, "team": "HOME"})
    }
    minute = 0
    for i := 0; i < awayScore; i++ {
        minute += rand.Intn(90/(homeScore+awayScore+1)) + 5
        if minute > 90 { minute = 90 }
        goals = append(goals, map[string]interface{}{"minute": minute, "team": "AWAY"})
    }
    
    goalsJSON, _ := json.Marshal(goals)
    statsJSON, _ := json.Marshal(stats)
    
    h.DB.Exec(`UPDATE daily_matches SET status='COMPLETED', home_score=$1, away_score=$2, winner=$3 WHERE id=$4`,
        homeScore, awayScore, winner, match.ID)
    h.DB.Exec(`UPDATE fixtures SET home_score=$1, away_score=$2, winner=$3, status='COMPLETED' WHERE id=$4`,
        homeScore, awayScore, winner, match.FixtureID)
    
    resultID := "RES_" + uuid.New().String()[:12]
    h.DB.Exec(`INSERT INTO match_results (id, league_id, fixture_id, week_number, home_team_id, away_team_id, home_score, away_score, winner, goals, stats)
               VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
        resultID, leagueID, match.FixtureID, match.WeekNum, match.HomeID, match.AwayID, homeScore, awayScore, winner, goalsJSON, statsJSON)
    
    h.DB.Exec(`UPDATE league_table SET played=played+1, goals_for=goals_for+$1, goals_against=goals_against+$2,
               won=won+CASE WHEN $3='HOME' THEN 1 ELSE 0 END, drawn=drawn+CASE WHEN $3='DRAW' THEN 1 ELSE 0 END, lost=lost+CASE WHEN $3='AWAY' THEN 1 ELSE 0 END,
               points=points+CASE WHEN $3='HOME' THEN 3 WHEN $3='DRAW' THEN 1 ELSE 0 END WHERE team_id=$4 AND league_id=$5`,
        homeScore, awayScore, winner, match.HomeID, leagueID)
    h.DB.Exec(`UPDATE league_table SET played=played+1, goals_for=goals_for+$1, goals_against=goals_against+$2,
               won=won+CASE WHEN $3='AWAY' THEN 1 ELSE 0 END, drawn=drawn+CASE WHEN $3='DRAW' THEN 1 ELSE 0 END, lost=lost+CASE WHEN $3='HOME' THEN 1 ELSE 0 END,
               points=points+CASE WHEN $3='AWAY' THEN 3 WHEN $3='DRAW' THEN 1 ELSE 0 END WHERE team_id=$4 AND league_id=$5`,
        awayScore, homeScore, winner, match.AwayID, leagueID)
}

func (h *Handler) settleAllBets(leagueID string) {
    rows, _ := h.DB.Query(`
        SELECT b.id, b.user_id, b.bets::text, b.amount
        FROM bets b WHERE b.league_id = $1 AND b.status = 'PENDING'
    `, leagueID)
    if rows == nil { return }
    defer rows.Close()
    
    for rows.Next() {
        var betID, userID, betsJSON string
        var amount float64
        rows.Scan(&betID, &userID, &betsJSON, &amount)
        
        var predictions []map[string]interface{}
        json.Unmarshal([]byte(betsJSON), &predictions)
        
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
                netPayout, int(amount*2), int(amount*0.5), userID)
        } else {
            h.DB.Exec(`UPDATE bets SET status='LOST', points=5, settled_at=NOW() WHERE id=$1`, betID)
            h.DB.Exec(`UPDATE wallets SET points=points+5 WHERE user_id=$1`, userID)
        }
    }
}