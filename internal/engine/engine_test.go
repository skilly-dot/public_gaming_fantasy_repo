package engine

import (
	"fmt"
	"testing"
)

func TestGenerateFullLeagueCustom(t *testing.T) {
    g := NewUltimateGenerator()
    
    tests := []struct {
        name       string
        teamCount  int
        difficulty string
        wantTeams  int
        wantWeeks  int
    }{
        {"Mini League", 5, "BEGINNER", 5, 8},
        {"Minor League", 10, "NORMAL", 10, 18},
        {"Major League", 15, "HARD", 15, 28},
        {"Premier League", 20, "GRANDMASTER", 20, 38},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            league := g.GenerateFullLeagueCustom("Test", "CUSTOM", tt.difficulty, tt.teamCount)
            
            if league == nil {
                t.Fatal("GenerateFullLeagueCustom returned nil")
            }
            
            if len(league.Teams) != tt.wantTeams {
                t.Errorf("Expected %d teams, got %d", tt.wantTeams, len(league.Teams))
            }
            
            if league.League.TotalWeeks != tt.wantWeeks {
                t.Errorf("Expected %d weeks, got %d", tt.wantWeeks, league.League.TotalWeeks)
            }
            
            if len(league.Fixtures) == 0 {
                t.Error("No fixtures generated")
            }
            
            if len(league.Table) != tt.wantTeams {
                t.Errorf("Expected %d table rows, got %d", tt.wantTeams, len(league.Table))
            }
            
            // Verify each team has players
            for _, team := range league.Teams {
                if len(team.Players) == 0 {
                    t.Errorf("Team %s has no players", team.Team.Name)
                }
                
                if team.Team.ID == "" {
                    t.Error("Team has empty ID")
                }
                
                if team.Coach.Name == "" {
                    t.Errorf("Team %s has no coach", team.Team.Name)
                }
            }
            
            // Verify fixtures have valid references
            teamIDs := make(map[string]bool)
            for _, team := range league.Teams {
                teamIDs[team.Team.ID] = true
            }
            
            for _, fixture := range league.Fixtures {
                if !teamIDs[fixture.Fixture.HomeTeamID] {
                    t.Errorf("Fixture references unknown home team: %s", fixture.Fixture.HomeTeamID)
                }
                if !teamIDs[fixture.Fixture.AwayTeamID] {
                    t.Errorf("Fixture references unknown away team: %s", fixture.Fixture.AwayTeamID)
                }
            }
        })
    }
}

func TestGenerateFullLeague(t *testing.T) {
    g := NewUltimateGenerator()
    
    for leagueType, config := range LeagueTypes {
        t.Run(leagueType, func(t *testing.T) {
            var league *FullLeague
            
            if leagueType == "CUSTOM" {
                // Test CUSTOM with 10 teams
                league = g.GenerateFullLeagueCustom("Test Custom", "CUSTOM", "NORMAL", 10)
                
                if len(league.Teams) != 10 {
                    t.Errorf("Expected 10 teams, got %d", len(league.Teams))
                }
                
                if league.League.TotalWeeks != 18 {
                    t.Errorf("Expected 18 weeks, got %d", league.League.TotalWeeks)
                }
            } else {
                league = g.GenerateFullLeague("Test "+leagueType, leagueType, "NORMAL")
                
                if len(league.Teams) != config.TeamsCount {
                    t.Errorf("Expected %d teams, got %d", config.TeamsCount, len(league.Teams))
                }
                
                if league.League.TotalWeeks != config.TotalWeeks {
                    t.Errorf("Expected %d weeks, got %d", config.TotalWeeks, league.League.TotalWeeks)
                }
            }
            
            if league == nil {
                t.Fatal("GenerateFullLeague returned nil")
            }
            
            if len(league.Fixtures) == 0 {
                t.Error("No fixtures generated")
            }
        })
    }
}

func TestPickRandomFromPool(t *testing.T) {
    g := NewUltimateGenerator()
    
    pool := []string{"A", "B", "C", "D", "E"}
    
    // Test with count less than pool
    result := g.pickRandomFromPool(pool, 3)
    if len(result) != 3 {
        t.Errorf("Expected 3 items, got %d", len(result))
    }
    
    // Check uniqueness
    seen := make(map[string]bool)
    for _, item := range result {
        if seen[item] {
            t.Errorf("Duplicate item: %s", item)
        }
        seen[item] = true
    }
    
    // Test with count greater than pool
    result2 := g.pickRandomFromPool(pool, 10)
    if len(result2) != len(pool) {
        t.Errorf("Expected %d items (capped), got %d", len(pool), len(result2))
    }
}

func TestPoissonDistribution(t *testing.T) {
    g := NewUltimateGenerator()
    
    // Test with different lambda values
    lambdas := []float64{0.5, 1.0, 1.5, 2.0, 3.0}
    
    for _, lambda := range lambdas {
        // Run 1000 times
        sum := 0
        min := 999
        max := -1
        
        for i := 0; i < 1000; i++ {
            result := g.poisson(lambda)
            sum += result
            if result < min { min = result }
            if result > max { max = result }
        }
        
        avg := float64(sum) / 1000.0
        
        if avg < lambda*0.5 || avg > lambda*1.5 {
            t.Errorf("Poisson(%v) average = %.2f, expected around %.2f", lambda, avg, lambda)
        }
        
        if min < 0 {
            t.Errorf("Poisson(%v) produced negative value", lambda)
        }
        
        if max > 10 {
            t.Errorf("Poisson(%v) produced too high value: %d", lambda, max)
        }
    }
}

func TestCalculateAllOdds(t *testing.T) {
    g := NewUltimateGenerator()
    
    // Create test teams
    teams := []TeamWithDetails{
        {
            Team: TeamData{ID: "TEAM_A", Name: "Team A", Rating: 5.0, Attack: 90, Midfield: 85, Defense: 80},
            Coach: CoachData{Name: "Coach A", Attacking: 90, Defending: 80},
            Players: g.generatePlayers("TEAM_A", "LEAGUE_TEST", 5.0, 18),
        },
        {
            Team: TeamData{ID: "TEAM_B", Name: "Team B", Rating: 3.0, Attack: 70, Midfield: 65, Defense: 60},
            Coach: CoachData{Name: "Coach B", Attacking: 70, Defending: 60},
            Players: g.generatePlayers("TEAM_B", "LEAGUE_TEST", 3.0, 18),
        },
    }
    
    fixtures := []FixtureData{
        {ID: "FIX_1", LeagueID: "LEAGUE_TEST", WeekNumber: 1, HomeTeamID: "TEAM_A", AwayTeamID: "TEAM_B"},
    }
    
    config := DifficultyLevels["NORMAL"]
    
    results := g.calculateAllOdds(fixtures, teams, config)
    
    if len(results) != 1 {
        t.Fatalf("Expected 1 result, got %d", len(results))
    }
    
    odds := results[0].Odds
    if odds.Home <= 0 || odds.Draw <= 0 || odds.Away <= 0 {
        t.Errorf("Invalid odds: %v", odds)
    }
    
    // Strong team should have lower odds (higher probability)
    if odds.Home > odds.Away {
        t.Errorf("Home team stronger but has higher odds: home=%v away=%v", odds.Home, odds.Away)
    }
}

func TestSimulateAllMatches(t *testing.T) {
    g := NewUltimateGenerator()
    
    // Create 4 test teams
    teams := []TeamWithDetails{}
    for i := 0; i < 4; i++ {
        rating := 5.0 - float64(i)
        teams = append(teams, TeamWithDetails{
            Team: TeamData{ID: fmt.Sprintf("TEAM_%d", i), Name: fmt.Sprintf("Team %d", i), Rating: rating, Attack: 80, Midfield: 75, Defense: 70},
            Coach: CoachData{Name: fmt.Sprintf("Coach %d", i), Attacking: 75, Defending: 70},
            Players: g.generatePlayers(fmt.Sprintf("TEAM_%d", i), "LEAGUE_TEST", rating, 18),
        })
    }
    
    fixtures := g.generateFixturesCustom("LEAGUE_TEST", teams, 6)
    
    config := DifficultyLevels["NORMAL"]
    
    results, table, scorers := g.simulateAllMatches(fixtures, teams, config)
    
    if len(results) != len(fixtures) {
        t.Errorf("Expected %d results, got %d", len(fixtures), len(results))
    }
    
    if len(table) != len(teams) {
        t.Errorf("Expected %d table rows, got %d", len(teams), len(table))
    }
    
    // Verify table points add up
    totalPoints := 0
    for _, row := range table {
        totalPoints += row.Points
    }
    
    if totalPoints == 0 {
        t.Error("No points in table")
    }
    
    // Verify scorers
    if len(scorers) == 0 {
        t.Error("No scorers generated")
    }
}

func TestFindTeam(t *testing.T) {
    g := NewUltimateGenerator()
    
    teams := []TeamWithDetails{
        {Team: TeamData{ID: "TEAM_1", Name: "Team 1"}},
        {Team: TeamData{ID: "TEAM_2", Name: "Team 2"}},
    }
    
    found := g.findTeam(teams, "TEAM_1")
    if found.Team.Name != "Team 1" {
        t.Errorf("Expected Team 1, got %s", found.Team.Name)
    }
    
    notFound := g.findTeam(teams, "TEAM_999")
    if notFound.Team.ID != "" {
        t.Errorf("Expected empty team, got %s", notFound.Team.ID)
    }
}

func TestGeneratePlayers(t *testing.T) {
    g := NewUltimateGenerator()
    
    players := g.generatePlayers("TEAM_TEST", "LEAGUE_TEST", 4.0, 18)
    
    if len(players) != 18 {
        t.Errorf("Expected 18 players, got %d", len(players))
    }
    
    positions := make(map[string]int)
    for _, p := range players {
        positions[p.Position]++
        
        if p.Name == "" {
            t.Error("Player has empty name")
        }
        
        if p.Rating < 50 || p.Rating > 95 {
            t.Errorf("Player %s rating %v out of range", p.Name, p.Rating)
        }
    }
    
    // Check position distribution
    if positions["GK"] < 1 {
        t.Error("No goalkeepers")
    }
    if positions["DEF"] < 3 {
        t.Error("Not enough defenders")
    }
    if positions["MID"] < 3 {
        t.Error("Not enough midfielders")
    }
    if positions["FWD"] < 1 {
        t.Error("No forwards")
    }
}