package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// In state_endpoint.go - Update GetState

func (h *Handler) GetState(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	leagueID := r.URL.Query().Get("league_id")
	if leagueID == "" {
		// Auto-detect user's active league
		h.DB.Get(&leagueID, "SELECT id FROM leagues WHERE user_id=$1 AND status='ACTIVE' LIMIT 1", user.ID)
	}
	
	// Build full state
	state := h.BuildFullState(user.ID, leagueID)
	
	// Generate hash for change detection
	stateJSON, _ := json.Marshal(state)
	hash := fmt.Sprintf("%x", sha256.Sum256(stateJSON))
	
	// Check if client has same hash
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch == hash {
		w.Header().Set("ETag", hash)
		w.WriteHeader(http.StatusNotModified)
		return
	}
	
	w.Header().Set("ETag", hash)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"hash":      hash,
		"timestamp": time.Now(),
		"state":     state,
	})
}