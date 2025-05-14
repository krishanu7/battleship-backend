package main

import (
	"encoding/json"
	"log"

	"github.com/krishanu7/battleship-backend/internal/match"
	"github.com/krishanu7/battleship-backend/pkg/redis"
)

func main() {
	// Connect to Redis
	rdb := redis.NewRedisClient()

	// Initialize match service
	matchService := match.NewService(rdb)

	// Channel to receive match results
	matchChan := make(chan match.MatchResult)

	// Start matchmaker
	log.Println("Matchmaker service starting...")
	go matchService.RunMatchmaker(matchChan)

	// Handle match results and publish to Redis
	for result := range matchChan {
		log.Printf("Matched players %s and %s in room %s", result.Player1, result.Player2, result.RoomID)

		// Create notification
		notification := struct {
			Type   string `json:"type"`
			RoomID string `json:"roomId"`
			Player string `json:"player"`
		}{
			Type:   "match_found",
			RoomID: result.RoomID,
		}

		// Notify player 1
		notification.Player = result.Player1
		notificationBytes, err := json.Marshal(notification)
		if err != nil {
			log.Printf("Failed to marshal notification for %s: %v", result.Player1, err)
			continue
		}
		if err := rdb.Publish(redis.Ctx, "notifications", notificationBytes).Err(); err != nil {
			log.Printf("Failed to publish notification for %s: %v", result.Player1, err)
		}

		// Notify player 2
		notification.Player = result.Player2
		notificationBytes, err = json.Marshal(notification)
		if err != nil {
			log.Printf("Failed to marshal notification for %s: %v", result.Player2, err)
			continue
		}
		if err := rdb.Publish(redis.Ctx, "notifications", notificationBytes).Err(); err != nil {
			log.Printf("Failed to publish notification for %s: %v", result.Player2, err)
		}
	}
}