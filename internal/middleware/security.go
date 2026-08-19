// internal/middleware/security.go - FULL PURPOSE

package middleware

import (
    "net/http"
    "os"
    "strings"
)

// SecurityHeaders - Adds protection headers to every response
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Allow OPTIONS for CORS preflight
        if r.Method == "OPTIONS" {
            // Set CORS headers
            origin := r.Header.Get("Origin")
            if isAllowedOrigin(origin) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-App-Key")
                w.Header().Set("Access-Control-Allow-Credentials", "true")
            }
            w.WriteHeader(http.StatusOK)
            return
        }

        // Security headers for ALL responses
        w.Header().Set("X-Content-Type-Options", "nosniff")           // Prevent MIME sniffing
        w.Header().Set("X-Frame-Options", "DENY")                     // Prevent clickjacking
        w.Header().Set("X-XSS-Protection", "1; mode=block")          // XSS protection
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin") // Privacy
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()") // Block sensors
        w.Header().Set("Cache-Control", "no-store, max-age=0")       // No caching

        next.ServeHTTP(w, r)
    })
}

// AppKeyMiddleware - Validates app key for API access
func AppKeyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Allow OPTIONS for CORS preflight
        if r.Method == "OPTIONS" {
            next.ServeHTTP(w, r)
            return
        }
        
        // Skip health check
        if r.URL.Path == "/health" {
            next.ServeHTTP(w, r)
            return
        }
        
        // Get app key from header
        appKey := r.Header.Get("X-App-Key")
        expectedKey := os.Getenv("APP_SECRET")
        
        // Development - allow all if no secret set
        if expectedKey == "" {
            next.ServeHTTP(w, r)
            return
        }
        
        // Production - require valid key
        if os.Getenv("ENVIRONMENT") == "production" {
            if appKey != expectedKey {
                http.Error(w, `{"error":"Unauthorized application"}`, http.StatusForbidden)
                return
            }
        } else {
            // Development - allow if matches or empty
            if appKey != "" && appKey != expectedKey {
                http.Error(w, `{"error":"Invalid app key"}`, http.StatusForbidden)
                return
            }
        }
        
        next.ServeHTTP(w, r)
    })
}

// isAllowedOrigin - Checks if origin is in allowed list
func isAllowedOrigin(origin string) bool {
    originsStr := os.Getenv("ALLOWED_ORIGINS")
    
    if originsStr == "" {
        originsStr = "http://localhost:5173,http://localhost:3000"
    }
    
    allowedOrigins := strings.Split(originsStr, ",")
    for i, allowed := range allowedOrigins {
        allowedOrigins[i] = strings.TrimSpace(allowed)
    }
    
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}

// extractToken - Extracts bearer token from Authorization header
func extractToken(r *http.Request) string {
    bearer := r.Header.Get("Authorization")
    if strings.HasPrefix(bearer, "Bearer ") {
        return strings.TrimPrefix(bearer, "Bearer ")
    }
    return ""
}