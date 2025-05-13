package main

import (
	"encoding/json"
	"log"

	"github.com/krishanu7/battleship-backend/internal/match"
	"github.com/krishanu7/battleship-backend/pkg/redis"
	wsPkg "github.com/krishanu7/battleship-backend/pkg/websocket"
)

func main() {
	// Connect to redis
	rdb := redis.NewRedisClient()

	// Initialize match service
	matchService := match.NewService(rdb)

	//Initialize general hub for notifications
	generalHub := wsPkg.NewGeneralHub()

	//Channel to receive match results
	matchChan := make(chan match.MatchResult)

	log.Println("Matchmaker service starting...")
	go matchService.RunMatchmaker(matchChan)

	for result := range matchChan {
		log.Printf("Matched players %s and %s in room %s", result.Player1, result.Player2, result.RoomID)
		// WebSocket notification logic will be added here
		notification := struct {
			Type   string `json:"type"`
			RoomID string `json:"roomId"`
		}{
			Type:   "match_found",
			RoomID: result.RoomID,
		}
		notificationBytes, err := json.Marshal(notification)

		if err != nil {
			log.Printf("Failed to marshal notification: %v", err)
			continue
		}
		//Notify player 1
		if !generalHub.SendToClient(result.Player1, notificationBytes) {
			log.Printf("Failed to notify player %s", result.Player1)
		}
		//Notify player 2
		if !generalHub.SendToClient(result.Player2, notificationBytes) {
			log.Printf("Failded to notify player %s", result.Player2)
		}
	}
}
