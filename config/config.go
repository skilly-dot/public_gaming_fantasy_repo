// config/config.go

package config

import (
	"os"
	"strings"
)

type Config struct {
    Port           string
    PostgresDSN    string
    RedisAddr      string
    MongoURI       string
    AppSecret      string
    AllowedOrigins []string
    Environment    string
}

func Load() *Config {
    return &Config{
        Port:        getEnv("PORT", "8080"),
        PostgresDSN: getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/betking_rich?sslmode=disable"),
        RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
        MongoURI:    getEnv("MONGO_URI", ""),
        AppSecret:   getEnv("APP_SECRET", "blingbling"),
        AllowedOrigins: getAllowedOrigins(),
        Environment: getEnv("ENVIRONMENT", "development"),
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

func getAllowedOrigins() []string {
    originsStr := getEnv("ALLOWED_ORIGINS","https://fantasy-games-frontend.onrender.com")
    
    if originsStr == "" {
        originsStr = "http://localhost:5173,http://localhost:3000"
    }
    
    origins := strings.Split(originsStr, ",")
    for i, origin := range origins {
        origins[i] = strings.TrimSpace(origin)
    }
    
    return origins
}