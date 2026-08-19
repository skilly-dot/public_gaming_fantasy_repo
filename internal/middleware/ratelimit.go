// internal/middleware/ratelimit.go - RATE LIMITER
package middleware

import (
    "net/http"
    "sync"
    "time"
)

type RateLimiter struct {
    mu       sync.Mutex
    visitors map[string]*visitor
}

type visitor struct {
    lastSeen time.Time
    count    int
}

func NewRateLimiter() *RateLimiter {
    rl := &RateLimiter{visitors: make(map[string]*visitor)}
    go rl.cleanup()
    return rl
}

func (rl *RateLimiter) cleanup() {
    for {
        time.Sleep(time.Minute)
        rl.mu.Lock()
        for ip, v := range rl.visitors {
            if time.Since(v.lastSeen) > time.Minute {
                delete(rl.visitors, ip)
            }
        }
        rl.mu.Unlock()
    }
}

func (rl *RateLimiter) Allow(ip string, limit int) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    v, exists := rl.visitors[ip]
    if !exists || time.Since(v.lastSeen) > time.Minute {
        rl.visitors[ip] = &visitor{lastSeen: time.Now(), count: 1}
        return true
    }
    
    v.count++
    v.lastSeen = time.Now()
    return v.count <= limit
}

func RateLimit(limit int) func(http.Handler) http.Handler {
    limiter := NewRateLimiter()
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := r.RemoteAddr
            if !limiter.Allow(ip, limit) {
                http.Error(w, `{"error":"Too many requests. Slow down."}`, http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}