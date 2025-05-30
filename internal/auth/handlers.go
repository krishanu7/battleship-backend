package auth

import (
	"encoding/json"
	"net/http"

	"github.com/krishanu7/battleship-backend/db"
)

type AuthHandler struct {
	service *Service
}

func NewAuthHandler(service *Service) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.service.Register(req.Username, req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := struct {
		Message string   `json:"message"`
		User    db.User  `json:"user"`
	}{
		Message: "User registered successfully!",
		User:    user,
	}
	resp.User.Password = ""
	json.NewEncoder(w).Encode(resp)
}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := struct {
		Token string `json:"token"`
	}{Token: token}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
