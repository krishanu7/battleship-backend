package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/krishanu7/battleship-backend/config"
	"github.com/krishanu7/battleship-backend/internal/auth"
	"github.com/krishanu7/battleship-backend/internal/game"
	"github.com/krishanu7/battleship-backend/internal/match"
	"github.com/krishanu7/battleship-backend/internal/ws"
	"github.com/krishanu7/battleship-backend/pkg/redis"
	wsPkg "github.com/krishanu7/battleship-backend/pkg/websocket"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Connect to Postgres
	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}
	defer db.Close()

	// Connect to Redis
	rdb := redis.NewRedisClient()

	// Initialize Services & Handlers
	authService := auth.NewService(db, cfg)
	authHandler := auth.NewAuthHandler(authService)

	matchService := match.NewService(rdb)
	matchChan := make(chan match.MatchResult)
	matchHandler := match.NewHandler(matchService, matchChan)

	gameService := game.NewService(rdb)
	gameHandler := game.NewHandler(gameService)

	hub := wsPkg.NewHub()
	wsHandler := ws.NewHandler(hub, gameService)

	generalHub := wsPkg.NewGeneralHub()
	generalWsHandler := ws.NewGeneralHandler(generalHub)
	
	// Start notification worker
	notificationWorker := ws.NewNotificationWorker(rdb, generalHub, gameService)
	go notificationWorker.Run()
	
	// Route Handlers
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/auth/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/api/v1/auth/login", authHandler.Login).Methods("POST")

	r.HandleFunc("/api/v1/match/join", matchHandler.JoinQueue).Methods("POST")
	r.HandleFunc("/api/v1/match/leave", matchHandler.LeaveQueue).Methods("POST")
	r.HandleFunc("/api/v1/match/start", matchHandler.StartMatch).Methods("POST")
	r.HandleFunc("/api/v1/match/cancel", matchHandler.CancelMatch).Methods("POST")
	r.HandleFunc("/api/v1/match/status", matchHandler.GetMatchStatus).Methods("GET")

	r.HandleFunc("/api/v1/game/place-ships", gameHandler.PlaceShips).Methods("POST")

	r.HandleFunc("/ws", wsHandler.ServeWS).Methods("GET")
	r.HandleFunc("/ws/general", generalWsHandler.ServeGeneralWS).Methods("GET")

	// Start Server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}