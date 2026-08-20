// cmd/api/routes.go - COMPLETE

package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/betking/rich-backend/internal/handlers"
	mw "github.com/betking/rich-backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func setupRoutes(r chi.Router, h *handlers.Handler) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// ===== CORS FIRST - Before SecurityHeaders and AppKey =====
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   getAllowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-App-Key", "If-None-Match"},
		ExposedHeaders:   []string{"ETag"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// THEN SecurityHeaders (must allow OPTIONS pass-through)
	r.Use(mw.SecurityHeaders)

	// THEN AppKeyMiddleware (must allow OPTIONS pass-through)
	r.Use(mw.AppKeyMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/ws", h.WebSocketHandler)

	r.Route("/api/v1", func(r chi.Router) {
		// ===== PUBLIC ROUTES =====
		r.Group(func(r chi.Router) {
			r.Use(mw.RateLimit(5))
			r.Post("/auth/register", h.Register)
			r.Post("/auth/login", h.Login)
			r.Post("/auth/logout", h.Logout)
		})

		// ===== PROTECTED ROUTES =====
		r.Group(func(r chi.Router) {
			r.Use(mw.AuthMiddleware(h.DB, h.Redis))

			// ===== STATE ENDPOINT (NEW) =====
			r.Get("/state", h.GetState)

			// ===== ADMIN ROUTES =====
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireAdmin)

				// Users
				r.Get("/admin/users", h.AdminGetUsers)
				r.Get("/admin/users/{userID}", h.AdminGetUser)
				r.Put("/admin/users/{userID}", h.AdminUpdateUser)
				r.Put("/admin/users/{userID}/full", h.AdminUpdateUserFull)
				r.Delete("/admin/users/{userID}", h.AdminDeleteUser)

				// Leagues
				r.Get("/admin/leagues", h.AdminGetLeagues)
				r.Put("/admin/leagues/{leagueID}", h.AdminUpdateLeague)
				r.Delete("/admin/leagues/{leagueID}", h.AdminDeleteLeague)

				// Wallets
				r.Put("/admin/wallets/{userID}", h.AdminUpdateWallet)
				r.Post("/admin/wallets/{userID}/add", h.AdminAddToWallet)

				// Admin Match Bets
				r.Post("/admin/match-bets/create", h.AdminCreateMatchBet)
				r.Delete("/admin/match-bets/{betID}", h.AdminDeleteMatchBet)
				r.Put("/admin/match-bets/{betID}/lock", h.AdminLockMatchBet)
				r.Get("/admin/match-bets", h.AdminGetMatchBets)
				r.Post("/admin/matches/{matchID}/settle", h.AdminSettleMatch)
				r.Get("/admin/match-bets/{betID}/matches", h.AdminGetBetMatches)

				// 50-50 Bets
				r.Post("/admin/fifty-fifty/create", h.AdminCreateFiftyFifty)
				r.Put("/admin/fifty-fifty/{betID}/lock", h.AdminLockFiftyFifty)
				r.Put("/admin/fifty-fifty/{betID}/settle", h.AdminSettleFiftyFifty)
				r.Delete("/admin/fifty-fifty/{betID}", h.AdminDeleteFiftyFifty)
				r.Get("/admin/fifty-fifty", h.AdminGetFiftyFifty)

				// Cleanup
				r.Post("/admin/cleanup", h.CleanupOldData)
			})

			// ===== LEAGUES =====
			r.Post("/leagues/create", h.CreateLeague)
			r.Get("/leagues/my", h.GetMyLeagues)
			r.Get("/leagues/{leagueID}", h.GetFullLeague)
			r.Get("/leagues/{leagueID}/state", h.GetLeagueState)
			r.Get("/leagues/{leagueID}/daily", h.GetDailyData)
			r.Get("/leagues/{leagueID}/table", h.GetLeagueTable)
			r.Get("/leagues/{leagueID}/results", h.GetMatchResults)
			r.Get("/leagues/{leagueID}/scorers", h.GetTopScorers)
			r.Get("/leagues/{leagueID}/probabilities", h.GetMatchProbabilities)
			r.Get("/leagues/{leagueID}/teams", h.GetLeagueTeams)
			r.Get("/leagues/{leagueID}/players", h.GetLeaguePlayers)
			r.Get("/leagues/{leagueID}/fixtures", h.GetLeagueFixtures)
			r.Post("/leagues/{leagueID}/start-week", h.StartWeek)
			r.Post("/leagues/{leagueID}/next-week", h.NextWeek)
			r.Post("/leagues/{leagueID}/winner-bet", h.PlaceLeagueWinnerBet)
			r.Post("/leagues/{leagueID}/finish", h.FinishLeague)
			r.Post("/leagues/{leagueID}/forfeit", h.ForfeitLeague)

			r.Post("/leagues/{leagueID}/quick-match/generate", h.GenerateQuickMatch)
			r.Post("/leagues/{leagueID}/quick-match/{quickMatchID}/bet", h.BetOnQuickMatch)
			r.Post("/leagues/{leagueID}/quick-match/{quickMatchID}/start", h.StartQuickMatch)

			// ===== BETTING =====
			r.Group(func(r chi.Router) {
				r.Use(mw.RateLimit(30))
				r.Post("/bets/place", h.PlaceBet)
			})
			r.Get("/bets/active", h.GetActiveBets)
			r.Get("/bets/history", h.GetBetHistory)

			// Admin Match Betting (User endpoints)
			r.Get("/bets/admin-matches", h.GetAdminMatchesForBetting)
			r.Post("/bets/admin-place", h.PlaceAdminMatchBet)
			r.Get("/bets/admin-active", h.GetActiveAdminBets)
			r.Get("/bets/admin-history", h.GetAdminBetHistory)

			// 50-50 Betting (User endpoints)
			r.Get("/bets/fifty-fifty-available", h.GetFiftyFiftyForBetting)
			r.Post("/bets/fifty-fifty-place", h.PlaceFiftyFiftyBet)
			r.Get("/bets/fifty-fifty-active", h.GetActiveFiftyFiftyBets)
			r.Get("/bets/fifty-fifty-history", h.GetFiftyFiftyHistory)

			// ===== WALLET =====
			r.Get("/wallet", h.GetWallet)
			r.Post("/wallet/sync", h.SyncWallet)
			r.Post("/wallet/daily-bonus", h.DailyBonus)
			r.Post("/wallet/exchange", h.ExchangeCurrency)

			// ===== RANKINGS =====
			r.Get("/rankings", h.GetRankings)

			// ===== NOTIFICATIONS =====
			r.Get("/notifications", h.GetNotifications)
			r.Post("/notifications/{id}/read", h.MarkNotificationRead)
			r.Post("/notifications/read-all", h.MarkAllNotificationsRead)

			// ===== ACCOUNT =====
			r.Delete("/account", h.DeleteAccount)
		})
	})
}

// getAllowedOrigins - Reads from env with dev fallback
func getAllowedOrigins() []string {
	originsStr := os.Getenv("ALLOWED_ORIGINS")
	
	if originsStr == "" {
		return []string{
			"http://localhost:5173",
			"http://localhost:3000",
		}
	}
	
	origins := strings.Split(originsStr, ",")
	for i, origin := range origins {
		origins[i] = strings.TrimSpace(origin)
	}
	
	return origins
}