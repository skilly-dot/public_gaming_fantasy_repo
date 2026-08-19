package engine

import (
    "fmt"
    "math/rand"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/betking/rich-backend/internal/models"
)

type Generator struct {
    rng *rand.Rand
    mu  sync.Mutex
}

func NewGenerator() *Generator {
    return &Generator{
        rng: rand.New(rand.NewSource(time.Now().UnixNano())),
    }
}

func (g *Generator) safeIntn(n int) int {
    g.mu.Lock()
    defer g.mu.Unlock()
    return g.rng.Intn(n)
}

func (g *Generator) safeFloat64() float64 {
    g.mu.Lock()
    defer g.mu.Unlock()
    return g.rng.Float64()
}

func (g *Generator) GenerateLeague(leagueID, leagueName, leagueType string) *models.FullLeagueData {
    totalWeeks := 38
    if leagueType == "BUNDESLIGA" {
        totalWeeks = 34
    }

    // Map predefined stats to teams
    teamNames := []string{
        "London City", "Nottingham City", "Bristol United", "Manchester Reds",
        "Liverpool FC", "Arsenal Gunners", "Chelsea Blues", "Tottenham Hotspur",
        "Newcastle United", "Aston Villa", "West Ham United", "Crystal Palace",
        "Everton FC", "Leicester City", "Southampton FC", "Wolves FC",
        "Leeds United", "Brighton FC", "Brentford FC", "Fulham FC",
    }

    coaches := g.generateCoaches()
    teams := g.generateTeams(leagueID, teamNames, coaches)
    
    var wg sync.WaitGroup
    var players []models.Player
    var playerDetails []models.PlayerDetail
    var teamDetails []models.TeamDetail
    var fixtures []models.Fixture

    wg.Add(3)
    go func() {
        defer wg.Done()
        players, playerDetails = g.generateAllPlayers(teams)
    }()
    go func() {
        defer wg.Done()
        teamDetails = g.generateTeamDetails(teams, players)
    }()
    go func() {
        defer wg.Done()
        fixtures = g.generateFixtures(leagueID, teams, totalWeeks)
    }()
    wg.Wait()

    return &models.FullLeagueData{
        League: models.League{
            ID: leagueID, Name: leagueName, Type: leagueType,
            Status: "ACTIVE", TotalWeeks: totalWeeks, CurrentWeek: 0,
        },
        Teams: teams, Players: players, Fixtures: fixtures,
        Details: models.LeagueDetails{
            Coaches: coaches, PlayerStats: playerDetails, TeamDetails: teamDetails,
        },
    }
}

func (g *Generator) generateCoaches() []models.CoachDetail {
    names := []string{
        "Alex Ferguson", "Pep Guardiola", "Jose Mourinho", "Carlo Ancelotti",
        "Jurgen Klopp", "Thomas Tuchel", "Diego Simeone", "Antonio Conte",
        "Zinedine Zidane", "Massimiliano Allegri", "Luis Enrique", "Mauricio Pochettino",
        "Julian Nagelsmann", "Hansi Flick", "Erik ten Hag", "Brendan Rodgers",
        "Unai Emery", "Ralf Rangnick", "Graham Potter", "Luciano Spalletti",
    }
    formations := []string{"4-3-3", "4-4-2", "3-5-2", "4-2-3-1", "3-4-3", "4-1-4-1"}
    playstyles := []string{"GEGENPRESS", "TIKI_TAKA", "ROUTE_ONE", "PARK_THE_BUS", "WING_PLAY", "CATENACCIO", "FALSE_NINE"}
    mentalities := []string{"Attacking", "Balanced", "Defensive"}
    nations := []string{"English", "Spanish", "German", "Italian", "French", "Dutch", "Portuguese", "Argentine"}

    coaches := make([]models.CoachDetail, 20)
    for i, name := range names {
        formation := formations[g.safeIntn(len(formations))]
        playstyle := playstyles[g.safeIntn(len(playstyles))]
        coaches[i] = models.CoachDetail{
            CoachID: fmt.Sprintf("COACH_%s", uuid.New().String()[:8]),
            Name: name, Nationality: nations[g.safeIntn(len(nations))],
            Age: 40 + g.safeIntn(28), Rating: 3.5 + g.safeFloat64()*1.5,
            Formation: formation, PlayStyle: playstyle,
            SecondaryStyle: playstyles[g.safeIntn(len(playstyles))],
            Mentality: mentalities[g.safeIntn(len(mentalities))],
            Stats: models.CoachStats{
                Attacking: 50 + g.safeIntn(45), Defending: 50 + g.safeIntn(45),
                Motivation: 50 + g.safeIntn(45), Tactical: 50 + g.safeIntn(45),
                YouthDev: 30 + g.safeIntn(50), Discipline: 30 + g.safeIntn(50),
                Adaptability: 30 + g.safeIntn(50),
            },
            Abilities: g.randomAbilities(),
        }
    }
    return coaches
}

func (g *Generator) generateTeams(leagueID string, names []string, coaches []models.CoachDetail) []models.Team {
    teams := make([]models.Team, 20)
    for i, name := range names {
        rating := 3.0
        if i < 4 { rating = 4.2 + g.safeFloat64()*0.8 } else if i < 8 { rating = 3.6 + g.safeFloat64()*0.6 } else if i < 14 { rating = 3.0 + g.safeFloat64()*0.5 } else { rating = 2.3 + g.safeFloat64()*0.6 }
        teams[i] = models.Team{
            ID: fmt.Sprintf("TEAM_%s", uuid.New().String()[:8]),
            LeagueID: leagueID, Name: name, Rating: rating, CoachID: coaches[i].CoachID,
        }
    }
    return teams
}

func (g *Generator) generateAllPlayers(teams []models.Team) ([]models.Player, []models.PlayerDetail) {
    positionCounts := []struct{ pos string; count int }{{"GK", 2}, {"DEF", 6}, {"MID", 6}, {"FWD", 4}}
    firstNames := []string{"David", "Marcus", "Joseph", "Michael", "James", "Robert", "William", "Daniel", "Kevin", "Paul", "Harry", "Thomas", "Andrew", "Richard", "Steven", "Mark", "Anthony", "Brian", "Jason", "Matthew"}
    lastNames := []string{"Jones", "Garcia", "Davis", "Jackson", "Wilson", "Martinez", "Anderson", "Taylor", "Thomas", "Moore", "White", "Harris", "Martin", "Thompson", "Robinson", "Clark", "Lewis", "Walker", "Hall", "Allen"}

    var allPlayers []models.Player
    var allDetails []models.PlayerDetail

    for _, team := range teams {
        num := 1
        for _, pc := range positionCounts {
            for i := 0; i < pc.count; i++ {
                id := fmt.Sprintf("PLAYER_%s", uuid.New().String()[:8])
                name := fmt.Sprintf("%s %s", firstNames[g.safeIntn(len(firstNames))], lastNames[g.safeIntn(len(lastNames))])
                
                // Realistic rating distribution
                baseRating := 55.0 + (team.Rating-1.0)*10.0
                rating := baseRating + g.safeFloat64()*15.0
                if rating > 91 { rating = 91 }
                if rating < 55 { rating = 55 }

                allPlayers = append(allPlayers, models.Player{
                    ID: id, TeamID: team.ID, Name: name, Position: pc.pos, Number: num, Rating: rating,
                })

                stats := g.generateRealisticStats(pc.pos, int(rating))
                allDetails = append(allDetails, models.PlayerDetail{
                    PlayerID: id, Age: 18 + g.safeIntn(20), Nationality: g.randomNationality(),
                    Height: 168 + g.safeIntn(32), Weight: 62 + g.safeIntn(28),
                    StrongFoot: g.randomFoot(), Stats: stats,
                    Traits: g.randomTraits(pc.pos), Form: 5.0 + g.safeFloat64()*5.0,
                })
                num++
            }
        }
    }
    return allPlayers, allDetails
}

func (g *Generator) generateRealisticStats(pos string, rating int) models.PlayerStats {
    b := rating
    v := func(s int) int { return g.safeIntn(s*2) - s }
    
    switch pos {
    case "GK":
        return models.PlayerStats{
            Pace: c(40+v(20)), Shooting: c(15+v(15)), Passing: c(35+v(25)),
            Dribbling: c(25+v(20)), Defending: c(35+v(20)), Physical: c(60+v(25)),
            Stamina: c(50+v(20)), Aggression: c(30+v(25)), Composure: c(55+v(25)),
            Vision: c(45+v(25)), Crossing: c(15+v(15)), Finishing: c(10+v(10)),
            LongShots: c(10+v(15)), Penalties: c(20+v(25)), FreeKicks: c(15+v(20)),
            Heading: c(40+v(20)), Tackling: c(20+v(20)), Positioning: c(65+v(25)),
            Reflexes: c(b+v(15)), Diving: c(b+v(15)), Handling: c(b+v(15)),
        }
    case "DEF":
        return models.PlayerStats{
            Pace: c(55+v(25)), Shooting: c(35+v(25)), Passing: c(55+v(25)),
            Dribbling: c(45+v(25)), Defending: c(b+v(15)), Physical: c(65+v(25)),
            Stamina: c(60+v(20)), Aggression: c(55+v(30)), Composure: c(55+v(25)),
            Vision: c(45+v(25)), Crossing: c(50+v(30)), Finishing: c(25+v(20)),
            LongShots: c(30+v(25)), Penalties: c(30+v(30)), FreeKicks: c(25+v(25)),
            Heading: c(65+v(25)), Tackling: c(b+v(15)), Positioning: c(65+v(25)),
            Reflexes: c(25+v(25)), Diving: c(5+v(10)), Handling: c(5+v(10)),
        }
    case "MID":
        return models.PlayerStats{
            Pace: c(60+v(25)), Shooting: c(55+v(25)), Passing: c(b+v(15)),
            Dribbling: c(60+v(25)), Defending: c(45+v(25)), Physical: c(55+v(25)),
            Stamina: c(65+v(20)), Aggression: c(45+v(30)), Composure: c(60+v(25)),
            Vision: c(b+v(15)), Crossing: c(55+v(25)), Finishing: c(50+v(25)),
            LongShots: c(50+v(30)), Penalties: c(45+v(35)), FreeKicks: c(45+v(35)),
            Heading: c(45+v(25)), Tackling: c(45+v(25)), Positioning: c(55+v(25)),
            Reflexes: c(20+v(20)), Diving: c(5+v(8)), Handling: c(5+v(8)),
        }
    case "FWD":
        return models.PlayerStats{
            Pace: c(b+v(15)), Shooting: c(b+v(15)), Passing: c(50+v(25)),
            Dribbling: c(65+v(25)), Defending: c(20+v(20)), Physical: c(55+v(25)),
            Stamina: c(55+v(25)), Aggression: c(45+v(30)), Composure: c(60+v(25)),
            Vision: c(50+v(25)), Crossing: c(45+v(30)), Finishing: c(b+v(15)),
            LongShots: c(50+v(30)), Penalties: c(55+v(35)), FreeKicks: c(45+v(35)),
            Heading: c(55+v(30)), Tackling: c(15+v(15)), Positioning: c(60+v(25)),
            Reflexes: c(20+v(25)), Diving: c(5+v(8)), Handling: c(5+v(8)),
        }
    }
    return models.PlayerStats{}
}

func (g *Generator) generateTeamDetails(teams []models.Team, allPlayers []models.Player) []models.TeamDetail {
    formations := []string{"4-3-3", "4-4-2", "3-5-2", "4-2-3-1", "3-4-3", "4-1-4-1"}
    details := make([]models.TeamDetail, len(teams))
    for i, team := range teams {
        var teamPlayers []models.Player
        for _, p := range allPlayers { if p.TeamID == team.ID { teamPlayers = append(teamPlayers, p) } }
        var captain, vc, pk, fk, ck string
        if len(teamPlayers) > 0 { captain = teamPlayers[0].ID }
        if len(teamPlayers) > 1 { vc = teamPlayers[1].ID }
        if len(teamPlayers) > 2 { pk = teamPlayers[2].ID }
        for _, p := range teamPlayers { if p.Position == "MID" && fk == "" { fk = p.ID }; if p.Position == "MID" && ck == "" { ck = p.ID } }
        
        // Map predefined stats based on team position
        att, mid, def := 50+g.safeIntn(40), 50+g.safeIntn(40), 50+g.safeIntn(40)
        if i < 4 { att += 15; mid += 10; def += 8 } else if i < 8 { att += 5; mid += 8 } else if i >= 14 { att -= 10; def -= 5 }
        
        details[i] = models.TeamDetail{
            TeamID: team.ID, Formation: formations[g.safeIntn(len(formations))],
            CaptainID: captain, ViceCaptain: vc, PenaltyTaker: pk, FreeKickTaker: fk, CornerTaker: ck,
            Stats: models.TeamStats{
                Attack: c(att), Midfield: c(mid), Defense: c(def),
                Teamwork: c(50+g.safeIntn(40)), Experience: c(40+g.safeIntn(50)), SetPieces: c(40+g.safeIntn(40)),
            },
        }
    }
    return details
}

func (g *Generator) generateFixtures(leagueID string, teams []models.Team, totalWeeks int) []models.Fixture {
    n := len(teams)
    fixtures := make([]models.Fixture, 0, totalWeeks*10)
    for week := 1; week <= totalWeeks; week++ {
        rotated := make([]models.Team, n)
        copy(rotated, teams)
        shift := (week - 1) % (n - 1)
        for i := 1; i < n; i++ {
            newPos := 1 + (i-1+shift)%(n-1)
            rotated[newPos] = teams[i]
        }
        for i := 0; i < n; i += 2 {
            home, away := rotated[i], rotated[i+1]
            if week > totalWeeks/2 { home, away = away, home }
            fixtures = append(fixtures, models.Fixture{
                LeagueID: leagueID, WeekNumber: week, HomeTeamID: home.ID, AwayTeamID: away.ID,
            })
        }
    }
    return fixtures
}

func c(v int) int { if v < 1 { return 1 }; if v > 99 { return 99 }; return v }
func (g *Generator) randomFoot() string { if g.safeFloat64() < 0.7 { return "Right" }; return "Left" }
func (g *Generator) randomNationality() string {
    n := []string{"English","Spanish","German","French","Italian","Brazilian","Argentine","Dutch","Portuguese","Belgian","Croatian","Serbian","Polish","Swedish","Norwegian"}
    return n[g.safeIntn(len(n))]
}
func (g *Generator) randomTraits(pos string) []string {
    t := map[string][]string{
        "GK": {"Diving Specialist","Penalty Stopper","Sweeper Keeper","Leader","Long Throw"},
        "DEF": {"Tackling Master","Aerial Threat","Ball Playing Defender","Speedster","Last Man"},
        "MID": {"Playmaker","Box-to-Box","Deep Lying","Free Kick Specialist","Long Shot Taker"},
        "FWD": {"Clinical Finisher","Speed Demon","Target Man","Poacher","False 9"},
    }
    pool := t[pos]; count := 2+g.safeIntn(3); seen := map[string]bool{}; var res []string
    for i := 0; i < count && i < len(pool); i++ { tr := pool[g.safeIntn(len(pool))]; if !seen[tr] { res = append(res, tr); seen[tr] = true } }
    return res
}
func (g *Generator) randomAbilities() []string {
    a := []string{"Motivator","Tactician","Youth Developer","Defensive Master","Attacking Genius","Man Manager","Set Piece Specialist","Counter-Attack Expert"}
    count := 2+g.safeIntn(3); seen := map[string]bool{}; var res []string
    for i := 0; i < count; i++ { ab := a[g.safeIntn(len(a))]; if !seen[ab] { res = append(res, ab); seen[ab] = true } }
    return res
}
