package game

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	rdbPkg "github.com/krishanu7/battleship-backend/pkg/redis"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	rdb *redis.Client
}

func NewService(rdb *redis.Client) *Service {
	return &Service{
		rdb: rdb,
	}
}

// PlaceShips validates and stores a player's ship placements.
func (s *Service) PlaceShips(roomID, playerID string, ships []Ship) (*Board, error) {
	// Verify player is in room
	isMember, err := s.rdb.SIsMember(rdbPkg.Ctx, "room:"+roomID, playerID).Result()
	if err != nil || !isMember {
		return nil, fmt.Errorf("player %s not in room %s", playerID, roomID)
	}

	// Validate ship count and types
	if len(ships) != len(ShipConfig) {
		return nil, fmt.Errorf("expected %d ships, got %d", len(ShipConfig), len(ships))
	}
	shipCounts := make(map[ShipType]int)
	for _, ship := range ships {
		expectedSize, exists := ShipConfig[ship.Type]
		if !exists {
			return nil, fmt.Errorf("invalid ship type: %s", ship.Type)
		}
		if expectedSize != ship.Size {
			return nil, fmt.Errorf("invalid size for %s: expected %d, got %d", ship.Type, expectedSize, ship.Size)
		}
		shipCounts[ship.Type]++
	}
	for shipType, count := range shipCounts {
		if count != 1 {
			return nil, fmt.Errorf("exactly one %s required, got %d", shipType, count)
		}
	}

	// Validate and compute ship cells
	board := &Board{
		PlayerID: playerID,
		RoomID:   roomID,
		Ships:    ships,
		Grid:     make(map[string]string),
	}
	for i, ship := range ships {
		row, col, err := ParseCoordinate(ship.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start for %s: %v", ship.Type, err)
		}
		var cells []string
		if ship.Orientation == "horizontal" {
			if col+ship.Size > 10 {
				return nil, fmt.Errorf("%s out of bounds horizontally at %s", ship.Type, ship.Start)
			}
			for j := 0; j < ship.Size; j++ {
				cells = append(cells, FormatCoordinate(row, col+j))
			}
		} else if ship.Orientation == "vertical" {
			if row+ship.Size > 10 {
				return nil, fmt.Errorf("%s out of bounds vertically at %s", ship.Type, ship.Start)
			}
			for j := 0; j < ship.Size; j++ {
				cells = append(cells, FormatCoordinate(row+j, col))
			}
		} else {
			return nil, fmt.Errorf("invalid orientation for %s: %s", ship.Type, ship.Orientation)
		}

		// Check for overlaps
		for _, cell := range cells {
			if _, exists := board.Grid[cell]; exists {
				return nil, fmt.Errorf("overlap at %s for %s", cell, ship.Type)
			}
			board.Grid[cell] = string(ship.Type)
		}
		ships[i].Cells = cells
	}

	boardJSON, err := json.Marshal(board)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board: %v", err)
	}
	key := fmt.Sprintf("room:%s:board:%s", roomID, playerID)
	if err := s.rdb.Set(rdbPkg.Ctx, key, boardJSON, 24*time.Hour).Err(); err != nil {
		return nil, fmt.Errorf("failed to store board: %v", err)
	}
	log.Printf("Stored board for player %s in room %s", playerID, roomID)

	// Publish ships_placed notification
	notification := struct {
		Type   string `json:"type"`
		RoomID string `json:"roomId"`
		Player string `json:"player"`
	}{
		Type:   "ships_placed",
		RoomID: roomID,
		Player: playerID,
	}
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Failed to marshal ships_placed notification for %s: %v", playerID, err)
	} else {
		if err := s.rdb.Publish(rdbPkg.Ctx, "notifications", notificationBytes).Err(); err != nil {
			log.Printf("Failed to publish ships_placed notification for %s: %v", playerID, err)
		} else {
			log.Printf("Published ships_placed notification for player %s in room %s", playerID, roomID)
		}
	}
	// Check if opponent has placed ships
	players, err := s.rdb.SMembers(rdbPkg.Ctx, "room:"+roomID).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %v", err)
	}
	opponentID := ""
	for _, p := range players {
		if p != playerID {
			opponentID = p
			break
		}
	}
	if opponentID != "" {
		opponentKey := fmt.Sprintf("room:%s:board:%s", roomID, opponentID)
		exists, err := s.rdb.Exists(rdbPkg.Ctx, opponentKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check opponent board: %v", err)
		}
		if exists == 1 {
			log.Printf("Both players in room %s have placed ships", roomID)
			//[TODO] Game start notification handled by NotificationWorker
		}
	}

	return board, nil
}