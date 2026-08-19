-- internal/database/migrations/00002_admin_and_fifty_fifty.sql
-- +goose Up

-- ============================================
-- ADMIN MATCH BETS (1X2 format, same as league)
-- ============================================

-- Parent table: Groups matches into a betting slip
CREATE TABLE IF NOT EXISTS admin_match_bets (
    id TEXT PRIMARY KEY,
    admin_id TEXT REFERENCES users(id),
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'OPEN',           -- OPEN/LOCKED/SETTLED/DELETED
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Individual matches within an admin bet slip
CREATE TABLE IF NOT EXISTS admin_matches (
    id TEXT PRIMARY KEY,
    admin_bet_id TEXT REFERENCES admin_match_bets(id) ON DELETE CASCADE,
    match_index INTEGER NOT NULL,         -- Order within the slip (1, 2, 3...)
    
    -- Match data (EXACT same structure as daily_matches)
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    home_odds DECIMAL NOT NULL,
    draw_odds DECIMAL NOT NULL,
    away_odds DECIMAL NOT NULL,
    
    -- Settlement fields
    home_score INTEGER,
    away_score INTEGER,
    result TEXT,                          -- HOME_WIN/DRAW/AWAY_WIN
    status TEXT DEFAULT 'SCHEDULED',      -- SCHEDULED/LOCKED/COMPLETED
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- User bets on admin matches
CREATE TABLE IF NOT EXISTS user_admin_match_bets (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    admin_bet_id TEXT REFERENCES admin_match_bets(id),
    
    -- Multiple match picks in JSON format
    match_picks JSONB NOT NULL,           -- [{"match_id":"xxx","home_team":"Man Utd","away_team":"Arsenal","prediction":"HOME_WIN","odds":1.85}]
    total_odds DECIMAL NOT NULL,
    amount DECIMAL NOT NULL,
    potential_win DECIMAL NOT NULL,       -- amount * total_odds * 0.88 (after 12% tax)
    
    status TEXT DEFAULT 'PENDING',        -- PENDING/WON/LOST
    payout DECIMAL DEFAULT 0,
    
    placed_at TIMESTAMP DEFAULT NOW(),
    settled_at TIMESTAMP
);

-- ============================================
-- 50-50 BINARY BETS (YES/NO propositions)
-- ============================================

-- Admin-created 50-50 propositions
CREATE TABLE IF NOT EXISTS fifty_fifty_bets (
    id TEXT PRIMARY KEY,
    admin_id TEXT REFERENCES users(id),
    
    -- The question/proposition
    title TEXT NOT NULL,                  -- "Will Trump bomb Iran today?"
    description TEXT,                     -- Additional context
    
    -- Fixed odds (always 1.85 for both sides)
    yes_odds DECIMAL DEFAULT 1.85,
    no_odds DECIMAL DEFAULT 1.85,
    
    -- Status and settlement
    status TEXT DEFAULT 'OPEN',           -- OPEN/LOCKED/SETTLED/DELETED
    result TEXT,                          -- YES/NO
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    settled_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- User bets on 50-50 propositions
CREATE TABLE IF NOT EXISTS user_fifty_fifty_bets (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    fifty_fifty_id TEXT REFERENCES fifty_fifty_bets(id),
    
    -- Simple binary choice
    prediction TEXT NOT NULL,             -- YES or NO
    odds DECIMAL NOT NULL,                -- 1.85
    
    amount DECIMAL NOT NULL,
    potential_win DECIMAL NOT NULL,       -- amount * 1.85 * 0.88
    
    status TEXT DEFAULT 'PENDING',        -- PENDING/WON/LOST
    payout DECIMAL DEFAULT 0,
    
    placed_at TIMESTAMP DEFAULT NOW(),
    settled_at TIMESTAMP
);

-- ============================================
-- INDEXES
-- ============================================

-- Admin match indexes
CREATE INDEX IF NOT EXISTS idx_admin_match_bets_status ON admin_match_bets(status);
CREATE INDEX IF NOT EXISTS idx_admin_match_bets_admin ON admin_match_bets(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_matches_bet ON admin_matches(admin_bet_id);
CREATE INDEX IF NOT EXISTS idx_admin_matches_status ON admin_matches(status);
CREATE INDEX IF NOT EXISTS idx_user_admin_match_bets_user ON user_admin_match_bets(user_id);
CREATE INDEX IF NOT EXISTS idx_user_admin_match_bets_status ON user_admin_match_bets(status);

-- 50-50 indexes
CREATE INDEX IF NOT EXISTS idx_fifty_fifty_bets_status ON fifty_fifty_bets(status);
CREATE INDEX IF NOT EXISTS idx_fifty_fifty_bets_admin ON fifty_fifty_bets(admin_id);
CREATE INDEX IF NOT EXISTS idx_user_fifty_fifty_bets_user ON user_fifty_fifty_bets(user_id);
CREATE INDEX IF NOT EXISTS idx_user_fifty_fifty_bets_status ON user_fifty_fifty_bets(status);
CREATE INDEX IF NOT EXISTS idx_user_fifty_fifty_bets_fifty ON user_fifty_fifty_bets(fifty_fifty_id);

-- +goose Down
DROP TABLE IF EXISTS user_fifty_fifty_bets;
DROP TABLE IF EXISTS fifty_fifty_bets;
DROP TABLE IF EXISTS user_admin_match_bets;
DROP TABLE IF EXISTS admin_matches;
DROP TABLE IF EXISTS admin_match_bets;