package main

import (
	"database/sql"
	"log"
	"net/http"
	"github.com/krishanu7/battleship-backend/internal/auth"
	"github.com/krishanu7/battleship-backend/config"
)

func main() {
	cfg := config.LoadConfig()
	db, err := sql.Open("postgres", cfg.DBUrl)

	if err != nil {
        log.Fatal("Failed to connect database:", err)
    }
	defer db.Close()

    authService := auth.NewService(db, cfg)
    authHandler := auth.NewAuthHandler(authService)

	http.HandleFunc("/api/v1/auth/register", authHandler.Register)
	http.HandleFunc("/api/v1/auth/login", authHandler.Login)


	log.Println("Server started at :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}