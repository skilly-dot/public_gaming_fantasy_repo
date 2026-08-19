package engine

import (
    "fmt"
    "math"
    "math/rand"
    "sort"
    "time"
    "github.com/google/uuid"
)

type UltimateGenerator struct {
    rng *rand.Rand
}

func NewUltimateGenerator() *UltimateGenerator {
    return &UltimateGenerator{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

type FullLeague struct {
    League   LeagueInfo       `json:"league"`
    Teams    []TeamWithDetails `json:"teams"`
    Fixtures []FixtureWithOdds `json:"fixtures"`
    Results  []MatchWithStats  `json:"results"`
    Table    []TableRow        `json:"table"`
    Scorers  []ScorerRow       `json:"scorers"`
}

type TeamWithDetails struct {
    Team    TeamData    `json:"team"`
    Coach   CoachData   `json:"coach"`
    Players []PlayerData `json:"players"`
}

type FixtureWithOdds struct {
    Fixture FixtureData `json:"fixture"`
    Odds    OddsData    `json:"odds"`
    Prob    ProbData    `json:"probability"`
}

type MatchWithStats struct {
    Result MatchResult `json:"result"`
    Goals  []GoalEvent `json:"goals"`
    Stats  MatchStats  `json:"stats"`
}

// internal/engine/generator.go - Add custom generation



// GenerateFullLeague creates everything
func (g *UltimateGenerator) GenerateFullLeague(name, leagueType, difficulty string) *FullLeague {
    if difficulty == "" { difficulty = "NORMAL" }
    
    config := DifficultyLevels[difficulty]
    lt := LeagueTypes[leagueType]
    leagueID := "LEAGUE_" + uuid.New().String()[:12]
    teamsCount := lt.TeamsCount
    totalWeeks := lt.TotalWeeks
    playersPerTeam := lt.PlayersPerTeam

    fmt.Printf("🎲 Generating: %s (%s, %s) - %d teams\n", name, leagueType, difficulty, teamsCount)

    // STEP 1: Pick random teams, coaches, stats from pools
    teamNames := g.pickRandom(TeamNames, teamsCount)
    coaches := g.pickCoaches(teamsCount)
    statSets := g.pickStats(teamsCount, config)

    // STEP 2: Create teams with ratings based on difficulty spread
    var allTeams []TeamWithDetails
    
    for i := 0; i < teamsCount; i++ {
        rating := 5.0 - (float64(i) * config.RatingSpread / float64(teamsCount))
        if rating < 1.0 { rating = 1.0 }
        
        team := TeamData{
            ID: "TEAM_"+uuid.New().String()[:8], Name: teamNames[i],
            Rating: math.Round(rating*100)/100, Formation: coaches[i].Formation,
            Attack: statSets[i].Attack, Midfield: statSets[i].Midfield,
            Defense: statSets[i].Defense, Teamwork: statSets[i].Teamwork,
            Experience: statSets[i].Experience, SetPieces: statSets[i].SetPieces,
        }
        
        coach := CoachData{
            ID: "COACH_"+uuid.New().String()[:8], Name: coaches[i].Name,
            Nationality: coaches[i].Nationality, Formation: coaches[i].Formation,
            PlayStyle: coaches[i].PlayStyle, Mentality: coaches[i].Mentality,
            Attacking: coaches[i].Attacking, Defending: coaches[i].Defending,
            Tactical: coaches[i].Tactical, Motivation: coaches[i].Motivation,
            Abilities: coaches[i].Abilities,
        }
        
        // Generate players for this team
        players := g.generatePlayers(team.ID, leagueID, team.Rating, playersPerTeam)
        
        allTeams = append(allTeams, TeamWithDetails{
            Team: team, Coach: coach, Players: players,
        })
    }

    // STEP 3: Generate fixtures (round-robin schedule)
    fixtures := g.generateFixtures(leagueID, allTeams, totalWeeks)
    
    // STEP 4: Calculate odds for each fixture based on team + coach strength
    fixturesWithOdds := g.calculateAllOdds(fixtures, allTeams, config)
    
    // STEP 5: Simulate all matches with detailed stats
    results, tableData, scorerData := g.simulateAllMatches(fixtures, allTeams, config)

    return &FullLeague{
        League:   LeagueInfo{ID: leagueID, Name: name, Type: leagueType, Difficulty: difficulty, TotalWeeks: totalWeeks},
        Teams:    allTeams,
        Fixtures: fixturesWithOdds,
        Results:  results,
        Table:    tableData,
        Scorers:  scorerData,
    }
}

// Calculate odds for all fixtures
func (g *UltimateGenerator) calculateAllOdds(fixtures []FixtureData, teams []TeamWithDetails, config DifficultyConfig) []FixtureWithOdds {
    var result []FixtureWithOdds
    
    for _, f := range fixtures {
        home := g.findTeam(teams, f.HomeTeamID)
        away := g.findTeam(teams, f.AwayTeamID)
        
        // Calculate probability based on team + coach strength
        homeStr := float64(home.Team.Attack+home.Team.Midfield)/2 + float64(home.Coach.Attacking)/10
        awayStr := float64(away.Team.Attack+away.Team.Midfield)/2 + float64(away.Coach.Attacking)/10
        homeDef := float64(home.Team.Defense) + float64(home.Coach.Defending)/10
        awayDef := float64(away.Team.Defense) + float64(away.Coach.Defending)/10
        
        homeWin := 0.35 + (homeStr-awayStr)/200 + (homeDef-awayDef)/300 + config.HomeAdvantage
        awayWin := 0.30 + (awayStr-homeStr)/200 + (awayDef-homeDef)/300 - config.HomeAdvantage
        
        homeWin += (g.rng.Float64()-0.5)*0.08
        awayWin += (g.rng.Float64()-0.5)*0.06
        
        homeWin = math.Max(0.12, math.Min(0.78, homeWin))
        awayWin = math.Max(0.08, math.Min(0.65, awayWin))
        draw := 1.0 - homeWin - awayWin
        draw = math.Max(0.15, math.Min(0.35, draw))
        
        total := homeWin + draw + awayWin
        prob := ProbData{HomeWin: r2(homeWin/total), Draw: r2(draw/total), AwayWin: r2(awayWin/total)}
        odds := OddsData{Home: r2(1.06/prob.HomeWin), Draw: r2(1.06/prob.Draw), Away: r2(1.06/prob.AwayWin)}
        
        result = append(result, FixtureWithOdds{Fixture: f, Odds: odds, Prob: prob})
    }
    return result
}

// Simulate all matches with detailed stats
func (g *UltimateGenerator) simulateAllMatches(fixtures []FixtureData, teams []TeamWithDetails, config DifficultyConfig) ([]MatchWithStats, []TableRow, []ScorerRow) {
    tableMap := make(map[string]*TableRow)
    scorerMap := make(map[string]*ScorerRow)
    
    for _, t := range teams {
        tableMap[t.Team.ID] = &TableRow{TeamID: t.Team.ID, TeamName: t.Team.Name}
    }
    
    var results []MatchWithStats
    for _, f := range fixtures {
        home := g.findTeam(teams, f.HomeTeamID)
        away := g.findTeam(teams, f.AwayTeamID)
        
        // Simulate match
        match := g.simulateMatch(home, away, f, config)
        results = append(results, match)
        
        // Update table
        h := tableMap[f.HomeTeamID]
        a := tableMap[f.AwayTeamID]
        h.Played++; a.Played++
        h.GoalsFor += match.Result.HomeScore; h.GoalsAgainst += match.Result.AwayScore
        a.GoalsFor += match.Result.AwayScore; a.GoalsAgainst += match.Result.HomeScore
        
        switch match.Result.Winner {
        case "HOME": h.Won++; h.Points += 3; a.Lost++
        case "AWAY": a.Won++; a.Points += 3; h.Lost++
        case "DRAW": h.Drawn++; h.Points++; a.Drawn++; a.Points++
        }
        
        // Update scorers
        for _, goal := range match.Goals {
            if _, ok := scorerMap[goal.ScorerID]; !ok {
                scorerMap[goal.ScorerID] = &ScorerRow{PlayerID: goal.ScorerID, Name: goal.Scorer}
            }
            scorerMap[goal.ScorerID].Goals++
        }
    }
    
    // Sort table
    var table []TableRow
    for _, t := range tableMap { table = append(table, *t) }
    sort.Slice(table, func(i, j int) bool {
        if table[i].Points != table[j].Points { return table[i].Points > table[j].Points }
        return (table[i].GoalsFor-table[i].GoalsAgainst) > (table[j].GoalsFor-table[j].GoalsAgainst)
    })
    
    // Sort scorers
    var scorers []ScorerRow
    for _, s := range scorerMap { scorers = append(scorers, *s) }
    sort.Slice(scorers, func(i, j int) bool { return scorers[i].Goals > scorers[j].Goals })
    
    return results, table, scorers
}

// Simulate single match
func (g *UltimateGenerator) simulateMatch(home, away TeamWithDetails, f FixtureData, config DifficultyConfig) MatchWithStats {
    homeStr := (float64(home.Team.Attack)+float64(home.Coach.Attacking))/2 * config.ScoringMultiplier
    awayStr := (float64(away.Team.Attack)+float64(away.Coach.Attacking))/2 * config.ScoringMultiplier
    homeDef := (float64(home.Team.Defense)+float64(home.Coach.Defending))/2
    awayDef := (float64(away.Team.Defense)+float64(away.Coach.Defending))/2
    
    hg := g.poisson(homeStr/(awayDef*0.7))
    ag := g.poisson(awayStr/(homeDef*0.7))
    
    // Determine winner
    roll := g.rng.Float64()
    var winner string
    homeWinChance := 0.35 + (homeStr-awayStr)/200 + config.HomeAdvantage
    if roll < homeWinChance {
        if hg <= ag { hg = ag + 1 + g.rng.Intn(2) }
        winner = "HOME"
    } else if roll < homeWinChance+0.25 {
        g2 := (hg+ag)/2; hg, ag = g2, g2
        winner = "DRAW"
    } else {
        if ag <= hg { ag = hg + 1 + g.rng.Intn(2) }
        winner = "AWAY"
    }
    
    if hg > 8 { hg = 8 }; if ag > 8 { ag = 8 }
    
    // Generate goals
    goals := g.generateMatchGoals(home, away, hg, ag)
    
    // Generate stats
    stats := MatchStats{
        PossessionHome: 45+g.rng.Intn(15),
        ShotsOnTargetHome: hg+g.rng.Intn(6), ShotsOnTargetAway: ag+g.rng.Intn(5),
        CornersHome: 2+g.rng.Intn(10), CornersAway: 1+g.rng.Intn(8),
        YellowCardsHome: g.rng.Intn(4), YellowCardsAway: g.rng.Intn(4),
        RedCardsHome: int(g.rng.Float64()*0.15), RedCardsAway: int(g.rng.Float64()*0.12),
    }
    stats.PossessionAway = 100 - stats.PossessionHome
    
    return MatchWithStats{
        Result: MatchResult{
            ID: fmt.Sprintf("%s_%s_W%d", f.HomeTeamID, f.AwayTeamID, f.WeekNumber),
            LeagueID:f.LeagueID, HomeTeamID: f.HomeTeamID, AwayTeamID: f.AwayTeamID,
            WeekNumber: f.WeekNumber, HomeScore: hg, AwayScore: ag, Winner: winner,
        },
        Goals: goals, Stats: stats,
    }
}

func (g *UltimateGenerator) generateMatchGoals(home, away TeamWithDetails, hg, ag int) []GoalEvent {
    var goals []GoalEvent
    for i := 0; i < hg; i++ {
        scorer := g.pickScorer(home.Players)
        goals = append(goals, GoalEvent{
            Minute: 5+g.rng.Intn(85), Team: "HOME",
            ScorerID: scorer.ID, Scorer: scorer.Name,
            Type: g.randomGoalType(),
        })
    }
    for i := 0; i < ag; i++ {
        scorer := g.pickScorer(away.Players)
        goals = append(goals, GoalEvent{
            Minute: 5+g.rng.Intn(85), Team: "AWAY",
            ScorerID: scorer.ID, Scorer: scorer.Name,
            Type: g.randomGoalType(),
        })
    }
    sort.Slice(goals, func(i, j int) bool { return goals[i].Minute < goals[j].Minute })
    return goals
}

func (g *UltimateGenerator) pickScorer(players []PlayerData) PlayerData {
    forwards := []PlayerData{}; mids := []PlayerData{}; defs := []PlayerData{}
    for _, p := range players {
        switch p.Position {
        case "FWD": forwards = append(forwards, p)
        case "MID": mids = append(mids, p)
        case "DEF": defs = append(defs, p)
        }
    }
    roll := g.rng.Float64()
    switch {
    case roll < 0.60 && len(forwards) > 0: return forwards[g.rng.Intn(len(forwards))]
    case roll < 0.90 && len(mids) > 0: return mids[g.rng.Intn(len(mids))]
    case len(defs) > 0: return defs[g.rng.Intn(len(defs))]
    default: return players[g.rng.Intn(len(players))]
    }
}

func (g *UltimateGenerator) randomGoalType() string {
    types := []string{"open_play","header","penalty","free_kick","long_range","counter_attack"}
    return types[g.rng.Intn(len(types))]
}

func (g *UltimateGenerator) generatePlayers(teamID, leagueID string, teamRating float64, count int) []PlayerData {
    positions := make([]string, count)
    gk := 2; def := count*6/18; mid := count*6/18; fwd := count*4/18
    idx := 0
    for i := 0; i < gk; i++ { positions[idx] = "GK"; idx++ }
    for i := 0; i < def; i++ { positions[idx] = "DEF"; idx++ }
    for i := 0; i < mid; i++ { positions[idx] = "MID"; idx++ }
    for i := 0; i < fwd; i++ { positions[idx] = "FWD"; idx++ }
    
    players := make([]PlayerData, count)
    for i := 0; i < count; i++ {
        name := FirstNames[g.rng.Intn(len(FirstNames))] + " " + LastNames[g.rng.Intn(len(LastNames))]
        rating := teamRating*8 + 50 + g.rng.Float64()*10
	if rating > 92 { rating = 92 }
	if rating < 55 { rating = 55 }
	if rating > 95 { rating = 95 }
	if rating < 50 { rating = 50 }
        traits := g.randomTraits(positions[i])
        
        players[i] = PlayerData{
            ID: "PLAYER_"+uuid.New().String()[:8], TeamID: teamID, LeagueID: leagueID,
            Name: name, Position: positions[i], Number: i+1,
            Rating: math.Round(rating*10)/10,
            Form: 5.0 + g.rng.Float64()*5.0, Traits: traits,
        }
        
        // Position-specific stats
        switch positions[i] {
        case "GK":
            players[i].Pace=40+g.rng.Intn(20); players[i].Shooting=15+g.rng.Intn(15)
            players[i].Passing=35+g.rng.Intn(30); players[i].Dribbling=20+g.rng.Intn(20)
            players[i].Defending=30+g.rng.Intn(30); players[i].Physical=60+g.rng.Intn(30)
        case "DEF":
            players[i].Pace=55+g.rng.Intn(30); players[i].Shooting=35+g.rng.Intn(30)
            players[i].Passing=55+g.rng.Intn(30); players[i].Dribbling=45+g.rng.Intn(30)
            players[i].Defending=65+g.rng.Intn(30); players[i].Physical=65+g.rng.Intn(30)
        case "MID":
            players[i].Pace=60+g.rng.Intn(30); players[i].Shooting=55+g.rng.Intn(30)
            players[i].Passing=65+g.rng.Intn(30); players[i].Dribbling=60+g.rng.Intn(30)
            players[i].Defending=45+g.rng.Intn(30); players[i].Physical=55+g.rng.Intn(30)
        case "FWD":
            players[i].Pace=65+g.rng.Intn(30); players[i].Shooting=65+g.rng.Intn(30)
            players[i].Passing=50+g.rng.Intn(30); players[i].Dribbling=65+g.rng.Intn(30)
            players[i].Defending=20+g.rng.Intn(20); players[i].Physical=55+g.rng.Intn(30)
        }
        players[i].Stamina = 50 + g.rng.Intn(40)
    }
    return players
}

// Helpers
func (g *UltimateGenerator) pickRandom(pool []string, count int) []string {
    shuffled := make([]string, len(pool))
    copy(shuffled, pool)
    g.rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
    return shuffled[:count]
}

func (g *UltimateGenerator) pickCoaches(count int) []CoachPreset {
    shuffled := make([]CoachPreset, len(CoachPool))
    copy(shuffled, CoachPool)
    g.rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
    return shuffled[:count]
}

func (g *UltimateGenerator) pickStats(count int, config DifficultyConfig) []TeamStatSet {
    shuffled := make([]TeamStatSet, len(TeamStatPool))
    copy(shuffled, TeamStatPool)
    g.rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
    result := shuffled[:count]
    sort.Slice(result, func(i, j int) bool {
        return (result[i].Attack+result[i].Midfield+result[i].Defense) > (result[j].Attack+result[j].Midfield+result[j].Defense)
    })
    return result
}

func (g *UltimateGenerator) generateFixtures(leagueID string, teams []TeamWithDetails, weeks int) []FixtureData {
    n := len(teams)
    fixtures := make([]FixtureData, 0, weeks*n/2)
    weekStart := time.Now().Unix() * 1000
    
    for week := 1; week <= weeks; week++ {
        // Create rotated copy for this week
        rotated := make([]TeamWithDetails, n)
        copy(rotated, teams)
        
        // Rotate teams (except first one - standard round-robin)
        shift := (week - 1) % (n - 1)
        for i := 1; i < n; i++ {
            newPos := 1 + (i-1+shift)%(n-1)
            rotated[newPos] = teams[i]
        }
        
        // Pair adjacent teams
        for i := 0; i < n; i += 2 {
            if i+1 < n {
                home := rotated[i]
                away := rotated[i+1]
                
                // Second half of season: swap home/away (second leg)
                if week > weeks/2 {
                    home, away = away, home
                }
                
                kickoff := weekStart + int64(week-1)*86400000 + int64(i/2)*7200000
                fixtures = append(fixtures, FixtureData{
                    ID: fmt.Sprintf("FIX_%s_%s_W%d", home.Team.ID, away.Team.ID, week),
                    LeagueID: leagueID, WeekNumber: week,
                    HomeTeamID: home.Team.ID, AwayTeamID: away.Team.ID,
                    KickoffTime: kickoff,
                })
            }
        }
    }
    return fixtures
}
func (g *UltimateGenerator) findTeam(teams []TeamWithDetails, id string) TeamWithDetails {
    for _, t := range teams { if t.Team.ID == id { return t } }
    return TeamWithDetails{}
}

func (g *UltimateGenerator) randomTraits(pos string) []string {
    t := map[string][]string{
        "GK": {"Diving Specialist","Penalty Stopper","Sweeper Keeper","Leader"},
        "DEF": {"Tackling Master","Aerial Threat","Ball Player","Speedster"},
        "MID": {"Playmaker","Box-to-Box","Deep Lying","Free Kick Specialist"},
        "FWD": {"Clinical Finisher","Speed Demon","Target Man","Dribbler"},
    }
    pool := t[pos]
    count := 2 + g.rng.Intn(2)
    result := make([]string, count)
    for i := 0; i < count; i++ { result[i] = pool[g.rng.Intn(len(pool))] }
    return result
}

func (g *UltimateGenerator) poisson(lambda float64) int {
    L := math.Exp(-lambda); k := 0; p := 1.0
    for p > L { k++; p *= g.rng.Float64() }
    return k - 1
}

func r2(v float64) float64 { return math.Round(v*10000) / 10000 }

// Replace the rating calculation in generatePlayers with:
// rating := teamRating*12 + 45 + g.rng.Float64()*8  // Range: 45-95, most between 55-85
// GenerateFullLeagueCustom creates a league with custom team count and difficulty
func (g *UltimateGenerator) GenerateFullLeagueCustom(name, leagueType, difficulty string, teamCount int) *FullLeague {
    if difficulty == "" { 
        difficulty = "NORMAL" 
    }
    if teamCount < 5 { 
        teamCount = 5 
    }
    if teamCount > 20 { 
        teamCount = 20 
    }
    
    config := DifficultyLevels[difficulty]
    if config.Name == "" {
        config = DifficultyLevels["NORMAL"]
    }
    
    totalWeeks := (teamCount - 1) * 2
    playersPerTeam := 18
    
    leagueID := "LEAGUE_" + uuid.New().String()[:12]
    
    fmt.Printf("🎲 Generating Custom League: %s (%d teams, %s) - %d weeks\n", 
        name, teamCount, difficulty, totalWeeks)
    
    // STEP 1: Pick random teams from the 100-team pool
    teamNames := g.pickRandomFromPool(TeamNames, teamCount)
    coaches := g.pickCoaches(teamCount)
    statSets := g.pickStats(teamCount, config)
    
    // STEP 2: Create teams
    var allTeams []TeamWithDetails
    
    for i := 0; i < teamCount; i++ {
        rating := 5.0 - (float64(i) * config.RatingSpread / float64(teamCount))
        if rating < 1.0 { 
            rating = 1.0 
        }
        
        team := TeamData{
            ID: "TEAM_"+uuid.New().String()[:8], 
            Name: teamNames[i],
            Rating: math.Round(rating*100)/100, 
            Formation: coaches[i].Formation,
            Attack: statSets[i].Attack, 
            Midfield: statSets[i].Midfield,
            Defense: statSets[i].Defense, 
            Teamwork: statSets[i].Teamwork,
            Experience: statSets[i].Experience, 
            SetPieces: statSets[i].SetPieces,
        }
        
        coach := CoachData{
            ID: "COACH_"+uuid.New().String()[:8], 
            Name: coaches[i].Name,
            Nationality: coaches[i].Nationality, 
            Formation: coaches[i].Formation,
            PlayStyle: coaches[i].PlayStyle, 
            Mentality: coaches[i].Mentality,
            Attacking: coaches[i].Attacking, 
            Defending: coaches[i].Defending,
            Tactical: coaches[i].Tactical, 
            Motivation: coaches[i].Motivation,
            Abilities: coaches[i].Abilities,
        }
        
        players := g.generatePlayers(team.ID, leagueID, team.Rating, playersPerTeam)
        
        allTeams = append(allTeams, TeamWithDetails{
            Team: team, 
            Coach: coach, 
            Players: players,
        })
    }
    
    // STEP 3: Generate fixtures (round-robin)
    fixtures := g.generateFixturesCustom(leagueID, allTeams, totalWeeks)
    
    // STEP 4: Calculate odds
    fixturesWithOdds := g.calculateAllOdds(fixtures, allTeams, config)
    
    // STEP 5: Simulate all matches
    results, tableData, scorerData := g.simulateAllMatches(fixtures, allTeams, config)
    
    return &FullLeague{
        League:   LeagueInfo{
            ID: leagueID, 
            Name: name, 
            Type: leagueType, 
            Difficulty: difficulty, 
            TotalWeeks: totalWeeks,
        },
        Teams:    allTeams,
        Fixtures: fixturesWithOdds,
        Results:  results,
        Table:    tableData,
        Scorers:  scorerData,
    }
}

// pickRandomFromPool picks random unique names from a pool
func (g *UltimateGenerator) pickRandomFromPool(pool []string, count int) []string {
    if count > len(pool) {
        count = len(pool)
    }
    
    shuffled := make([]string, len(pool))
    copy(shuffled, pool)
    g.rng.Shuffle(len(shuffled), func(i, j int) { 
        shuffled[i], shuffled[j] = shuffled[j], shuffled[i] 
    })
    return shuffled[:count]
}

// generateFixturesCustom generates round-robin fixtures for custom team count
func (g *UltimateGenerator) generateFixturesCustom(leagueID string, teams []TeamWithDetails, weeks int) []FixtureData {
    n := len(teams)
    fixtures := make([]FixtureData, 0, weeks*n/2)
    weekStart := time.Now().Unix() * 1000
    
    for week := 1; week <= weeks; week++ {
        // Create rotated copy for this week
        rotated := make([]TeamWithDetails, n)
        copy(rotated, teams)
        
        // Rotate teams (except first one - standard round-robin)
        shift := (week - 1) % (n - 1)
        if n > 1 {
            for i := 1; i < n; i++ {
                newPos := 1 + (i-1+shift)%(n-1)
                rotated[newPos] = teams[i]
            }
        }
        
        // Pair adjacent teams
        for i := 0; i < n; i += 2 {
            if i+1 < n {
                home := rotated[i]
                away := rotated[i+1]
                
                // Second half of season: swap home/away
                if week > weeks/2 {
                    home, away = away, home
                }
                
                kickoff := weekStart + int64(week-1)*86400000 + int64(i/2)*7200000
                fixtures = append(fixtures, FixtureData{
                    ID: fmt.Sprintf("FIX_%s_%s_W%d", home.Team.ID, away.Team.ID, week),
                    LeagueID: leagueID, 
                    WeekNumber: week,
                    HomeTeamID: home.Team.ID, 
                    AwayTeamID: away.Team.ID,
                    KickoffTime: kickoff,
                })
            }
        }
    }
    return fixtures
}