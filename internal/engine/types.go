package engine

// ============================================
// ALL TYPES
// ============================================

type DifficultyConfig struct {
    Name              string
    RatingSpread      float64
    HomeAdvantage     float64
    UpsetRate         float64
    ScoringMultiplier float64
    DrawRate          float64
}

type LeagueTypeConfig struct {
    Name           string
    TotalWeeks     int
    TeamsCount     int
    PlayersPerTeam int
    Style          string
}

type LeagueInfo struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    Type       string `json:"type"`
    Difficulty string `json:"difficulty"`
    TotalWeeks int    `json:"total_weeks"`
}

type TeamData struct {
    ID         string  `json:"id"`
    Name       string  `json:"name"`
    Rating     float64 `json:"rating"`
    Formation  string  `json:"formation"`
    Attack     int     `json:"attack"`
    Midfield   int     `json:"midfield"`
    Defense    int     `json:"defense"`
    Teamwork   int     `json:"teamwork"`
    Experience int     `json:"experience"`
    SetPieces  int     `json:"set_pieces"`
}

type CoachData struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Nationality string   `json:"nationality"`
    Formation   string   `json:"formation"`
    PlayStyle   string   `json:"playstyle"`
    Mentality   string   `json:"mentality"`
    Attacking   int      `json:"attacking"`
    Defending   int      `json:"defending"`
    Tactical    int      `json:"tactical"`
    Motivation  int      `json:"motivation"`
    Abilities   []string `json:"abilities"`
}

type PlayerData struct {
    ID        string   `json:"id"`
    TeamID    string   `json:"team_id"`
    LeagueID  string   `json:"league_id"`
    Name      string   `json:"name"`
    Position  string   `json:"position"`
    Number    int      `json:"number"`
    Rating    float64  `json:"rating"`
    Pace      int      `json:"pace"`
    Shooting  int      `json:"shooting"`
    Passing   int      `json:"passing"`
    Dribbling int      `json:"dribbling"`
    Defending int      `json:"defending"`
    Physical  int      `json:"physical"`
    Stamina   int      `json:"stamina"`
    Form      float64  `json:"form"`
    Traits    []string `json:"traits"`
}

type FixtureData struct {
    ID          string `json:"id"`
    LeagueID    string `json:"league_id"`
    WeekNumber  int    `json:"week_number"`
    HomeTeamID  string `json:"home_team_id"`
    AwayTeamID  string `json:"away_team_id"`
    KickoffTime int64  `json:"kickoff_time"`
}

type MatchResult struct {
    ID          string `json:"id"`
    LeagueID    string `json:"league_id"`
    WeekNumber  int    `json:"week_number"`
    HomeTeamID  string `json:"home_team_id"`
    AwayTeamID  string `json:"away_team_id"`
    HomeScore   int    `json:"home_score"`
    AwayScore   int    `json:"away_score"`
    Winner      string `json:"winner"`
}

type GoalEvent struct {
    Minute   int    `json:"minute"`
    Team     string `json:"team"`
    ScorerID string `json:"scorer_id"`
    Scorer   string `json:"scorer"`
    Type     string `json:"type"`
}

type MatchStats struct {
    PossessionHome    int `json:"possession_home"`
    PossessionAway    int `json:"possession_away"`
    ShotsOnTargetHome int `json:"shots_on_target_home"`
    ShotsOnTargetAway int `json:"shots_on_target_away"`
    CornersHome       int `json:"corners_home"`
    CornersAway       int `json:"corners_away"`
    YellowCardsHome   int `json:"yellow_cards_home"`
    YellowCardsAway   int `json:"yellow_cards_away"`
    RedCardsHome      int `json:"red_cards_home"`
    RedCardsAway      int `json:"red_cards_away"`
}

type ProbData struct {
    HomeWin float64 `json:"home_win"`
    Draw    float64 `json:"draw"`
    AwayWin float64 `json:"away_win"`
}

type OddsData struct {
    Home float64 `json:"home"`
    Draw float64 `json:"draw"`
    Away float64 `json:"away"`
}

type TableRow struct {
    TeamID       string `json:"team_id"`
    TeamName     string `json:"team_name"`
    Played       int    `json:"played"`
    Won          int    `json:"won"`
    Drawn        int    `json:"drawn"`
    Lost         int    `json:"lost"`
    GoalsFor     int    `json:"goals_for"`
    GoalsAgainst int    `json:"goals_against"`
    Points       int    `json:"points"`
}

type ScorerRow struct {
    PlayerID string `json:"player_id"`
    Name     string `json:"name"`
    TeamName string `json:"team_name"`
    Goals    int    `json:"goals"`
}



type CoachPreset struct {
    Name          string
    Nationality   string
    Formation     string
    PlayStyle     string
    Mentality     string
    Attacking     int
    Defending     int
    Tactical      int
    Motivation    int
    YouthDev      int
    Discipline    int
    Adaptability  int
    Abilities     []string
}

type TeamStatSet struct {
    Attack     int
    Midfield   int
    Defense    int
    Teamwork   int
    Experience int
    SetPieces  int
}

type TeamPackage struct {
    Team    TeamData
    Coach   CoachData
    Players []PlayerData
}
