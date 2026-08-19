package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/betking/rich-backend/config"
	"github.com/betking/rich-backend/internal/database"
	"github.com/betking/rich-backend/internal/handlers"
)

func main() {
	cfg := config.Load()

	
	pg, err := database.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("Postgres connection failed:", err)
	}
	defer pg.Close()
	
	log.Println("Running migrations...")
	pg.RunMigrations()
	pg.AddPerformanceIndexes()
	pg.AddNewIndexes()
	log.Println("Migrations completed")

	redis := database.NewRedis(cfg.RedisAddr)
    defer redis.Close()

	mongo, _ := database.NewMongo(cfg.MongoURI)
	if mongo != nil {
		defer mongo.Close()
	}

	
	h := handlers.NewHandler(pg, redis, mongo, cfg)

	go h.WSHub.Run()
	log.Println("WebSocket hub started")

	
	r := chi.NewRouter()
	setupRoutes(r, h)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ===== START SERVER =====
	go func() {
		fmt.Printf(" BetKing Backend running on :%s\n", cfg.Port)
		fmt.Printf(" WebSocket endpoint: ws://localhost:%s/ws\n", cfg.Port)
		fmt.Printf(" Health check: http://localhost:%s/health\n", cfg.Port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed:", err)
		}
	}()

	// ===== GRACEFUL SHUTDOWN =====
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server stopped gracefully")
}