package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CleanupOldData(w http.ResponseWriter, r *http.Request) {
    r1, _ := h.DB.Exec("DELETE FROM match_results WHERE created_at < NOW() - INTERVAL '3 days'")
    rows, _ := r1.RowsAffected()
    respondJSON(w, http.StatusOK, map[string]interface{}{"deleted":rows,"message":"Old data cleaned"})
}



// GetLeagueTeams - Get all teams in a league
func (h *Handler) GetLeagueTeams(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    
    rows, err := h.DB.Query(`
        SELECT t.id, t.name, t.rating, t.formation, 
               t.attack, t.midfield, t.defense, 
               t.teamwork, t.experience, t.set_pieces,
               c.name as coach_name, c.playstyle, c.mentality
        FROM teams t
        LEFT JOIN coaches c ON t.id = c.team_id
        WHERE t.league_id = $1
        ORDER BY t.rating DESC
    `, leagueID)
    if err != nil {
        respondJSON(w, http.StatusOK, []interface{}{})
        return
    }
    defer rows.Close()
    
    var teams []map[string]interface{}
    for rows.Next() {
        var id, name, formation, coachName, playstyle, mentality string
        var rating float64
        var attack, midfield, defense, teamwork, experience, setPieces int
        
        rows.Scan(&id, &name, &rating, &formation, 
                  &attack, &midfield, &defense, 
                  &teamwork, &experience, &setPieces,
                  &coachName, &playstyle, &mentality)
        
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
    respondJSON(w, http.StatusOK, teams)
}

// GetLeaguePlayers - Get all players in a league
func (h *Handler) GetLeaguePlayers(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    teamID := r.URL.Query().Get("team_id") // Optional filter
    
    query := `
        SELECT p.id, p.name, p.position, p.number, p.rating,
               p.pace, p.shooting, p.passing, p.dribbling,
               p.defending, p.physical, p.stamina, p.form, p.traits,
               t.name as team_name
        FROM players p
        JOIN teams t ON p.team_id = t.id
        WHERE p.league_id = $1
    `
    args := []interface{}{leagueID}
    
    if teamID != "" {
        query += " AND p.team_id = $2"
        args = append(args, teamID)
    }
    
    query += " ORDER BY p.rating DESC"
    
    rows, err := h.DB.Query(query, args...)
    if err != nil {
        respondJSON(w, http.StatusOK, []interface{}{})
        return
    }
    defer rows.Close()
    
    var players []map[string]interface{}
    for rows.Next() {
        var id, name, position, teamName, traits string
        var number int
        var rating, form float64
        var pace, shooting, passing, dribbling, defending, physical, stamina int
        
        rows.Scan(&id, &name, &position, &number, &rating,
                  &pace, &shooting, &passing, &dribbling,
                  &defending, &physical, &stamina, &form, &traits,
                  &teamName)
        
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
    respondJSON(w, http.StatusOK, players)
}

// GetLeagueFixtures - Get upcoming fixtures
func (h *Handler) GetLeagueFixtures(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    week := r.URL.Query().Get("week")
    
    query := `
        SELECT f.id, f.week_number, f.kickoff_time,
               t1.name as home_team, t2.name as away_team,
               COALESCE(mr.home_score, -1) as home_score,
               COALESCE(mr.away_score, -1) as away_score,
               COALESCE(mr.winner, '') as winner
        FROM fixtures f
        JOIN teams t1 ON f.home_team_id = t1.id
        JOIN teams t2 ON f.away_team_id = t2.id
        LEFT JOIN match_results mr ON f.id = mr.fixture_id
        WHERE f.league_id = $1
    `
    args := []interface{}{leagueID}
    
    if week != "" {
        query += " AND f.week_number = $2"
        args = append(args, week)
    }
    
    query += " ORDER BY f.week_number, f.kickoff_time"
    
    rows, err := h.DB.Query(query, args...)
    if err != nil {
        respondJSON(w, http.StatusOK, []interface{}{})
        return
    }
    defer rows.Close()
    
    var fixtures []map[string]interface{}
    for rows.Next() {
        var id, homeTeam, awayTeam, winner string
        var weekNum, homeScore, awayScore int
        var kickoff int64
        
        rows.Scan(&id, &weekNum, &kickoff, &homeTeam, &awayTeam, 
                  &homeScore, &awayScore, &winner)
        
        played := homeScore >= 0
        
        fixtures = append(fixtures, map[string]interface{}{
            "id": id, "week": weekNum, "kickoff": kickoff,
            "home_team": homeTeam, "away_team": awayTeam,
            "played": played,
            "home_score": homeScore, "away_score": awayScore, "winner": winner,
        })
    }
    if fixtures == nil { fixtures = []map[string]interface{}{} }
    respondJSON(w, http.StatusOK, fixtures)
}

// MarkNotificationRead - Mark single notification as read
func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    notifID := chi.URLParam(r, "id")
    
    h.DB.Exec("UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2", notifID, user.ID)
    respondJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

// MarkAllNotificationsRead - Mark all notifications as read
func (h *Handler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    h.DB.Exec("UPDATE notifications SET is_read=true WHERE user_id=$1", user.ID)
    respondJSON(w, http.StatusOK, map[string]string{"status": "all_read"})
}