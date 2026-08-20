package handlers

import (
	"encoding/json"
	"log"
	"time"
)

// ============================================
// STATE BUILDERS - Complete Data Assembly
// ============================================

// BuildUserState - Complete user data for WebSocket push
func (h *Handler) BuildUserState(userID string) map[string]interface{} {
	state := make(map[string]interface{})
	
	// 1. Wallet
	wallet := h.getWalletData(userID)
	state["wallet"] = wallet
	
	// 2. Active Bets
	activeBets := h.getActiveBetsData(userID)
	state["active_bets"] = activeBets
	
	// 3. Bet History (last 10)
	betHistory := h.getBetHistoryData(userID, 10)
	state["bet_history"] = betHistory
	
	// 4. Bet Stats
	betStats := h.getBetStatsData(userID)
	state["bet_stats"] = betStats
	
	// 5. Notifications (unread)
	notifications := h.getNotificationsData(userID)
	state["notifications"] = notifications
	
	// 6. Unread count
	var unreadCount int
	h.DB.Get(&unreadCount, "SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND is_read=false", userID)
	state["unread_notifications"] = unreadCount
	
	return state
}

// BuildLeagueState - Complete league data for WebSocket push
func (h *Handler) BuildLeagueState(leagueID string) map[string]interface{} {
	state := make(map[string]interface{})
	
	// 1. League info
	league := h.getLeagueInfo(leagueID)
	state["league"] = league
	
	// 2. Matches (today's)
	matches := h.getTodayMatches(leagueID)
	state["matches"] = matches
	
	// 3. Results
	results := h.getResultsData(leagueID)
	state["results"] = results
	
	// 4. League Table
	table := h.getCurrentTable(leagueID)
	state["table"] = table
	
	// 5. Top Scorers
	scorers := h.getScorersData(leagueID)
	state["top_scorers"] = scorers
	
	// 6. Odds
	week := h.getCurrentWeek(leagueID)
	odds := h.getWeekOdds(leagueID, week)
	state["odds"] = odds
	
	// 7. Teams
	teams := h.getTeamsForLeague(leagueID)
	state["teams"] = teams
	
	// 8. Players
	players := h.getPlayersForLeague(leagueID)
	state["players"] = players
	
	return state
}

// BuildFullState - Everything for initial load or fallback
// In state_builder.go - Update BuildFullState

func (h *Handler) BuildFullState(userID, leagueID string) map[string]interface{} {
	state := make(map[string]interface{})
	
	// User state
	userState := h.BuildUserState(userID)
	for k, v := range userState {
		state[k] = v
	}
	
	// My Leagues
	myLeagues := h.getMyLeaguesData(userID)
	state["my_leagues"] = myLeagues
	
	// League state (if exists)
	if leagueID != "" {
		leagueState := h.BuildLeagueState(leagueID)
		state["league"] = leagueState
	}
	
	// Available admin bets
	adminBets := h.getAvailableAdminBets()
	state["admin_bets"] = adminBets
	
	// Available 50-50 bets
	fiftyBets := h.getAvailableFiftyFiftyBets()
	state["fifty_fifty_bets"] = fiftyBets
	
	// Rankings
	rankings := h.getRankingsData()
	state["rankings"] = rankings
	
	return state
}

// Add this function to get user's leagues
func (h *Handler) getMyLeaguesData(userID string) []map[string]interface{} {
    rows, err := h.DB.Query(`
        SELECT l.id, l.name, l.type, l.difficulty, l.total_weeks, l.status, l.day_number
        FROM leagues l
        WHERE l.user_id = $1
        ORDER BY l.created_at DESC
    `, userID)
    
    if err != nil {
        return []map[string]interface{}{}
    }
    defer rows.Close()
    
    var leagues []map[string]interface{}
    for rows.Next() {
        var id, name, ltype, diff, status string
        var weeks, day int
        
        rows.Scan(&id, &name, &ltype, &diff, &weeks, &status, &day)
        
        leagues = append(leagues, map[string]interface{}{
            "id":          id,
            "name":        name,
            "type":        ltype,
            "difficulty":  diff,
            "total_weeks": weeks,
            "status":      status,
            "day_number":  day,
        })
    }
    
    if leagues == nil {
        leagues = []map[string]interface{}{}
    }
    return leagues
}
// ============================================
// INDIVIDUAL DATA GETTERS
// ============================================

func (h *Handler) getWalletData(userID string) map[string]interface{} {
	var wallet struct {
		Kash   float64 `db:"kash"`
		Points float64 `db:"points"`
		Coins  float64 `db:"coins"`
	}
	
	err := h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", userID)
	if err != nil {
		return map[string]interface{}{
			"kash": 0, "points": 0, "coins": 0,
		}
	}
	
	return map[string]interface{}{
		"kash":   wallet.Kash,
		"points": wallet.Points,
		"coins":  wallet.Coins,
	}
}

func (h *Handler) getActiveBetsData(userID string) []map[string]interface{} {
	rows, err := h.DB.Query(`
		SELECT id, league_id, week, bets::text, amount, total_odds, 
		       is_custom, is_quick_match, is_winner_bet, status, placed_at
		FROM bets 
		WHERE user_id=$1 AND status IN ('PENDING', 'LOCKED')
		ORDER BY placed_at DESC
	`, userID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	
	var bets []map[string]interface{}
	for rows.Next() {
		var id, leagueID, betsStr, status string
		var week int
		var amount, totalOdds float64
		var isCustom, isQuick, isWinner bool
		var placedAt time.Time
		
		rows.Scan(&id, &leagueID, &week, &betsStr, &amount, &totalOdds,
			&isCustom, &isQuick, &isWinner, &status, &placedAt)
		
		var betsData interface{}
		json.Unmarshal([]byte(betsStr), &betsData)
		
		bets = append(bets, map[string]interface{}{
			"id":            id,
			"league_id":     leagueID,
			"week":          week,
			"bets":          betsData,
			"amount":        amount,
			"total_odds":    totalOdds,
			"is_custom":     isCustom,
			"is_quick_match": isQuick,
			"is_winner_bet": isWinner,
			"status":        status,
			"placed_at":     placedAt,
		})
	}
	
	if bets == nil {
		bets = []map[string]interface{}{}
	}
	return bets
}

func (h *Handler) getBetHistoryData(userID string, limit int) []map[string]interface{} {
	rows, err := h.DB.Query(`
		SELECT id, league_id, week, bets::text, amount, total_odds,
		       is_custom, is_quick_match, is_winner_bet, status, 
		       payout, tax, points, coins, placed_at, settled_at
		FROM bets 
		WHERE user_id=$1 AND status IN ('WON', 'LOST')
		ORDER BY COALESCE(settled_at, placed_at) DESC 
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	
	var bets []map[string]interface{}
	for rows.Next() {
		var id, leagueID, betsStr, status string
		var week int
		var amount, totalOdds, payout, tax float64
		var isCustom, isQuick, isWinner bool
		var points, coins int
		var placedAt time.Time
		var settledAt *time.Time
		
		rows.Scan(&id, &leagueID, &week, &betsStr, &amount, &totalOdds,
			&isCustom, &isQuick, &isWinner, &status, &payout, &tax,
			&points, &coins, &placedAt, &settledAt)
		
		var betsData interface{}
		json.Unmarshal([]byte(betsStr), &betsData)
		
		bets = append(bets, map[string]interface{}{
			"id":            id,
			"league_id":     leagueID,
			"week":          week,
			"bets":          betsData,
			"amount":        amount,
			"total_odds":    totalOdds,
			"is_custom":     isCustom,
			"is_quick_match": isQuick,
			"is_winner_bet": isWinner,
			"status":        status,
			"payout":        payout,
			"tax":           tax,
			"points":        points,
			"coins":         coins,
			"placed_at":     placedAt,
			"settled_at":    settledAt,
		})
	}
	
	if bets == nil {
		bets = []map[string]interface{}{}
	}
	return bets
}

func (h *Handler) getBetStatsData(userID string) map[string]interface{} {
	var stats struct {
		TotalBets    int     `db:"total_bets"`
		Wins         int     `db:"wins"`
		Losses       int     `db:"losses"`
		TotalWagered float64 `db:"total_wagered"`
		TotalWon     float64 `db:"total_won"`
	}
	
	h.DB.Get(&stats, `
		SELECT 
			COUNT(*) as total_bets,
			COUNT(*) FILTER (WHERE status='WON') as wins,
			COUNT(*) FILTER (WHERE status='LOST') as losses,
			COALESCE(SUM(amount), 0) as total_wagered,
			COALESCE(SUM(payout), 0) as total_won
		FROM bets 
		WHERE user_id=$1 AND status IN ('WON', 'LOST')
	`, userID)
	
	winRate := 0.0
	if stats.TotalBets > 0 {
		winRate = float64(stats.Wins) / float64(stats.TotalBets) * 100
	}
	
	return map[string]interface{}{
		"total_bets":    stats.TotalBets,
		"wins":          stats.Wins,
		"losses":        stats.Losses,
		"win_rate":      winRate,
		"total_wagered": stats.TotalWagered,
		"total_won":     stats.TotalWon,
	}
}

func (h *Handler) getNotificationsData(userID string) []map[string]interface{} {
	rows, err := h.DB.Query(`
		SELECT id, title, message, type, is_read, created_at
		FROM notifications 
		WHERE user_id=$1 
		ORDER BY created_at DESC 
		LIMIT 20
	`, userID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	
	var notifications []map[string]interface{}
	for rows.Next() {
		var id int
		var title, message, ntype string
		var isRead bool
		var createdAt time.Time
		
		rows.Scan(&id, &title, &message, &ntype, &isRead, &createdAt)
		
		notifications = append(notifications, map[string]interface{}{
			"id":         id,
			"title":      title,
			"message":    message,
			"type":       ntype,
			"is_read":    isRead,
			"created_at": createdAt,
		})
	}
	
	if notifications == nil {
		notifications = []map[string]interface{}{}
	}
	return notifications
}

func (h *Handler) getLeagueInfo(leagueID string) map[string]interface{} {
	var league struct {
		ID         string `db:"id"`
		Name       string `db:"name"`
		Type       string `db:"type"`
		Difficulty string `db:"difficulty"`
		Status     string `db:"status"`
		DayNumber  int    `db:"day_number"`
		TotalWeeks int    `db:"total_weeks"`
	}
	
	err := h.DB.Get(&league, `
		SELECT id, name, type, difficulty, status, day_number, total_weeks
		FROM leagues WHERE id=$1
	`, leagueID)
	if err != nil {
		return map[string]interface{}{}
	}
	
	return map[string]interface{}{
		"id":          league.ID,
		"name":        league.Name,
		"type":        league.Type,
		"difficulty":  league.Difficulty,
		"status":      league.Status,
		"day_number":  league.DayNumber,
		"total_weeks": league.TotalWeeks,
	}
}

func (h *Handler) getResultsData(leagueID string) []map[string]interface{} {
	rows, err := h.DB.Query(`
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
		ORDER BY mr.created_at DESC
		LIMIT 20
	`, leagueID)
	if err != nil {
		return []map[string]interface{}{}
	}
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
			"id":         id,
			"week":       week,
			"home_team":  home,
			"away_team":  away,
			"home_score": hs,
			"away_score": as,
			"winner":     winner,
			"goals":      goals,
			"stats":      stats,
			"created_at": createdAt,
		})
	}
	
	if results == nil {
		results = []map[string]interface{}{}
	}
	return results
}

func (h *Handler) getScorersData(leagueID string) []map[string]interface{} {
	rows, err := h.DB.Query(`
		SELECT player_name, goals
		FROM top_scorers
		WHERE league_id = $1
		ORDER BY goals DESC
		LIMIT 20
	`, leagueID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	
	var scorers []map[string]interface{}
	position := 1
	for rows.Next() {
		var name string
		var goals int
		rows.Scan(&name, &goals)
		
		scorers = append(scorers, map[string]interface{}{
			"position": position,
			"name":     name,
			"goals":    goals,
		})
		position++
	}
	
	if scorers == nil {
		scorers = []map[string]interface{}{}
	}
	return scorers
}

// In state_builder.go - Update getAvailableAdminBets

func (h *Handler) getAvailableAdminBets() []map[string]interface{} {
    rows, err := h.DB.Query(`
        SELECT 
            amb.id,
            amb.title,
            amb.description,
            amb.status,
            am.id as match_id,
            am.match_index,
            am.home_team,
            am.away_team,
            am.home_odds,
            am.draw_odds,
            am.away_odds,
            am.status as match_status
        FROM admin_match_bets amb
        LEFT JOIN admin_matches am ON amb.id = am.admin_bet_id
        WHERE amb.status = 'OPEN' AND amb.deleted_at IS NULL
        ORDER BY amb.created_at DESC, am.match_index ASC
    `)
    if err != nil {
        return []map[string]interface{}{}
    }
    defer rows.Close()
    
    // Group matches by bet
    betMap := make(map[string]map[string]interface{})
    var betOrder []string
    
    for rows.Next() {
        var betID, title, desc, betStatus string
        var matchID, homeTeam, awayTeam, matchStatus string
        var matchIndex int
        var homeOdds, drawOdds, awayOdds float64
        
        rows.Scan(&betID, &title, &desc, &betStatus, 
            &matchID, &matchIndex, &homeTeam, &awayTeam,
            &homeOdds, &drawOdds, &awayOdds, &matchStatus)
        
        // Check if bet already in map
        if _, exists := betMap[betID]; !exists {
            betMap[betID] = map[string]interface{}{
                "id":          betID,
                "title":       title,
                "description": desc,
                "status":      betStatus,
                "matches":     []map[string]interface{}{},
            }
            betOrder = append(betOrder, betID)
        }
        
        // Add match to bet
        if matchID != "" {
            matches := betMap[betID]["matches"].([]map[string]interface{})
            matches = append(matches, map[string]interface{}{
                "fixture_id":  matchID,
                "match_index": matchIndex,
                "home_team":   homeTeam,
                "away_team":   awayTeam,
                "odds": map[string]float64{
                    "home": homeOdds,
                    "draw": drawOdds,
                    "away": awayOdds,
                },
                "status": matchStatus,
            })
            betMap[betID]["matches"] = matches
        }
    }
    
    // Convert map to slice
    var bets []map[string]interface{}
    for _, betID := range betOrder {
        bets = append(bets, betMap[betID])
    }
    
    if bets == nil {
        bets = []map[string]interface{}{}
    }
    return bets
}

// In state_builder.go - Update getAvailableFiftyFiftyBets

func (h *Handler) getAvailableFiftyFiftyBets() []map[string]interface{} {
    rows, err := h.DB.Query(`
        SELECT 
            id, 
            title, 
            description, 
            yes_odds, 
            no_odds, 
            status,
            expires_at,
            created_at
        FROM fifty_fifty_bets
        WHERE status = 'OPEN' AND deleted_at IS NULL
        ORDER BY created_at DESC
    `)
    if err != nil {
        return []map[string]interface{}{}
    }
    defer rows.Close()
    
    var bets []map[string]interface{}
    for rows.Next() {
        var id, title, desc, status string
        var yesOdds, noOdds float64
        var expiresAt, createdAt *time.Time
        
        rows.Scan(&id, &title, &desc, &yesOdds, &noOdds, &status, &expiresAt, &createdAt)
        
        bet := map[string]interface{}{
            "id":          id,
            "title":       title,
            "description": desc,
            "yes_odds":    yesOdds,
            "no_odds":     noOdds,
            "status":      status,
        }
        
        if expiresAt != nil {
            bet["expires_at"] = *expiresAt
        }
        
        bets = append(bets, bet)
    }
    
    if bets == nil {
        bets = []map[string]interface{}{}
    }
    return bets
}

func (h *Handler) getRankingsData() []map[string]interface{} {
	rows, err := h.DB.Query(`
		SELECT 
			u.gamename,
			u.username,
			COALESCE(w.kash, 0) as kash,
			COALESCE(w.points, 0) as points,
			COALESCE(SUM(CASE WHEN b.status = 'WON' THEN 1 ELSE 0 END), 0) as wins,
			COALESCE(SUM(CASE WHEN b.status = 'LOST' THEN 1 ELSE 0 END), 0) as losses,
			COALESCE(SUM(b.payout), 0) as total_winnings,
			ROW_NUMBER() OVER (ORDER BY COALESCE(w.kash, 0) DESC) as rank
		FROM users u
		LEFT JOIN wallets w ON u.id = w.user_id
		LEFT JOIN bets b ON u.id = b.user_id
		GROUP BY u.id, u.gamename, u.username, w.kash, w.points
		ORDER BY COALESCE(w.kash, 0) DESC
		LIMIT 50
	`)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	
	var rankings []map[string]interface{}
	for rows.Next() {
		var gamename, username string
		var kash, points, totalWinnings float64
		var wins, losses, rank int
		
		rows.Scan(&gamename, &username, &kash, &points, &wins, &losses, &totalWinnings, &rank)
		
		winRate := 0.0
		if wins+losses > 0 {
			winRate = float64(wins) / float64(wins+losses) * 100
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
	return rankings
}

// PushFullState - Helper to push complete state via WebSocket
func (h *Handler) PushFullState(userID, leagueID, eventType string, extraData map[string]interface{}) {
	state := h.BuildFullState(userID, leagueID)
	
	// Add event type and extra data
	message := map[string]interface{}{
		"event": eventType,
		"state": state,
	}
	
	for k, v := range extraData {
		message[k] = v
	}
	
	h.WSHub.SendToUser(userID, eventType, message)
	log.Printf("📤 Pushed full state to user %s (event: %s)", userID, eventType)
}

// PushUserState - Helper to push user-specific state
func (h *Handler) PushUserState(userID, eventType string, extraData map[string]interface{}) {
	state := h.BuildUserState(userID)
	
	message := map[string]interface{}{
		"event": eventType,
		"state": state,
	}
	
	for k, v := range extraData {
		message[k] = v
	}
	
	h.WSHub.SendToUser(userID, eventType, message)
	log.Printf("📤 Pushed user state to %s (event: %s)", userID, eventType)
}