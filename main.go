package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/krishanu7/battleship-backend/config"
	"github.com/krishanu7/battleship-backend/internal/auth"
	"github.com/krishanu7/battleship-backend/internal/match"
	"github.com/krishanu7/battleship-backend/pkg/redis"
)

func main() {
	cfg := config.LoadConfig()
	// 1. Connect to Postgres
	db, err := sql.Open("postgres", cfg.DBUrl)

	if err != nil {
        log.Fatal("Failed to connect database:", err)
    }
	defer db.Close()
	// 2. Connect to Redis
	rdb := redis.NewRedisClient()

	// 3. Initialize Services & Handlers
    authService := auth.NewService(db, cfg)
    authHandler := auth.NewAuthHandler(authService)

	matchService := match.NewService(rdb)
	matchHandler := match.NewHandler(matchService)

	// 4. Route Handlers
	http.HandleFunc("/api/v1/auth/register", authHandler.Register)
	http.HandleFunc("/api/v1/auth/login", authHandler.Login)
	

	http.HandleFunc("/api/v1/match/join", matchHandler.JoinQueue)
	http.HandleFunc("/api/v1/match/start", matchHandler.StartMatch)
	http.HandleFunc("/api/v1/match/leave", matchHandler.LeaveQueue)

	log.Println("Server started at :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}