package engine

import (
    "fmt"
	//"sync"
    "math"
    "math/rand"
    "sort"
    "sync"
    "time"
)

const (
    WORKERS = 12        // Number of concurrent goroutines
    CHUNK_SIZE = 40     // Process 40 matches per batch
)

type LeagueProcessor struct {
    teams      []TeamWithDetails
    fixtures   []FixtureData
    results    []MatchWithStats
    table      map[string]*TableRow
    scorers    map[string]*ScorerRow
    odds       []FixtureWithOdds
    
    // Concurrency
    resultChan chan MatchWithStats
    oddsChan   chan FixtureWithOdds
    workerPool chan struct{}
    mu         sync.RWMutex
    wg         sync.WaitGroup
}

func NewLeagueProcessor(teams []TeamWithDetails, fixtures []FixtureData) *LeagueProcessor {
    return &LeagueProcessor{
        teams:      teams,
        fixtures:   fixtures,
        table:      make(map[string]*TableRow),
        scorers:    make(map[string]*ScorerRow),
        resultChan: make(chan MatchWithStats, 500),
        oddsChan:   make(chan FixtureWithOdds, 500),
        workerPool: make(chan struct{}, WORKERS),
    }
}

// ProcessAll runs odds calculation and match simulation concurrently
func (p *LeagueProcessor) ProcessAll(config DifficultyConfig) ([]FixtureWithOdds, []MatchWithStats, []TableRow, []ScorerRow) {
    start := time.Now()
    
    // Initialize table
    for _, t := range p.teams {
        p.table[t.Team.ID] = &TableRow{TeamID: t.Team.ID, TeamName: t.Team.Name}
    }
    
    // PHASE 1: Calculate odds for ALL fixtures (parallel)
    fmt.Println("📊 Phase 1: Calculating odds...")
    p.calculateAllOddsParallel(config)
    
    // PHASE 2: Simulate ALL matches (parallel)
    fmt.Println("⚽ Phase 2: Simulating matches...")
    p.simulateAllMatchesParallel(config)
    
    // PHASE 3: Build final table and scorers
    fmt.Println("📈 Phase 3: Building table...")
    table, scorers := p.buildFinalTable()
    
    elapsed := time.Since(start)
    fmt.Printf("✅ League processed in %v (%d matches, %d teams)\n", elapsed, len(p.fixtures), len(p.teams))
    
    return p.odds, p.results, table, scorers
}

// Phase 1: Calculate odds using ALL available data
func (p *LeagueProcessor) calculateAllOddsParallel(config DifficultyConfig) {
    total := len(p.fixtures)
    p.odds = make([]FixtureWithOdds, total)
    
    // Process in chunks to avoid memory issues
    for start := 0; start < total; start += CHUNK_SIZE {
        end := start + CHUNK_SIZE
        if end > total { end = total }
        
        chunk := p.fixtures[start:end]
        
        // Launch goroutines for this chunk
        for i, fixture := range chunk {
            p.wg.Add(1)
            go func(idx int, f FixtureData) {
                defer p.wg.Done()
                p.workerPool <- struct{}{}
                defer func() { <-p.workerPool }()
                
                odds := p.calculateSingleOdds(f, config)
                p.odds[start+idx] = odds
            }(i, fixture)
        }
        p.wg.Wait()
    }
}

func (p *LeagueProcessor) calculateSingleOdds(f FixtureData, config DifficultyConfig) FixtureWithOdds {
    // Use random probability from the 100-set pool instead of complex calculation
    homeProb, drawProb, awayProb := GetRandomProbability()
    
    prob := ProbData{
        HomeWin: r2(homeProb),
        Draw:    r2(drawProb),
        AwayWin: r2(awayProb),
    }
    
    odds := OddsData{
        Home: r2(1.06 / homeProb),
        Draw: r2(1.06 / drawProb),
        Away: r2(1.06 / awayProb),
    }
    
    return FixtureWithOdds{Fixture: f, Odds: odds, Prob: prob}
}
// Phase 2: Simulate matches using odds and team data
func (p *LeagueProcessor) simulateAllMatchesParallel(config DifficultyConfig) {
    total := len(p.fixtures)
    p.results = make([]MatchWithStats, total)
    
    for start := 0; start < total; start += CHUNK_SIZE {
        end := start + CHUNK_SIZE
        if end > total { end = total }
        
        chunk := p.fixtures[start:end]
        
        for i, fixture := range chunk {
            p.wg.Add(1)
            go func(idx int, f FixtureData) {
                defer p.wg.Done()
                p.workerPool <- struct{}{}
                defer func() { <-p.workerPool }()
                
                result := p.simulateSingleMatch(f, config)
                
                // Thread-safe update of table and scorers
                p.mu.Lock()
                p.results[start+idx] = result
                p.updateTableInMemory(result)
                p.updateScorersInMemoryFixed(result)
                p.mu.Unlock()
            }(i, fixture)
        }
        p.wg.Wait()
    }
}

func (p *LeagueProcessor) simulateSingleMatch(f FixtureData, config DifficultyConfig) MatchWithStats {
    home := p.getTeam(f.HomeTeamID)
    away := p.getTeam(f.AwayTeamID)
    odds := p.getOddsForFixture(f.ID)
    
    localRng := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    // Calculate expected goals using Poisson
    homeAttackStr := (float64(home.Team.Attack)*0.5 + float64(home.Coach.Attacking)*0.3) / 100
    awayDefStr := (float64(away.Team.Defense)*0.5 + float64(away.Coach.Defending)*0.3) / 100
    awayAttackStr := (float64(away.Team.Attack)*0.5 + float64(away.Coach.Attacking)*0.3) / 100
    homeDefStr := (float64(home.Team.Defense)*0.5 + float64(home.Coach.Defending)*0.3) / 100
    
    homeLambda := homeAttackStr * 3.0 * (1 - awayDefStr*0.5) * config.ScoringMultiplier
    awayLambda := awayAttackStr * 2.5 * (1 - homeDefStr*0.5) * config.ScoringMultiplier
    
    hg := p.poisson(homeLambda, localRng)
    ag := p.poisson(awayLambda, localRng)
    
    // Determine winner using odds
    //roll := localRng.Float64()
    var winner string
    if odds.Prob.HomeWin > odds.Prob.AwayWin && odds.Prob.HomeWin > odds.Prob.Draw {
        if hg <= ag { hg = ag + 1 + localRng.Intn(2) }
        winner = "HOME"
    } else if odds.Prob.AwayWin > odds.Prob.HomeWin && odds.Prob.AwayWin > odds.Prob.Draw {
        if ag <= hg { ag = hg + 1 + localRng.Intn(2) }
        winner = "AWAY"
    } else {
        // Draw likely
        if hg != ag {
            g2 := (hg + ag) / 2
            hg, ag = g2, g2
        }
        winner = "DRAW"
    }
    
    // Cap scores
    if hg > 8 { hg = 8 }
    if ag > 8 { ag = 8 }
    
    // Generate goals with real player names
    goals := p.generateGoals(f, hg, ag)
    
    // Generate match statistics
    stats := p.generateStats(hg, ag, home, away)
    
    return MatchWithStats{
        Result: MatchResult{
            ID: f.ID, LeagueID: f.LeagueID,
            WeekNumber: f.WeekNumber, HomeTeamID: f.HomeTeamID, AwayTeamID: f.AwayTeamID,
            HomeScore: hg, AwayScore: ag, Winner: winner,
        },
        Goals: goals, Stats: stats,
    }
}

func (p *LeagueProcessor) generateGoals(f FixtureData, hg, ag int) []GoalEvent {
    var goals []GoalEvent
    homePlayers := p.getTeamPlayers(f.HomeTeamID)
    awayPlayers := p.getTeamPlayers(f.AwayTeamID)
    
    localRng := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    for i := 0; i < hg; i++ {
        scorer := p.pickScorerWeighted(homePlayers, localRng)
        goals = append(goals, GoalEvent{
            Minute: p.realisticMinute(i, hg+ag, localRng),
            Team: "HOME", ScorerID: scorer.ID, Scorer: scorer.Name,
            Type: p.randomGoalType(localRng),
        })
    }
    for i := 0; i < ag; i++ {
        scorer := p.pickScorerWeighted(awayPlayers, localRng)
        goals = append(goals, GoalEvent{
            Minute: p.realisticMinute(i+hg, hg+ag, localRng),
            Team: "AWAY", ScorerID: scorer.ID, Scorer: scorer.Name,
            Type: p.randomGoalType(localRng),
        })
    }
    
    sort.Slice(goals, func(i, j int) bool { return goals[i].Minute < goals[j].Minute })
    return goals
}

// Realistic minute distribution
func (p *LeagueProcessor) realisticMinute(goalIndex, total int, rng *rand.Rand) int {
    roll := rng.Float64()
    var base int
    switch {
    case roll < 0.12: base = 0    // 0-15 min (12%)
    case roll < 0.28: base = 16   // 16-30 (16%)
    case roll < 0.48: base = 31   // 31-45 (20%)
    case roll < 0.53: base = 45   // Injury time (5%)
    case roll < 0.66: base = 46   // 46-60 (13%)
    case roll < 0.80: base = 61   // 61-75 (14%)
    case roll < 0.95: base = 76   // 76-90 (15%)
    default: base = 90             // 90+ (5%)
    }
    return base + rng.Intn(15)
}

func (p *LeagueProcessor) pickScorerWeighted(players []PlayerData, rng *rand.Rand) PlayerData {
    var forwards, mids, defs []PlayerData
    for _, pl := range players {
        switch pl.Position {
        case "FWD": forwards = append(forwards, pl)
        case "MID": mids = append(mids, pl)
        case "DEF": defs = append(defs, pl)
        }
    }
    
    roll := rng.Float64()
    switch {
    case roll < 0.55 && len(forwards) > 0:
        return forwards[rng.Intn(len(forwards))]
    case roll < 0.88 && len(mids) > 0:
        return mids[rng.Intn(len(mids))]
    case len(defs) > 0:
        return defs[rng.Intn(len(defs))]
    default:
        return players[rng.Intn(len(players))]
    }
}

func (p *LeagueProcessor) randomGoalType(rng *rand.Rand) string {
    types := []string{"open_play", "header", "penalty", "free_kick", "long_range", "counter_attack", "own_goal"}
    weights := []float64{0.45, 0.15, 0.10, 0.10, 0.08, 0.10, 0.02}
    roll := rng.Float64()
    cum := 0.0
    for i, w := range weights {
        cum += w
        if roll < cum { return types[i] }
    }
    return "open_play"
}

func (p *LeagueProcessor) generateStats(hg, ag int, home, away TeamWithDetails) MatchStats {
    rng := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    // Possession based on midfield strength
    homePoss := 40 + rng.Intn(20) + (home.Team.Midfield-away.Team.Midfield)/5
    
    return MatchStats{
        PossessionHome:    homePoss,
        PossessionAway:    100 - homePoss,
        ShotsOnTargetHome: hg + rng.Intn(7),
        ShotsOnTargetAway: ag + rng.Intn(6),
        CornersHome:       2 + rng.Intn(10),
        CornersAway:       1 + rng.Intn(8),
        YellowCardsHome:   rng.Intn(4),
        YellowCardsAway:   rng.Intn(4),
        RedCardsHome:      int(rng.Float64() * 0.15),
        RedCardsAway:      int(rng.Float64() * 0.12),
    }
}

// Incremental table update (called during simulation)
func (p *LeagueProcessor) updateTableInMemory(result MatchWithStats) {
    home := p.table[result.Result.HomeTeamID]
    away := p.table[result.Result.AwayTeamID]
    
    home.Played++; away.Played++
    home.GoalsFor += result.Result.HomeScore
    home.GoalsAgainst += result.Result.AwayScore
    away.GoalsFor += result.Result.AwayScore
    away.GoalsAgainst += result.Result.HomeScore
    
    switch result.Result.Winner {
    case "HOME": home.Won++; home.Points += 3; away.Lost++
    case "AWAY": away.Won++; away.Points += 3; home.Lost++
    case "DRAW": home.Drawn++; home.Points++; away.Drawn++; away.Points++
    }
}

func (p *LeagueProcessor) updateScorersInMemoryFixed(result MatchWithStats) {
    for _, goal := range result.Goals {
        if _, ok := p.scorers[goal.ScorerID]; !ok {
            // Find the player and get their team
            teamName := ""
            for _, t := range p.teams {
                for _, pl := range t.Players {
                    if pl.ID == goal.ScorerID {
                        teamName = t.Team.Name
                        break
                    }
                }
                if teamName != "" { break }
            }
            
            p.scorers[goal.ScorerID] = &ScorerRow{
                PlayerID: goal.ScorerID,
                Name:     goal.Scorer,
                TeamName: teamName,
            }
        }
        p.scorers[goal.ScorerID].Goals++
    }
}


func (p *LeagueProcessor) buildFinalTable() ([]TableRow, []ScorerRow) {
    var table []TableRow
    for _, t := range p.table {
        table = append(table, *t)
    }
    
    // Sort by points, then goal difference, then goals scored
    sort.Slice(table, func(i, j int) bool {
        if table[i].Points != table[j].Points {
            return table[i].Points > table[j].Points
        }
        gdI := table[i].GoalsFor - table[i].GoalsAgainst
        gdJ := table[j].GoalsFor - table[j].GoalsAgainst
        if gdI != gdJ {
            return gdI > gdJ
        }
        return table[i].GoalsFor > table[j].GoalsFor
    })
    
    var scorers []ScorerRow
    for _, s := range p.scorers {
        scorers = append(scorers, *s)
    }
    sort.Slice(scorers, func(i, j int) bool { return scorers[i].Goals > scorers[j].Goals })
    
    return table, scorers
}

// Helpers
func (p *LeagueProcessor) getTeam(id string) TeamWithDetails {
    for _, t := range p.teams { if t.Team.ID == id { return t } }
    return TeamWithDetails{}
}

func (p *LeagueProcessor) getTeamPlayers(teamID string) []PlayerData {
    for _, t := range p.teams {
        if t.Team.ID == teamID { return t.Players }
    }
    return nil
}

func (p *LeagueProcessor) getTeamPlayerAvg(teamID string) float64 {
    players := p.getTeamPlayers(teamID)
    if len(players) == 0 { return 50 }
    var sum float64
    for _, pl := range players { sum += pl.Rating }
    return sum / float64(len(players))
}

func (p *LeagueProcessor) getOddsForFixture(fixtureID string) FixtureWithOdds {
    for _, o := range p.odds {
        if o.Fixture.ID == fixtureID { return o }
    }
    return FixtureWithOdds{}
}

func (p *LeagueProcessor) poisson(lambda float64, rng *rand.Rand) int {
    if lambda <= 0 { 
        return 0 
    }
    L := math.Exp(-lambda)
    k := 0
    prob := 1.0
    for prob > L {
        k++
        prob *= rng.Float64()
    }
    return k - 1
}

// SimulateSingleMatch - Simulates ONE match (called by cron)
func (p *LeagueProcessor) SimulateSingleMatch(fixtureID string) MatchWithStats {
    var fixture FixtureData
    for _, f := range p.fixtures {
        if f.ID == fixtureID {
            fixture = f
            break
        }
    }
    
    if fixture.ID == "" {
        return MatchWithStats{}
    }
    
    home := p.getTeam(fixture.HomeTeamID)
    away := p.getTeam(fixture.AwayTeamID)
    
    localRng := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    // Calculate expected goals
    homeAttackStr := (float64(home.Team.Attack)*0.5 + float64(home.Coach.Attacking)*0.3) / 100
    awayDefStr := (float64(away.Team.Defense)*0.5 + float64(away.Coach.Defending)*0.3) / 100
    awayAttackStr := (float64(away.Team.Attack)*0.5 + float64(away.Coach.Attacking)*0.3) / 100
    homeDefStr := (float64(home.Team.Defense)*0.5 + float64(home.Coach.Defending)*0.3) / 100
    
    homeLambda := homeAttackStr * 3.0 * (1 - awayDefStr*0.5)
    awayLambda := awayAttackStr * 2.5 * (1 - homeDefStr*0.5)
    
    hg := p.poisson(homeLambda, localRng)
    ag := p.poisson(awayLambda, localRng)
    
    var winner string
    if hg > ag {
        winner = "HOME"
    } else if ag > hg {
        winner = "AWAY"
    } else {
        winner = "DRAW"
    }
    
    if hg > 8 { hg = 8 }
    if ag > 8 { ag = 8 }
    
    goals := p.generateGoals(fixture, hg, ag)
    stats := p.generateStats(hg, ag, home, away)
    
    return MatchWithStats{
        Result: MatchResult{
            ID: fixture.ID, LeagueID: fixture.LeagueID,
            WeekNumber: fixture.WeekNumber, HomeTeamID: fixture.HomeTeamID, AwayTeamID: fixture.AwayTeamID,
            HomeScore: hg, AwayScore: ag, Winner: winner,
        },
        Goals: goals, Stats: stats,
    }
}

// SimulateSingleMatchWithOdds - Simulates with known odds
func (p *LeagueProcessor) SimulateSingleMatchWithOdds(fixtureID string, odds FixtureWithOdds) MatchWithStats {
    var fixture FixtureData
    for _, f := range p.fixtures {
        if f.ID == fixtureID {
            fixture = f
            break
        }
    }
    
    if fixture.ID == "" {
        return MatchWithStats{}
    }
    
    home := p.getTeam(fixture.HomeTeamID)
    away := p.getTeam(fixture.AwayTeamID)
    localRng := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    // Use odds to influence result
    roll := localRng.Float64()
    var winner string
    if roll < odds.Prob.HomeWin {
        winner = "HOME"
    } else if roll < odds.Prob.HomeWin+odds.Prob.Draw {
        winner = "DRAW"
    } else {
        winner = "AWAY"
    }
    
    hg, ag := p.generateScore(winner, home, away, localRng)
    
    goals := p.generateGoals(fixture, hg, ag)
    stats := p.generateStats(hg, ag, home, away)
    
    return MatchWithStats{
        Result: MatchResult{
            ID: fixture.ID, LeagueID: fixture.LeagueID,
            WeekNumber: fixture.WeekNumber, HomeTeamID: fixture.HomeTeamID, AwayTeamID: fixture.AwayTeamID,
            HomeScore: hg, AwayScore: ag, Winner: winner,
        },
        Goals: goals, Stats: stats,
    }
}

func (p *LeagueProcessor) generateScore(winner string, home, away TeamWithDetails, rng *rand.Rand) (int, int) {
    switch winner {
    case "HOME":
        hg := 1 + rng.Intn(3)
        ag := rng.Intn(hg)
        return hg, ag
    case "AWAY":
        ag := 1 + rng.Intn(3)
        hg := rng.Intn(ag)
        return hg, ag
    default:
        g := rng.Intn(3) + 1
        return g, g
    }
}
