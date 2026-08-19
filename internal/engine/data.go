package engine

// ============================================
// 200 REALISTIC TEAM NAMES (Variations of real teams)
// ============================================
var TeamNames = []string{
    // ENGLISH STYLE (50)
    "Manchester Reds FC", "Manchester Blues FC", "London Gunners", "London Blues FC",
    "Liverpool Reds FC", "Tottenham Whites", "Newcastle Magpies", "Aston Lions",
    "West Ham Irons", "Crystal Palace Eagles", "Everton Toffees", "Leicester Foxes",
    "Southampton Saints", "Wolverhampton Wolves", "Leeds Whites FC", "Brighton Seagulls",
    "Brentford Bees FC", "Fulham Whites", "Bournemouth Cherries", "Nottingham Foresters",
    "Burnley Clarets", "Sheffield Blades", "Luton Hatters", "Coventry Sky Blues",
    "Millwall Lions FC", "Sunderland Cats", "Blackburn Rivers", "Middlesbrough Reds",
    "Stoke Potters FC", "Hull Tigers FC", "Cardiff Bluebirds", "Swansea Swans FC",
    "Norwich Canaries", "Watford Hornets", "Reading Royals", "Derby Rams FC",
    "Preston North End", "Bristol Robins", "QPR Rangers", "Birmingham Blues",
    "Wigan Warriors", "Bolton Wanderers", "Portsmouth Blues", "Plymouth Pilgrims",
    "Ipswich Blues", "Charlton Addicks", "Barnsley Tykes", "Rotherham Millers",
    "Peterborough Posh", "Wycombe Chairboys",

    // SPANISH STYLE (35)
    "Madrid Royals FC", "Barcelona Blues", "Atletico Bears", "Sevilla Oranges",
    "Valencia Bats FC", "Villarreal Submarine", "Real Sociedad Whites", "Real Betis Greens",
    "Celta Vigo Sky", "Osasuna Reds FC", "Mallorca Islands", "Rayo Vallecano Rays",
    "Girona Catalans", "Alaves Blues", "Getafe Blues FC", "Espanyol Parrots",
    "Granada Reds FC", "Cadiz Yellows", "Elche Greens FC", "Almeria Reds",
    "Levante Frogs", "Huesca Blues", "Tenerife Blues", "Las Palmas Yellows",
    "Zaragoza Whites", "Oviedo Blues", "Sporting Gijon Reds", "Racing Santander Greens",
    "Deportivo Blues", "Malaga Anchovies", "Cordoba Greens", "Albacete Whites",
    "Ponferradina Blues", "Lugo Reds", "Mirandes Reds",

    // ITALIAN STYLE (35)
    "Turin Stripes FC", "Milan Devils FC", "Milan Blues FC", "Naples Blues",
    "Rome Wolves FC", "Rome Blues FC", "Florence Lilies", "Bergamo Goddesses",
    "Bologna Reds FC", "Turin Bulls FC", "Udine Zebras", "Sassuolo Greens",
    "Genoa Griffins", "Cagliari Reds", "Verona Blues", "Salerno Reds",
    "Lecce Wolves", "Monza Reds FC", "Empoli Blues", "Frosinone Yellows",
    "Parma Yellows", "Brescia Blues", "Cremona Reds", "Perugia Reds",
    "Bari Reds FC", "Palermo Pinks", "Catania Reds", "Venezia Blacks",
    "Como Blues", "Modena Yellows", "Reggiana Reds", "Cesena Whites",
    "Pisa Blues", "Ascoli Whites", "Cittadella Reds",

    // GERMAN STYLE (30)
    "Munich Power FC", "Dortmund Yellows", "Leipzig Bulls FC", "Leverkusen Lions",
    "Frankfurt Eagles", "Wolfsburg Wolves", "Mönchengladbach Foals", "Freiburg Blacks",
    "Hoffenheim Blues", "Mainz Reds FC", "Augsburg Reds", "Bremen Greens",
    "Stuttgart Reds", "Cologne Goats", "Hertha Blues FC", "Schalke Blues",
    "Hamburg Reds FC", "Düsseldorf Reds", "Nuremberg Reds", "Hannover Reds",
    "Kaiserslautern Reds", "Bochum Blues", "Bielefeld Blues", "Darmstadt Lilies",
    "Heidenheim Reds", "Kiel Blues", "St. Pauli Browns", "Paderborn Blues",
    "Karlsruhe Blues", "Braunschweig Yellows",

    // FRENCH STYLE (25)
    "Paris Stars FC", "Marseille Blues", "Lyon Lions FC", "Monaco Royals",
    "Lille Mastiffs", "Rennes Reds FC", "Nice Eagles FC", "Strasbourg Blues",
    "Lens Bloods FC", "Nantes Canaries", "Reims Reds FC", "Montpellier Suns",
    "Toulouse Purples", "Brest Pirates", "Lorient Oranges", "Clermont Reds",
    "Auxerre Blues", "Ajaccio Bears", "Troyes Blues", "Angers Blacks",
    "Saint-Étienne Greens", "Bordeaux Reds", "Metz Reds", "Caen Reds",
    "Guingamp Reds",

    // EUROPEAN OTHER (25)
    "Amsterdam Masters", "Rotterdam Pride", "Eindhoven Reds", "Lisbon Eagles",
    "Porto Dragons FC", "Lisbon Lions FC", "Braga Warriors", "Glasgow Hoops",
    "Glasgow Bears FC", "Brussels Reds", "Bruges Blues", "Geneva Blues",
    "Basel Reds FC", "Zurich Lions FC", "Vienna Reds", "Salzburg Bulls",
    "Prague Reds", "Prague Blues", "Warsaw Eagles", "Copenhagen Lions",
    "Stockholm Blues", "Oslo Blues", "Helsinki Blues", "Athens Reds",
    "Istanbul Lions",
}

// ============================================
// 20 REALISTIC COACHES
// ============================================
var CoachPool = []CoachPreset{
    {"Alex Ferguson", "Scottish", "4-4-2", "ROUTE_ONE", "Attacking", 88, 72, 92, 95, 85, 90, 88, []string{"Motivator", "Tactician", "Man Manager"}},
    {"Pep Guardiola", "Spanish", "4-3-3", "TIKI_TAKA", "Attacking", 95, 75, 95, 85, 80, 70, 90, []string{"Attacking Genius", "Tactician", "Youth Developer"}},
    {"Jose Mourinho", "Portuguese", "4-2-3-1", "PARK_THE_BUS", "Defensive", 72, 95, 90, 88, 60, 85, 75, []string{"Defensive Master", "Motivator", "Tactician"}},
    {"Carlo Ancelotti", "Italian", "4-3-3", "BALANCED", "Balanced", 85, 82, 88, 90, 75, 70, 92, []string{"Man Manager", "Tactician", "Motivator"}},
    {"Jurgen Klopp", "German", "4-3-3", "GEGENPRESS", "Attacking", 92, 78, 85, 95, 85, 75, 80, []string{"Motivator", "Attacking Genius", "Youth Developer"}},
    {"Thomas Tuchel", "German", "3-4-3", "TIKI_TAKA", "Balanced", 82, 85, 88, 80, 78, 82, 85, []string{"Tactician", "Defensive Master"}},
    {"Diego Simeone", "Argentine", "4-4-2", "CATENACCIO", "Defensive", 75, 92, 85, 90, 70, 88, 78, []string{"Defensive Master", "Motivator"}},
    {"Antonio Conte", "Italian", "3-5-2", "WING_PLAY", "Balanced", 80, 88, 85, 88, 65, 90, 72, []string{"Tactician", "Disciplinarian"}},
    {"Zinedine Zidane", "French", "4-3-3", "BALANCED", "Attacking", 85, 78, 82, 88, 80, 75, 85, []string{"Man Manager", "Motivator"}},
    {"Luis Enrique", "Spanish", "4-3-3", "TIKI_TAKA", "Attacking", 88, 72, 85, 82, 85, 72, 80, []string{"Attacking Genius", "Youth Developer"}},
    {"Mauricio Pochettino", "Argentine", "4-2-3-1", "GEGENPRESS", "Attacking", 82, 75, 80, 85, 88, 78, 82, []string{"Youth Developer", "Motivator"}},
    {"Julian Nagelsmann", "German", "3-4-3", "GEGENPRESS", "Attacking", 85, 72, 88, 78, 90, 70, 85, []string{"Tactician", "Youth Developer"}},
    {"Hansi Flick", "German", "4-2-3-1", "GEGENPRESS", "Attacking", 88, 75, 82, 85, 75, 78, 80, []string{"Attacking Genius", "Motivator"}},
    {"Erik ten Hag", "Dutch", "4-3-3", "TIKI_TAKA", "Attacking", 82, 78, 85, 80, 85, 82, 78, []string{"Tactician", "Youth Developer"}},
    {"Mikel Arteta", "Spanish", "4-3-3", "TIKI_TAKA", "Balanced", 82, 78, 82, 82, 82, 78, 82, []string{"Tactician", "Youth Developer"}},
    {"Roberto De Zerbi", "Italian", "4-2-3-1", "TIKI_TAKA", "Attacking", 85, 68, 88, 78, 82, 72, 80, []string{"Attacking Genius", "Tactician"}},
    {"Eddie Howe", "English", "4-3-3", "BALANCED", "Attacking", 80, 72, 78, 85, 82, 75, 80, []string{"Motivator", "Youth Developer"}},
    {"Ruben Amorim", "Portuguese", "3-4-3", "BALANCED", "Balanced", 78, 75, 82, 85, 88, 78, 85, []string{"Youth Developer", "Tactician"}},
    {"Luciano Spalletti", "Italian", "4-3-3", "BALANCED", "Attacking", 82, 75, 80, 82, 72, 78, 80, []string{"Tactician", "Motivator"}},
    {"Ralf Rangnick", "German", "4-4-2", "GEGENPRESS", "Attacking", 78, 72, 85, 75, 82, 75, 78, []string{"Tactician", "Youth Developer"}},
}

// ============================================
// 100 TEAM STRENGTH SETS (TIERED)
// ============================================
var TeamStatPool = []TeamStatSet{
    // TIER 1: ELITE (5 sets - ratings 88-92)
    {92, 90, 85, 88, 85, 82},
    {90, 88, 82, 90, 88, 80},
    {88, 92, 80, 85, 92, 85},
    {90, 85, 88, 86, 88, 82},
    {85, 88, 90, 88, 90, 78},

    // TIER 2: STRONG (15 sets - ratings 80-87)
    {85, 84, 80, 86, 82, 78},
    {82, 85, 78, 80, 78, 76},
    {88, 82, 80, 90, 82, 80},
    {82, 80, 82, 84, 80, 82},
    {80, 82, 84, 82, 80, 80},
    {84, 78, 80, 78, 80, 80},
    {80, 84, 76, 82, 82, 78},
    {88, 85, 82, 82, 92, 88},
    {85, 82, 78, 80, 80, 82},
    {82, 80, 78, 84, 78, 78},
    {78, 80, 88, 88, 84, 80},
    {82, 82, 78, 82, 80, 76},
    {80, 78, 80, 80, 78, 80},
    {78, 82, 82, 80, 76, 84},
    {82, 80, 78, 82, 78, 78},

    // TIER 3: MID (30 sets - ratings 70-79)
    {78, 80, 76, 82, 76, 78},
    {76, 78, 80, 78, 74, 74},
    {80, 76, 74, 80, 72, 76},
    {74, 78, 78, 78, 76, 78},
    {78, 74, 76, 76, 78, 74},
    {76, 76, 78, 74, 72, 76},
    {74, 80, 72, 78, 70, 74},
    {78, 72, 76, 76, 74, 72},
    {72, 76, 78, 74, 76, 78},
    {76, 78, 72, 72, 78, 70},
    {70, 76, 76, 78, 72, 78},
    {76, 72, 74, 76, 70, 76},
    {72, 74, 78, 72, 74, 74},
    {74, 76, 70, 74, 76, 72},
    {70, 78, 72, 76, 68, 70},
    {78, 76, 74, 80, 78, 76},
    {76, 74, 76, 82, 80, 78},
    {74, 76, 78, 80, 82, 74},
    {78, 80, 72, 76, 78, 76},
    {76, 78, 76, 78, 76, 74},
    {78, 74, 74, 80, 78, 78},
    {76, 80, 72, 78, 74, 76},
    {78, 76, 74, 76, 72, 74},
    {74, 78, 76, 74, 76, 72},
    {76, 74, 78, 76, 74, 76},
    {78, 76, 72, 78, 72, 74},
    {74, 78, 74, 76, 74, 72},
    {76, 76, 76, 74, 76, 74},
    {78, 74, 74, 76, 78, 76},
    {76, 74, 76, 74, 70, 74},

    // TIER 4: WEAK (50 sets - ratings 50-69)
    {72, 70, 68, 78, 68, 72},
    {70, 68, 72, 76, 72, 70},
    {68, 72, 70, 74, 66, 68},
    {72, 68, 68, 72, 64, 70},
    {68, 70, 72, 70, 68, 68},
    {70, 66, 70, 72, 70, 66},
    {68, 68, 68, 70, 62, 70},
    {66, 70, 68, 68, 68, 68},
    {68, 66, 70, 68, 64, 66},
    {64, 68, 66, 70, 60, 64},
    {66, 64, 68, 68, 62, 68},
    {62, 68, 64, 66, 64, 62},
    {68, 62, 66, 64, 60, 66},
    {64, 66, 62, 68, 58, 64},
    {60, 64, 68, 62, 62, 60},
    {62, 60, 58, 68, 58, 60},
    {60, 58, 62, 66, 60, 58},
    {58, 62, 60, 64, 56, 60},
    {60, 58, 58, 62, 54, 58},
    {58, 60, 56, 60, 58, 56},
    {56, 58, 60, 58, 56, 56},
    {62, 60, 56, 60, 54, 58},
    {54, 58, 56, 56, 52, 54},
    {56, 54, 54, 54, 50, 52},
    {60, 58, 56, 62, 58, 58},
    {58, 56, 58, 60, 56, 56},
    {62, 58, 56, 64, 60, 60},
    {60, 60, 58, 62, 62, 58},
    {58, 62, 54, 60, 58, 56},
    {56, 58, 58, 58, 60, 54},
    {60, 56, 60, 56, 58, 58},
    {58, 60, 58, 58, 62, 56},
    {62, 58, 54, 62, 56, 58},
    {58, 56, 58, 60, 58, 56},
    {56, 58, 56, 58, 60, 54},
    {54, 56, 58, 56, 56, 52},
    {58, 54, 56, 60, 58, 56},
    {56, 56, 54, 58, 56, 54},
    {54, 52, 54, 56, 54, 52},
    {56, 54, 52, 58, 50, 54},
    {54, 52, 54, 56, 52, 52},
    {56, 54, 52, 58, 54, 54},
    {54, 56, 52, 56, 56, 52},
    {52, 54, 54, 54, 54, 50},
    {54, 52, 52, 56, 50, 52},
    {52, 52, 52, 54, 50, 50},
    {54, 52, 50, 56, 52, 52},
    {52, 54, 50, 54, 54, 50},
    {50, 52, 52, 52, 52, 48},
    {52, 50, 50, 54, 50, 50},
}
// ============================================
// DIFFICULTY LEVELS (8 tiers)
// ============================================
var DifficultyLevels = map[string]DifficultyConfig{
    "BEGINNER": {
        Name:              "Beginner",
        RatingSpread:      4.0,
        HomeAdvantage:     0.15,
        UpsetRate:         0.08,
        ScoringMultiplier: 2.0,
        DrawRate:          0.30,
    },
    "EASY": {
        Name:              "Easy",
        RatingSpread:      3.0,
        HomeAdvantage:     0.12,
        UpsetRate:         0.10,
        ScoringMultiplier: 1.5,
        DrawRate:          0.28,
    },
    "NORMAL": {
        Name:              "Normal",
        RatingSpread:      2.0,
        HomeAdvantage:     0.08,
        UpsetRate:         0.15,
        ScoringMultiplier: 1.0,
        DrawRate:          0.25,
    },
    "CHALLENGING": {
        Name:              "Challenging",
        RatingSpread:      1.5,
        HomeAdvantage:     0.06,
        UpsetRate:         0.20,
        ScoringMultiplier: 0.9,
        DrawRate:          0.22,
    },
    "HARD": {
        Name:              "Hard",
        RatingSpread:      1.0,
        HomeAdvantage:     0.05,
        UpsetRate:         0.25,
        ScoringMultiplier: 0.8,
        DrawRate:          0.20,
    },
    "EXPERT": {
        Name:              "Expert",
        RatingSpread:      0.8,
        HomeAdvantage:     0.03,
        UpsetRate:         0.30,
        ScoringMultiplier: 0.7,
        DrawRate:          0.18,
    },
    "MASTER": {
        Name:              "Master",
        RatingSpread:      0.5,
        HomeAdvantage:     0.02,
        UpsetRate:         0.35,
        ScoringMultiplier: 0.6,
        DrawRate:          0.15,
    },
    "GRANDMASTER": {
        Name:              "Grandmaster",
        RatingSpread:      0.3,
        HomeAdvantage:     0.01,
        UpsetRate:         0.40,
        ScoringMultiplier: 0.5,
        DrawRate:          0.12,
    },
}

// ============================================
// LEAGUE TYPES (Based on team count)
// ============================================
var LeagueTypes = map[string]LeagueTypeConfig{
    "MINI": {
        Name:           "Mini League",
        TotalWeeks:     8,
        TeamsCount:     5,
        PlayersPerTeam: 18,
        Style:          "Fast-paced, quick season, perfect for beginners",
    },
    "MICRO": {
        Name:           "Micro League",
        TotalWeeks:     12,
        TeamsCount:     7,
        PlayersPerTeam: 18,
        Style:          "Compact competition with intense rivalries",
    },
    "MINOR": {
        Name:           "Minor League",
        TotalWeeks:     18,
        TeamsCount:     10,
        PlayersPerTeam: 18,
        Style:          "Balanced league with good competition depth",
    },
    "MAJOR": {
        Name:           "Major League",
        TotalWeeks:     24,
        TeamsCount:     13,
        PlayersPerTeam: 18,
        Style:          "Extended season with tactical battles",
    },
    "PREMIER": {
        Name:           "Premier League",
        TotalWeeks:     30,
        TeamsCount:     16,
        PlayersPerTeam: 18,
        Style:          "Top-tier competition, elite teams battle",
    },
    "ELITE": {
        Name:           "Elite League",
        TotalWeeks:     38,
        TeamsCount:     20,
        PlayersPerTeam: 18,
        Style:          "The ultimate challenge, full season marathon",
    },
    "CUSTOM": {
        Name:           "Custom League",
        TotalWeeks:     0, // Calculated based on team count
        TeamsCount:     0, // User selected
        PlayersPerTeam: 18,
        Style:          "Your league, your rules",
    },
}

// ============================================
// LEAGUE COST MULTIPLIERS
// ============================================
var DifficultyCostMultiplier = map[string]float64{
    "BEGINNER":     0.8,
    "EASY":         0.9,
    "NORMAL":       1.0,
    "CHALLENGING":  1.1,
    "HARD":         1.2,
    "EXPERT":       1.4,
    "MASTER":       1.6,
    "GRANDMASTER":  2.0,
}

var TeamCountCostMultiplier = map[int]float64{
    5:  0.8,
    6:  0.85,
    7:  0.9,
    8:  0.95,
    9:  1.0,
    10: 1.0,
    11: 1.1,
    12: 1.2,
    13: 1.3,
    14: 1.4,
    15: 1.5,
    16: 1.6,
    17: 1.7,
    18: 1.8,
    19: 1.9,
    20: 2.0,
}

// ============================================
// REWARD MULTIPLIERS
// ============================================
var DifficultyRewardMultiplier = map[string]float64{
    "BEGINNER":     0.8,
    "EASY":         0.9,
    "NORMAL":       1.0,
    "CHALLENGING":  1.2,
    "HARD":         1.5,
    "EXPERT":       1.8,
    "MASTER":       2.2,
    "GRANDMASTER":  2.5,
}

// ============================================
// PLAYER FIRST NAMES (200 - Diverse & Realistic)
// ============================================
var FirstNames = []string{
    // English/UK (40)
    "David", "Marcus", "James", "Harry", "Jack", "Mason", "Oliver", "George",
    "Charlie", "Thomas", "William", "Daniel", "Ryan", "Callum", "Lewis", "Jamie",
    "Kieran", "Declan", "Jude", "Bukayo", "Phil", "Jordan", "Aaron", "Luke",
    "Nathan", "Ben", "Sam", "Joe", "Tom", "Alex", "Chris", "Matt",
    "Scott", "Kyle", "Liam", "Noah", "Ethan", "Jacob", "Reece", "Tyler",

    // Spanish/Latin (40)
    "Luis", "Carlos", "Diego", "Pedro", "Juan", "Miguel", "Rafael", "Jose",
    "Sergio", "Fernando", "Javier", "Alvaro", "Pablo", "Raul", "Iker", "Xavi",
    "Andres", "David", "Victor", "Mario", "Manuel", "Antonio", "Francisco", "Jorge",
    "Emilio", "Rodrigo", "Mateo", "Thiago", "Bruno", "Lucas", "Marco", "Nico",
    "Gonzalo", "Enzo", "Federico", "Maximiliano", "Santiago", "Sebastian", "Alejandro", "Ricardo",

    // Italian (30)
    "Marco", "Luca", "Andrea", "Alessandro", "Francesco", "Lorenzo", "Matteo",
    "Giuseppe", "Giovanni", "Antonio", "Mario", "Luigi", "Paolo", "Roberto",
    "Stefano", "Daniele", "Simone", "Federico", "Giorgio", "Nicolo", "Davide",
    "Christian", "Emanuele", "Filippo", "Gabriele", "Giacomo", "Leonardo", "Pietro", "Salvatore", "Vincenzo",

    // German (25)
    "Hans", "Franz", "Klaus", "Jurgen", "Karl", "Otto", "Felix", "Max",
    "Lukas", "Jonas", "Leon", "Paul", "Erik", "Finn", "Jan", "Tim",
    "Tom", "Ben", "Luis", "Moritz", "Jakob", "Emil", "Oskar", "Anton", "David",

    // French (25)
    "Pierre", "Jean", "Michel", "Antoine", "Francois", "Louis", "Henri",
    "Kylian", "Jules", "Hugo", "Lucas", "Leo", "Gabriel", "Arthur",
    "Raphael", "Adam", "Isaac", "Noah", "Ethan", "Nathan", "Theo", "Tom",
    "Mathis", "Enzo", "Maxime",

    // Dutch/Belgian (20)
    "Jan", "Pieter", "Willem", "Klaas", "Dirk", "Lars", "Sven", "Nils",
    "Frenkie", "Matthijs", "Virgil", "Memphis", "Donyell", "Cody", "Denzel",
    "Steven", "Kevin", "Wout", "Youri", "Leandro",

    // Portuguese/Brazilian (20)
    "Cristiano", "Bruno", "Bernardo", "Joao", "Ruben", "Diogo", "Goncalo",
    "Rafael", "Vinicius", "Rodrygo", "Neymar", "Gabriel", "Casemiro", "Alisson",
    "Ederson", "Richarlison", "Antony", "Martinelli", "Marquinhos", "Thiago",
}

// ============================================
// PLAYER LAST NAMES (200 - Diverse & Realistic)
// ============================================
var LastNames = []string{
    // English/UK (40)
    "Jones", "Smith", "Taylor", "Brown", "Wilson", "Davies", "Evans", "Thomas",
    "Johnson", "Walker", "Wright", "Robinson", "Thompson", "White", "Hughes", "Edwards",
    "Green", "Hall", "Wood", "Harris", "Martin", "Jackson", "Clarke", "James",
    "Kane", "Foden", "Rice", "Saka", "Grealish", "Mount", "Sterling", "Rashford",
    "Henderson", "Maguire", "Stones", "Walker", "Trippier", "Pickford", "Bellingham", "Maddison",

    // Spanish/Latin (40)
    "Garcia", "Rodriguez", "Lopez", "Perez", "Gonzalez", "Hernandez", "Torres", "Flores",
    "Ramirez", "Diaz", "Castro", "Morales", "Ortiz", "Gutierrez", "Mendoza", "Silva",
    "Romero", "Navarro", "Vargas", "Reyes", "Ruiz", "Jimenez", "Vega", "Molina",
    "Fernandez", "Alvarez", "Moreno", "Blanco", "Sanz", "Cruz", "Iglesias", "Santos",
    "Paredes", "Valverde", "Pedri", "Gavi", "Yamal", "Nico", "Olmo", "Merino",

    // Italian (30)
    "Rossi", "Russo", "Ferrari", "Colombo", "Bianchi", "Romano", "Ricci",
    "Marino", "Greco", "Bruno", "Gallo", "Conti", "De Luca", "Mancini",
    "Costa", "Giordano", "Rizzo", "Lombardi", "Moretti", "Barbieri", "Fontana",
    "Donnarumma", "Barella", "Chiesa", "Locatelli", "Bastoni", "Tonali", "Zaniolo", "Immobile", "Insigne",

    // German (25)
    "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner", "Becker",
    "Schulz", "Hoffmann", "Koch", "Richter", "Klein", "Wolf", "Schroder",
    "Neumann", "Schwarz", "Zimmermann", "Braun", "Kruger", "Hartmann", "Lange",
    "Werner", "Muller", "Kimmich", "Gundogan",

    // French (25)
    "Martin", "Bernard", "Dubois", "Thomas", "Robert", "Richard", "Petit",
    "Durand", "Leroy", "Moreau", "Simon", "Laurent", "Lefebvre", "Michel",
    "Garcia", "David", "Bertrand", "Roux", "Vincent", "Fournier", "Morel",
    "Girard", "Andre", "Lefevre", "Mercier",

    // Dutch/Belgian (20)
    "Jansen", "Vries", "Berg", "Bakker", "Dijk", "Smit", "Meijer",
    "Dumfries", "Depay", "Wijnaldum", "Ake", "Dumfries", "Berghuis", "Klaassen",
    "De Bruyne", "Hazard", "Lukaku", "Tielemans", "Carrasco", "Mertens",

    // Portuguese/Brazilian (20)
    "Silva", "Santos", "Ferreira", "Pereira", "Oliveira", "Costa", "Rodrigues",
    "Martins", "Jesus", "Sousa", "Fernandes", "Goncalves", "Alves", "Gomes",
    "Ronaldo", "Felix", "Leao", "Neves", "Vitinha", "Palhinha",
}