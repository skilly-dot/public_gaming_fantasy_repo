// internal/handlers/state_handler.go
package handlers

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
    "github.com/go-chi/chi/v5"
)

type cacheEntry struct {
    data interface{}
    time time.Time
}

var stateCache = make(map[string]cacheEntry)
var cacheMu sync.RWMutex

// GET /api/v1/leagues/{leagueID}/state
func (h *Handler) GetLeagueState(w http.ResponseWriter, r *http.Request) {
    leagueID := chi.URLParam(r, "leagueID")
    
    // Check cache (3 second TTL - short for near real-time)
    cacheMu.RLock()
    if cached, ok := stateCache[leagueID]; ok {
        if time.Since(cached.time) < 3*time.Second {
            cacheMu.RUnlock()
            respondJSON(w, http.StatusOK, cached.data)
            return
        }
    }
    cacheMu.RUnlock()
    
    // Get current week
    var currentWeek int
    h.DB.QueryRow("SELECT COALESCE(day_number, 1) FROM leagues WHERE id=$1", leagueID).Scan(&currentWeek)
    
    // Single query - PostgreSQL executes subqueries in parallel
    rows, err := h.DB.Query(`
        SELECT 'table' as type, COALESCE(json_agg(row_to_json(t) ORDER BY t.points DESC, (t.goals_for - t.goals_against) DESC)::text, '[]') as data
        FROM league_table t WHERE t.league_id = $1
        UNION ALL
        SELECT 'results' as type, COALESCE(json_agg(row_to_json(r) ORDER BY r.week_number, r.fixture_id)::text, '[]') as data
        FROM (
            SELECT mr.*, t1.name as home_team, t2.name as away_team
            FROM match_results mr
            JOIN teams t1 ON mr.home_team_id = t1.id
            JOIN teams t2 ON mr.away_team_id = t2.id
            WHERE mr.league_id = $1
        ) r
        UNION ALL
        SELECT 'matches' as type, COALESCE(json_agg(row_to_json(m) ORDER BY m.kickoff_time)::text, '[]') as data
        FROM (
            SELECT dm.*, t1.name as home_team, t2.name as away_team
            FROM daily_matches dm
            JOIN fixtures f ON dm.fixture_id = f.id
            JOIN teams t1 ON f.home_team_id = t1.id
            JOIN teams t2 ON f.away_team_id = t2.id
            WHERE dm.league_id = $1
        ) m
        UNION ALL
        SELECT 'odds' as type, COALESCE(json_agg(row_to_json(o))::text, '[]') as data
        FROM (
            SELECT mo.*, t1.name as home_team, t2.name as away_team
            FROM match_odds mo
            JOIN fixtures f ON mo.fixture_id = f.id
            JOIN teams t1 ON f.home_team_id = t1.id
            JOIN teams t2 ON f.away_team_id = t2.id
            WHERE mo.league_id = $1 AND f.week_number = $2
        ) o
        UNION ALL
        SELECT 'teams' as type, COALESCE(json_agg(row_to_json(tm))::text, '[]') as data
        FROM (
            SELECT t.*, 
                   CASE WHEN c.name IS NOT NULL 
                        THEN json_build_object('name', c.name, 'playstyle', c.playstyle, 'mentality', c.mentality) 
                        ELSE NULL END as coach
            FROM teams t
            LEFT JOIN coaches c ON t.id = c.team_id
            WHERE t.league_id = $1
            ORDER BY t.rating DESC
        ) tm
        UNION ALL
        SELECT 'players' as type, COALESCE(json_agg(row_to_json(p) ORDER BY p.rating DESC)::text, '[]') as data
        FROM (
            SELECT p.*, t.name as team_name
            FROM players p
            JOIN teams t ON p.team_id = t.id
            WHERE p.league_id = $1
        ) p
        UNION ALL
        SELECT 'scorers' as type, COALESCE(json_agg(row_to_json(s) ORDER BY s.goals DESC)::text, '[]') as data
        FROM top_scorers s WHERE s.league_id = $1
    `, leagueID, currentWeek)
    
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Query failed")
        return
    }
    defer rows.Close()
    
    response := make(map[string]interface{})
    for rows.Next() {
        var typ, data string
        rows.Scan(&typ, &data)
        response[typ] = json.RawMessage(data)
    }
    
    // Cache
    cacheMu.Lock()
    stateCache[leagueID] = cacheEntry{data: response, time: time.Now()}
    cacheMu.Unlock()
    
    respondJSON(w, http.StatusOK, response)
}