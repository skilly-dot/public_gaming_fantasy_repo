package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    type Wallet struct {
        Kash   float64 `json:"kash"`
        Points float64 `json:"points"`
        Coins  float64 `json:"coins"`
    }
    
    var wallet Wallet
    err := h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", user.ID)
    if err != nil {
        // Create wallet if doesn't exist (first time user)
        h.DB.Exec("INSERT INTO wallets (user_id, kash, points, coins) VALUES ($1, 1000, 100, 10)", user.ID)
        wallet = Wallet{Kash: 1000, Points: 100, Coins: 10}
    }
    
    respondJSON(w, http.StatusOK, wallet)
}

func (h *Handler) SyncWallet(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    var req struct {
        Kash   float64 `json:"kash"`
        Points float64 `json:"points"`
        Coins  float64 `json:"coins"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    // Check wallet exists
    var exists bool
    h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id=$1)", user.ID).Scan(&exists)
    
    if exists {
        h.DB.Exec("UPDATE wallets SET kash=$1, points=$2, coins=$3 WHERE user_id=$4",
            req.Kash, req.Points, req.Coins, user.ID)
    } else {
        h.DB.Exec("INSERT INTO wallets (user_id, kash, points, coins) VALUES ($1, $2, $3, $4)",
            user.ID, req.Kash, req.Points, req.Coins)
    }
    
    var wallet struct{ Kash, Points, Coins float64 }
    h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", user.ID)
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status": "synced",
        "wallet": wallet,
    })
}

func (h *Handler) DailyBonus(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    
    // Check if already claimed today
    var lastClaim time.Time
    err := h.DB.QueryRow("SELECT last_daily_bonus FROM wallets WHERE user_id=$1", user.ID).Scan(&lastClaim)
    if err == nil {
        today := time.Now().Truncate(24 * time.Hour)
        if lastClaim.Truncate(24 * time.Hour).Equal(today) {
            respondError(w, http.StatusTooManyRequests, "Already claimed today")
            return
        }
    }
    
    // Calculate streak
    var streak int
    h.DB.QueryRow("SELECT daily_streak FROM wallets WHERE user_id=$1", user.ID).Scan(&streak)
    
    // If last claim was yesterday, increment streak
    if err == nil && lastClaim.Truncate(24*time.Hour).Equal(time.Now().Add(-24*time.Hour).Truncate(24*time.Hour)) {
        streak++
    } else {
        streak = 1
    }
    
    // Bonus amounts based on streak
    bonusKash := 50.0 + float64(streak)*25.0  // 75, 100, 125, 150, 175, 200, 250
    bonusPoints := 20.0 + float64(streak)*10.0 // 30, 40, 50, 60, 70, 80, 90
    bonusCoins := 2.0 + float64(streak)*1.0    // 3, 4, 5, 6, 7, 8, 9
    
    if streak >= 7 {
        bonusKash += 500  // Big bonus for 7-day streak
        bonusPoints += 200
        bonusCoins += 20
    }
    
    // Update wallet
    h.DB.Exec(`UPDATE wallets SET 
        kash = kash + $1, 
        points = points + $2, 
        coins = coins + $3,
        last_daily_bonus = $4,
        daily_streak = $5
        WHERE user_id = $6`,
        bonusKash, bonusPoints, bonusCoins, time.Now(), streak%7, user.ID)
    
    var wallet struct{ Kash, Points, Coins float64 }
    h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", user.ID)
    
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status": "claimed",
        "streak": streak,
        "day": streak%7,
        "bonus": map[string]float64{
            "kash": bonusKash,
            "points": bonusPoints,
            "coins": bonusCoins,
        },
        "wallet": wallet,
        "message": "Daily bonus claimed! Come back tomorrow for more.",
    })
}

// POST /api/v1/wallet/exchange
func (h *Handler) ExchangeCurrency(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    var req struct {
        From   string  `json:"from"`
        To     string  `json:"to"`
        Amount float64 `json:"amount"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    if req.Amount <= 0 {
        respondError(w, http.StatusBadRequest, "Amount must be positive")
        return
    }

    // Normalize names
    from := normalizeCurrencyName(req.From)
    to := normalizeCurrencyName(req.To)
    
    if from == "" || to == "" || from == to {
        respondError(w, http.StatusBadRequest, "Invalid conversion")
        return
    }

    // Exchange rates
    rates := map[string]map[string]float64{
        "kash": {
            "points": 8.0,
            "coins": 1.6,
        },
        "points": {
            "kash": 0.1,
            "coins": 0.05,
        },
        "coins": {
            "kash": 0.5,
            "points": 20.0,
        },
    }

    rate, ok := rates[from][to]
    if !ok {
        respondError(w, http.StatusBadRequest, "Invalid conversion")
        return
    }

    toAmount := req.Amount * rate

    // Check balance
    var balance float64
    err := h.DB.QueryRow("SELECT "+from+" FROM wallets WHERE user_id=$1", user.ID).Scan(&balance)
    if err != nil || balance < req.Amount {
        respondError(w, http.StatusBadRequest, "Insufficient balance")
        return
    }

    // Update wallet
    h.DB.Exec("UPDATE wallets SET "+from+" = "+from+" - $1, "+to+" = "+to+" + $2 WHERE user_id=$3",
        req.Amount, toAmount, user.ID)

    var wallet struct{ Kash, Points, Coins float64 }
    h.DB.Get(&wallet, "SELECT kash, points, coins FROM wallets WHERE user_id=$1", user.ID)

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "from": from,
        "to": to,
        "amount": req.Amount,
        "received": toAmount,
        "wallet": map[string]float64{
            "kash": wallet.Kash,
            "points": wallet.Points,
            "coins": wallet.Coins,
        },
    })
}

// Helper to normalize currency names
func normalizeCurrencyName(name string) string {
    switch strings.ToLower(name) {
    case "usd", "kash", "dollars":
        return "kash"
    case "bp", "betpoints", "points":
        return "points"
    case "blings", "bling", "coins":
        return "coins"
    default:
        return ""
    }
}