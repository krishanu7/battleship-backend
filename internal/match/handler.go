package match

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	service *Service
	matchChan chan MatchResult
}

func NewHandler(service *Service, matchChan chan MatchResult) *Handler {
	return &Handler{
		service:   service,
		matchChan: matchChan,
	}
}

func (h *Handler) JoinQueue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.service.AddToQueue(req.PlayerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Player added to queue"))
}

func (h *Handler) LeaveQueue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlayerID == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := h.service.RemoveFromQueue(req.PlayerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Player removed from queue"))
}

func (h *Handler) StartMatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlayerID == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.service.StartMatching(req.PlayerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Player added to match start queue"))
}

func (h *Handler) CancelMatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"player_id"`
	}	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlayerID == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.service.CancelMatching(req.PlayerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Player removed from match start queue"))
}

func (h *Handler) GetMatchStatus(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		http.Error(w, "missing playerId", http.StatusBadRequest)
		return
	}

	status, roomID, err := h.service.GetMatchStatus(playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"status": status,
	}
	if roomID != "" {
		resp["roomId"] = roomID
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}