// internal/handlers/security_test.go

package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/betking/rich-backend/internal/middleware"
	"github.com/betking/rich-backend/internal/models"
)

func TestInvalidSessionToken(t *testing.T) {
    h := setupTestHandler()
    if h.DB == nil {
        t.Skip("Database not available")
    }
    
    tests := []struct {
        name  string
        token string
    }{
        {"Empty token", ""},
        {"Fake token", "fake-token-123"},
        {"SQL injection", "' OR '1'='1"},
        {"XSS attempt", "<script>alert('xss')</script>"},
        {"Long token", string(make([]byte, 1000))},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := httptest.NewRecorder()
            req := httptest.NewRequest("GET", "/api/v1/wallet", nil)
            req.Header.Set("Authorization", "Bearer "+tt.token)
            
            // Simulate middleware
            middleware.AuthMiddleware(h.DB, h.Redis)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
            })).ServeHTTP(w, req)
            
            if w.Code != http.StatusUnauthorized {
                t.Errorf("Expected 401 for token '%s', got %d", tt.name, w.Code)
            }
        })
    }
}

func TestSQLInjectionInUsername(t *testing.T) {
    h := setupTestHandler()
    if h.DB == nil {
        t.Skip("Database not available")
    }
    
    maliciousUsernames := []string{
        "' OR '1'='1",
        "'; DROP TABLE users; --",
        "admin' --",
        "' UNION SELECT * FROM users --",
        "<script>alert('xss')</script>",
        "'; DELETE FROM wallets; --",
    }
    
    for _, username := range maliciousUsernames {
        t.Run(username, func(t *testing.T) {
            w := httptest.NewRecorder()
            req := httptest.NewRequest("POST", "/api/v1/auth/register", nil)
            req.Header.Set("Content-Type", "application/json")
            
            h.Register(w, req)
            
            // Should NOT be 201 (should reject)
            if w.Code == http.StatusCreated {
                t.Errorf("SQL injection accepted: %s", username)
            }
        })
    }
}

func TestAppKeyRequired(t *testing.T) {
    // Set APP_SECRET for this test
    os.Setenv("APP_SECRET", "test_secret_123")
    os.Setenv("ENVIRONMENT", "production")
    defer os.Unsetenv("APP_SECRET")
    defer os.Unsetenv("ENVIRONMENT")
    
    // Test WITHOUT app key
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/wallet", nil)
    
    middleware.AppKeyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })).ServeHTTP(w, req)
    
    if w.Code != http.StatusForbidden {
        t.Errorf("Request without app key should be blocked, got %d", w.Code)
    }
    
    // Test with WRONG app key
    w2 := httptest.NewRecorder()
    req2 := httptest.NewRequest("GET", "/api/v1/wallet", nil)
    req2.Header.Set("X-App-Key", "wrong_key")
    
    middleware.AppKeyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })).ServeHTTP(w2, req2)
    
    if w2.Code != http.StatusForbidden {
        t.Errorf("Wrong app key should be blocked, got %d", w2.Code)
    }
    
    // Test with CORRECT app key
    w3 := httptest.NewRecorder()
    req3 := httptest.NewRequest("GET", "/api/v1/wallet", nil)
    req3.Header.Set("X-App-Key", "test_secret_123")
    
    middleware.AppKeyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })).ServeHTTP(w3, req3)
    
    if w3.Code != http.StatusOK {
        t.Errorf("Correct app key should pass, got %d", w3.Code)
    }
}

func TestSessionExpiry(t *testing.T) {
    h := setupTestHandler()
    if h.DB == nil {
        t.Skip("Database not available")
    }
    
    // Test with expired session
    h.DB.Exec("INSERT INTO sessions (id, user_id, token, expires_at) VALUES ($1, $2, $3, NOW() - INTERVAL '1 day')",
        "EXPIRED_SESSION", "USER_test", "expired_token_123")
    
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/wallet", nil)
    req.Header.Set("Authorization", "Bearer expired_token_123")
    
    middleware.AuthMiddleware(h.DB, h.Redis)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })).ServeHTTP(w, req)
    
    // Clean up
    h.DB.Exec("DELETE FROM sessions WHERE id='EXPIRED_SESSION'")
    
    if w.Code != http.StatusUnauthorized {
        t.Errorf("Expired session should be rejected, got %d", w.Code)
    }
}

func TestAdminAccessControl(t *testing.T) {
    tests := []struct {
        name    string
        isAdmin bool
        wantOK  bool
    }{
        {"Admin user", true, true},
        {"Regular user", false, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := httptest.NewRecorder()
            req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
            
            user := &models.User{ID: "USER_test", IsAdmin: tt.isAdmin}
            ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
            req = req.WithContext(ctx)
            
            middleware.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
            })).ServeHTTP(w, req)
            
            if tt.wantOK && w.Code != http.StatusOK {
                t.Errorf("Admin should have access, got %d", w.Code)
            }
            if !tt.wantOK && w.Code == http.StatusOK {
                t.Error("Regular user should NOT have admin access")
            }
        })
    }
}

func TestRateLimiting(t *testing.T) {
    // Test rate limiter
    limiter := middleware.NewRateLimiter()
    
    // Allow first 5 requests
    for i := 0; i < 5; i++ {
        if !limiter.Allow("127.0.0.1", 5) {
            t.Errorf("Request %d should be allowed", i+1)
        }
    }
    
    // 6th request should be blocked
    if limiter.Allow("127.0.0.1", 5) {
        t.Error("6th request should be blocked")
    }
}