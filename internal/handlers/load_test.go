// internal/handlers/load_test.go

package handlers

import (
    "testing"
    "sync"
    "fmt"
    "context"
    "net/http"
    "net/http/httptest"
    
    "github.com/betking/rich-backend/internal/middleware"
    "github.com/betking/rich-backend/internal/models"
)

func TestConcurrentBetPlacement(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test")
    }
    
    h := setupTestHandler()
    if h.DB == nil {
        t.Skip("Database not available")
    }
    
    var wg sync.WaitGroup
    errors := make(chan error, 100)
    
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            w := httptest.NewRecorder()
            req := httptest.NewRequest("POST", "/api/v1/bets/place", nil)
            req.Header.Set("Content-Type", "application/json")
            
            // Add user to context
            user := &models.User{ID: "USER_test"}
            ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
            req = req.WithContext(ctx)
            
            h.PlaceBet(w, req)
            
            if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
                errors <- fmt.Errorf("unexpected status: %d", w.Code)
            }
        }()
    }
    
    wg.Wait()
    close(errors)
    
    for err := range errors {
        t.Error(err)
    }
}

func TestConcurrentWalletFetch(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test")
    }
    
    h := setupTestHandler()
    if h.DB == nil {
        t.Skip("Database not available")
    }
    
    var wg sync.WaitGroup
    
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            w := httptest.NewRecorder()
            req := httptest.NewRequest("GET", "/api/v1/wallet", nil)
            
            // ADD USER TO CONTEXT - THIS WAS MISSING
            user := &models.User{ID: "USER_test"}
            ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
            req = req.WithContext(ctx)
            
            h.GetWallet(w, req)
            
            if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
                t.Errorf("Expected 200 or 500, got %d", w.Code)
            }
        }()
    }
    
    wg.Wait()
}