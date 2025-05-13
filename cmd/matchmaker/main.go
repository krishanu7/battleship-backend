package main 

import (
	"log"
	"github.com/krishanu7/battleship-backend/internal/match"
	"github.com/krishanu7/battleship-backend/pkg/redis"
)

func main() {
	// Connect to redis
	rdb := redis.NewRedisClient()

	// Initialize match service
	matchService := match.NewService(rdb)
	//Channel to receive match results

	matchChan := make(chan match.MatchResult)

	log.Println("Matchmaker service starting...")
	go matchService.RunMatchmaker(matchChan)

	for result := range matchChan {
		log.Printf("Matched players %s and %s in room %s", result.Player1, result.Player2, result.RoomID)
		// WebSocket notification logic will be added here
	}
}