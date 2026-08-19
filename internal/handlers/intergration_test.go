// internal/handlers/integration_test.go

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/betking/rich-backend/config"
	"github.com/betking/rich-backend/internal/database"
	"github.com/betking/rich-backend/internal/middleware"
	"github.com/betking/rich-backend/internal/models"
	"github.com/betking/rich-backend/internal/websocket"
	"github.com/go-chi/chi/v5"
)

// setupTestHandler - Creates handler with real DB
func setupTestHandler() *Handler {
    cfg := config.Load()
    
    pg, err := database.NewPostgres(cfg.PostgresDSN)
    if err != nil {
        return &Handler{DB: nil}
    }
    
    redis := database.NewRedis(cfg.RedisAddr)
    
    return &Handler{
        DB:     pg,
        Redis:  redis,
        Config: cfg,
        WSHub:  websocket.NewHub(),
    }
}

func TestForfeitLeagueIdempotent(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    h := setupTestHandler()
    if h.DB == nil {
        t.Skip("Database not available")
    }
    
    leagueID := "LEAGUE_nonexistent_test_123"
    
    createRequest := func() (*httptest.ResponseRecorder, *http.Request) {
        w := httptest.NewRecorder()
        req := httptest.NewRequest("POST", "/api/v1/leagues/"+leagueID+"/forfeit", nil)
        
        user := &models.User{ID: "USER_test"}
        ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
        req = req.WithContext(ctx)
        
        chiCtx := chi.NewRouteContext()
        chiCtx.URLParams.Add("leagueID", leagueID)
        req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
        
        return w, req
    }
    
    // First call - should return "already_deleted" (200)
    w1, req1 := createRequest()
    h.ForfeitLeague(w1, req1)
    
    if w1.Code != http.StatusOK {
        t.Errorf("First call: expected 200, got %d", w1.Code)
    }
    
    // Second call - should also return 200 (idempotent)
    w2, req2 := createRequest()
    h.ForfeitLeague(w2, req2)
    
    if w2.Code != http.StatusOK {
        t.Errorf("Second call: expected 200, got %d", w2.Code)
    }
    
    // Check response body
    var resp1 map[string]interface{}
    json.Unmarshal(w1.Body.Bytes(), &resp1)
    
    if resp1["status"] != "already_deleted" {
        t.Errorf("Expected 'already_deleted', got %v", resp1["status"])
    }
}

// internal/handlers/integration_test.go

func TestRegisterValidation(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    h := setupTestHandler()
    if h.DB == nil {
        t.Skip("Database not available")
    }
    
    // Use short unique suffix
    timestamp := fmt.Sprintf("%d", time.Now().Unix()%100000)
    
    tests := []struct {
        name       string
        body       string
        wantStatus int
    }{
        {"Valid", fmt.Sprintf(`{"username":"tu%s","password":"pass123","gamename":"Test"}`, timestamp), http.StatusCreated},
        {"Empty username", `{"username":"","password":"pass123","gamename":"Test"}`, http.StatusBadRequest},
        {"Short password", fmt.Sprintf(`{"username":"tu2%s","password":"1","gamename":"Test"}`, timestamp), http.StatusBadRequest},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := httptest.NewRecorder()
            req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(tt.body))
            req.Header.Set("Content-Type", "application/json")
            
            h.Register(w, req)
            
            if w.Code != tt.wantStatus {
                t.Errorf("Expected %d, got %d", tt.wantStatus, w.Code)
                t.Logf("Response: %s", w.Body.String())
            }
        })
    }
}

func TestCurrencyNormalization(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"usd", "kash"},
        {"kash", "kash"},
        {"betpoints", "points"},
        {"bp", "points"},
        {"blings", "coins"},
        {"bling", "coins"},
        {"coins", "coins"},
        {"invalid", ""},
    }
    
    for _, tt := range tests {
        result := normalizeCurrencyName(tt.input)
        if result != tt.expected {
            t.Errorf("normalizeCurrencyName(%s) = %s, expected %s", tt.input, result, tt.expected)
        }
    }
}