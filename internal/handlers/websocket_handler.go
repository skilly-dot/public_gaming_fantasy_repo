// internal/handlers/websocket_handler.go - WITH INITIAL STATE PUSH

package handlers

import (
	"net/http"
	"strings"
	"time"

	ws "github.com/betking/rich-backend/internal/websocket"
	"github.com/gorilla/websocket"
)

func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate FIRST before upgrading
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Verify token
	var userData struct {
		UserID   string
		Username string
	}
	err := h.DB.QueryRow(`
		SELECT u.id, u.username 
		FROM sessions s 
		JOIN users u ON s.user_id = u.id 
		WHERE s.token = $1 
		AND (s.expires_at IS NULL OR s.expires_at > NOW())
	`, token).Scan(&userData.UserID, &userData.Username)
	
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}
	
	// Secure origin check
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return strings.Contains(origin, "localhost") || 
			       strings.Contains(origin, "onrender.com") ||
			       strings.Contains(origin, "yourdomain.com")
		},
	}
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	
	client := &ws.Client{
		Hub:    h.WSHub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userData.UserID,
	}
	
	// Get user's active league
	var leagueID string
	h.DB.QueryRow("SELECT id FROM leagues WHERE user_id=$1 AND status='ACTIVE' LIMIT 1", 
		userData.UserID).Scan(&leagueID)
	client.LeagueID = leagueID
	
	h.WSHub.Register <- client
	
	// ===== SEND INITIAL FULL STATE ON CONNECTION =====
	go func() {
		// Small delay to ensure client is ready
		time.Sleep(100 * time.Millisecond)
		
		// Build and send full state
		fullState := h.BuildFullState(userData.UserID, leagueID)
		h.WSHub.SendToUser(userData.UserID, "INITIAL_STATE", map[string]interface{}{
			"state": fullState,
			"timestamp": time.Now(),
		})
	}()
	
	go client.WritePump()
	go client.ReadPump()
}