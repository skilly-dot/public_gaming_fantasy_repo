// internal/database/postgres.go
package database

import (
    "log"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

type PostgresDB struct {
    *sqlx.DB
}

func NewPostgres(dsn string) (*PostgresDB, error) {
    db, err := sqlx.Connect("postgres", dsn)
    if err != nil { return nil, err }
    db.SetMaxOpenConns(50)
    db.SetMaxIdleConns(25)
    return &PostgresDB{db}, nil
}

func (p *PostgresDB) RunMigrations() {
    p.MustExec(`
    CREATE TABLE IF NOT EXISTS users (
        id TEXT PRIMARY KEY, username TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL, gamename TEXT,
        is_admin BOOLEAN DEFAULT false, created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS sessions (
        id TEXT PRIMARY KEY, user_id TEXT REFERENCES users(id),
        token TEXT UNIQUE NOT NULL, created_at TIMESTAMP DEFAULT NOW(),expires_at TIMESTAMP
    );
    CREATE TABLE IF NOT EXISTS wallets (
        user_id TEXT PRIMARY KEY REFERENCES users(id),
        kash DOUBLE PRECISION DEFAULT 1000, points DOUBLE PRECISION DEFAULT 100, coins DOUBLE PRECISION DEFAULT 10,
        last_daily_bonus TIMESTAMP, daily_streak INTEGER DEFAULT 0
    );
    CREATE TABLE IF NOT EXISTS leagues (
        id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL,
        difficulty TEXT DEFAULT 'NORMAL', total_weeks INTEGER NOT NULL,
        user_id TEXT REFERENCES users(id), 
        day_number INTEGER DEFAULT 1,
        current_week INTEGER DEFAULT 1,
        status TEXT DEFAULT 'ACTIVE', created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS teams (
        id TEXT PRIMARY KEY, league_id TEXT REFERENCES leagues(id),
        name TEXT NOT NULL, rating DOUBLE PRECISION NOT NULL,
        formation TEXT DEFAULT '4-3-3',
        attack INTEGER, midfield INTEGER, defense INTEGER,
        teamwork INTEGER, experience INTEGER, set_pieces INTEGER
    );
    CREATE TABLE IF NOT EXISTS coaches (
        id TEXT PRIMARY KEY, league_id TEXT REFERENCES leagues(id),
        team_id TEXT REFERENCES teams(id),
        name TEXT NOT NULL, nationality TEXT,
        formation TEXT, playstyle TEXT, mentality TEXT,
        attacking INTEGER, defending INTEGER, tactical INTEGER, motivation INTEGER,
        abilities TEXT
    );
    CREATE TABLE IF NOT EXISTS players (
        id TEXT PRIMARY KEY, league_id TEXT REFERENCES leagues(id),
        team_id TEXT REFERENCES teams(id),
        name TEXT NOT NULL, position TEXT NOT NULL, number INTEGER,
        rating DOUBLE PRECISION,
        pace INTEGER, shooting INTEGER, passing INTEGER, dribbling INTEGER,
        defending INTEGER, physical INTEGER, stamina INTEGER,
        form DOUBLE PRECISION, traits TEXT
    );
    CREATE TABLE IF NOT EXISTS fixtures (
        id TEXT PRIMARY KEY, league_id TEXT REFERENCES leagues(id),
        week_number INTEGER NOT NULL,
        home_team_id TEXT, away_team_id TEXT,
        home_score INTEGER DEFAULT -1, away_score INTEGER DEFAULT -1,
        winner TEXT DEFAULT '', status TEXT DEFAULT 'SCHEDULED',
        kickoff_time BIGINT
    );
    CREATE TABLE IF NOT EXISTS match_odds (
        id TEXT PRIMARY KEY, league_id TEXT REFERENCES leagues(id),
        fixture_id TEXT REFERENCES fixtures(id),
        home_win DOUBLE PRECISION, draw DOUBLE PRECISION, away_win DOUBLE PRECISION,
        home_odds DOUBLE PRECISION, draw_odds DOUBLE PRECISION, away_odds DOUBLE PRECISION
    );
    CREATE TABLE IF NOT EXISTS match_results (
        id TEXT PRIMARY KEY, league_id TEXT REFERENCES leagues(id),
        fixture_id TEXT REFERENCES fixtures(id), week_number INTEGER,
        home_team_id TEXT, away_team_id TEXT,
        home_score INTEGER, away_score INTEGER, winner TEXT,
        goals JSONB, stats JSONB, created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS league_table (
        team_id TEXT, league_id TEXT REFERENCES leagues(id),
        team_name TEXT, played INTEGER DEFAULT 0,
        won INTEGER DEFAULT 0, drawn INTEGER DEFAULT 0, lost INTEGER DEFAULT 0,
        goals_for INTEGER DEFAULT 0, goals_against INTEGER DEFAULT 0,
        points INTEGER DEFAULT 0, PRIMARY KEY (team_id, league_id)
    );
    CREATE TABLE IF NOT EXISTS daily_matches (
        id SERIAL PRIMARY KEY, league_id TEXT REFERENCES leagues(id),
        day_number INTEGER NOT NULL, fixture_id TEXT REFERENCES fixtures(id),
        kickoff_time TIMESTAMP NOT NULL, status TEXT DEFAULT 'SCHEDULED',
        home_score INTEGER DEFAULT -1, away_score INTEGER DEFAULT -1,
        winner TEXT DEFAULT '', created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS bets (
        id TEXT PRIMARY KEY, user_id TEXT REFERENCES users(id),
        league_id TEXT REFERENCES leagues(id), week INTEGER,
        bets JSONB, amount DOUBLE PRECISION, total_odds DOUBLE PRECISION,
        is_custom BOOLEAN DEFAULT false, status TEXT DEFAULT 'PENDING',
        payout DOUBLE PRECISION DEFAULT 0, tax DOUBLE PRECISION DEFAULT 0,
        points INTEGER DEFAULT 0, coins INTEGER DEFAULT 0,
        placed_at TIMESTAMP DEFAULT NOW(), settled_at TIMESTAMP
    );
    ALTER TABLE bets ADD COLUMN IF NOT EXISTS is_winner_bet BOOLEAN DEFAULT false;
    CREATE TABLE IF NOT EXISTS admin_bets (
        id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT,
        options JSONB, expires_at TIMESTAMP,
        status TEXT DEFAULT 'OPEN', result TEXT DEFAULT '',
        created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS top_scorers (
        league_id TEXT NOT NULL,
        player_name TEXT NOT NULL,
        goals INTEGER DEFAULT 1,
        PRIMARY KEY (league_id, player_name)
    );
    CREATE TABLE IF NOT EXISTS user_admin_bets (
        id TEXT PRIMARY KEY, user_id TEXT REFERENCES users(id),
        admin_bet_id TEXT REFERENCES admin_bets(id),
        prediction TEXT NOT NULL, amount DOUBLE PRECISION NOT NULL,
        odds DOUBLE PRECISION DEFAULT 1.0, status TEXT DEFAULT 'PENDING',
        payout DOUBLE PRECISION DEFAULT 0, placed_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS notifications (
        id SERIAL PRIMARY KEY, user_id TEXT REFERENCES users(id),
        title TEXT, message TEXT, type TEXT DEFAULT 'system',
        is_read BOOLEAN DEFAULT false, created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS admin_match_bets (
        id TEXT PRIMARY KEY,
        admin_id TEXT REFERENCES users(id),
        title TEXT NOT NULL,
        description TEXT,
        status TEXT DEFAULT 'OPEN',
        created_at TIMESTAMP DEFAULT NOW(),
        updated_at TIMESTAMP DEFAULT NOW(),
        deleted_at TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS admin_matches (
        id TEXT PRIMARY KEY,
        admin_bet_id TEXT REFERENCES admin_match_bets(id) ON DELETE CASCADE,
        match_index INTEGER NOT NULL,
        home_team TEXT NOT NULL,
        away_team TEXT NOT NULL,
        home_odds DECIMAL NOT NULL,
        draw_odds DECIMAL NOT NULL,
        away_odds DECIMAL NOT NULL,
        home_score INTEGER,
        away_score INTEGER,
        result TEXT,
        status TEXT DEFAULT 'SCHEDULED',
        created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE TABLE IF NOT EXISTS quick_matches (
        id TEXT PRIMARY KEY,
        league_id TEXT REFERENCES leagues(id),
        user_id TEXT REFERENCES users(id),
        home_team_id TEXT,
        away_team_id TEXT,
        home_odds DOUBLE PRECISION,
        draw_odds DOUBLE PRECISION,
        away_odds DOUBLE PRECISION,
        home_score INTEGER DEFAULT -1,
        away_score INTEGER DEFAULT -1,
        winner TEXT DEFAULT '',
        status TEXT DEFAULT 'PENDING',
        created_at TIMESTAMP DEFAULT NOW()
    );
    ALTER TABLE bets ADD COLUMN IF NOT EXISTS is_quick_match BOOLEAN DEFAULT false;
    ALTER TABLE bets ADD COLUMN IF NOT EXISTS quick_match_id TEXT;
    CREATE TABLE IF NOT EXISTS user_admin_match_bets (
        id TEXT PRIMARY KEY,
        user_id TEXT REFERENCES users(id),
        admin_bet_id TEXT REFERENCES admin_match_bets(id),
        match_picks JSONB NOT NULL,
        total_odds DECIMAL NOT NULL,
        amount DECIMAL NOT NULL,
        potential_win DECIMAL NOT NULL,
        status TEXT DEFAULT 'PENDING',
        payout DECIMAL DEFAULT 0,
        placed_at TIMESTAMP DEFAULT NOW(),
        settled_at TIMESTAMP
    );

    -- ============================================
    -- 50-50 BINARY BETS (NEW)
    -- ============================================
    CREATE TABLE IF NOT EXISTS fifty_fifty_bets (
        id TEXT PRIMARY KEY,
        admin_id TEXT REFERENCES users(id),
        title TEXT NOT NULL,
        description TEXT,
        yes_odds DECIMAL DEFAULT 1.85,
        no_odds DECIMAL DEFAULT 1.85,
        status TEXT DEFAULT 'OPEN',
        result TEXT,
        created_at TIMESTAMP DEFAULT NOW(),
        expires_at TIMESTAMP,
        settled_at TIMESTAMP,
        deleted_at TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS user_fifty_fifty_bets (
        id TEXT PRIMARY KEY,
        user_id TEXT REFERENCES users(id),
        fifty_fifty_id TEXT REFERENCES fifty_fifty_bets(id),
        prediction TEXT NOT NULL,
        odds DECIMAL NOT NULL,
        amount DECIMAL NOT NULL,
        potential_win DECIMAL NOT NULL,
        status TEXT DEFAULT 'PENDING',
        payout DECIMAL DEFAULT 0,
        placed_at TIMESTAMP DEFAULT NOW(),
        settled_at TIMESTAMP
    );
    `)
    log.Println("Database migrations completed")
}

func (p *PostgresDB) AddPerformanceIndexes() {
    indexes := []string{
        "CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)",
        "CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
        "CREATE INDEX IF NOT EXISTS idx_teams_league ON teams(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_players_league ON players(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_players_team ON players(team_id)",
        "CREATE INDEX IF NOT EXISTS idx_fixtures_league ON fixtures(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_fixtures_week ON fixtures(league_id, week_number)",
        "CREATE INDEX IF NOT EXISTS idx_results_league ON match_results(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_odds_league ON match_odds(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_table_league ON league_table(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_bets_user ON bets(user_id)",
        "CREATE INDEX IF NOT EXISTS idx_bets_status ON bets(status)",
        "CREATE INDEX IF NOT EXISTS idx_wallets_user ON wallets(user_id)",
        "CREATE INDEX IF NOT EXISTS idx_notif_user ON notifications(user_id)",
        "CREATE INDEX IF NOT EXISTS idx_leagues_user ON leagues(user_id)",
        "CREATE INDEX IF NOT EXISTS idx_daily_matches_league ON daily_matches(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_daily_matches_status ON daily_matches(status)",
        "CREATE INDEX IF NOT EXISTS idx_coaches_league ON coaches(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_coaches_team ON coaches(team_id)",
        "CREATE INDEX IF NOT EXISTS idx_top_scorers_league ON top_scorers(league_id)",
        "CREATE INDEX IF NOT EXISTS idx_top_scorers_goals ON top_scorers(league_id, goals DESC)",
    }
    for _, idx := range indexes {
        _, err := p.Exec(idx)
        if err != nil {
            log.Printf(" Index warning: %v", err)
        }
    }
    log.Println(" Performance indexes created")
}

func (p *PostgresDB) AddNewIndexes() {
    newIndexes := []string{
        // Admin match indexes
        "CREATE INDEX IF NOT EXISTS idx_admin_match_bets_status ON admin_match_bets(status)",
        "CREATE INDEX IF NOT EXISTS idx_admin_match_bets_admin ON admin_match_bets(admin_id)",
        "CREATE INDEX IF NOT EXISTS idx_admin_matches_bet ON admin_matches(admin_bet_id)",
        "CREATE INDEX IF NOT EXISTS idx_admin_matches_status ON admin_matches(status)",
        "CREATE INDEX IF NOT EXISTS idx_user_admin_match_bets_user ON user_admin_match_bets(user_id)",
        "CREATE INDEX IF NOT EXISTS idx_user_admin_match_bets_status ON user_admin_match_bets(status)",
        
        // 50-50 indexes
        "CREATE INDEX IF NOT EXISTS idx_fifty_fifty_bets_status ON fifty_fifty_bets(status)",
        "CREATE INDEX IF NOT EXISTS idx_fifty_fifty_bets_admin ON fifty_fifty_bets(admin_id)",
        "CREATE INDEX IF NOT EXISTS idx_user_fifty_fifty_bets_user ON user_fifty_fifty_bets(user_id)",
        "CREATE INDEX IF NOT EXISTS idx_user_fifty_fifty_bets_status ON user_fifty_fifty_bets(status)",
        "CREATE INDEX IF NOT EXISTS idx_user_fifty_fifty_bets_fifty ON user_fifty_fifty_bets(fifty_fifty_id)",
    }
    
    for _, idx := range newIndexes {
        _, err := p.Exec(idx)
        if err != nil {
            log.Printf(" New index warning: %v", err)
        }
    }
    log.Println(" New indexes created")
}