package game

import (
	"encoding/json"
	"net/http"
	"log"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

type PlaceShipsRequest struct {
	PlayerID string `json:"player_id"`
	RoomID   string `json:"room_id"`
	Ships    []Ship `json:"ships"`
}

type PlaceShipsResponse struct {
	Message string `json:"message"`
}

func (h *Handler) PlaceShips(w http.ResponseWriter, r *http.Request) {
	var req PlaceShipsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.PlayerID == "" || req.RoomID == "" || len(req.Ships) == 0 {
		http.Error(w, "Missing player_id, room_id or ships", http.StatusBadRequest)
		return
	}

	_, err := h.service.PlaceShips(req.RoomID, req.PlayerID, req.Ships)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Failed to place ships for %s: %v", req.PlayerID, err)
		return
	}

	resp := PlaceShipsResponse{Message: "Ships placed successfully"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}