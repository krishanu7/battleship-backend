package game

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	rdbPkg "github.com/krishanu7/battleship-backend/pkg/redis"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	Rdb *redis.Client
	db *sql.DB
}

type GameOver struct {
	Winner string `json:"winner"`
	Loser  string `json:"loser"`
}

func NewService(rdb *redis.Client, db *sql.DB) *Service {
	return &Service{
		Rdb: rdb,
		db: db,
	}
}

// Initialize game state after both players placed ships
func (s *Service) InitializeGame(roomId string) error {
	players, err := s.Rdb.SMembers(rdbPkg.Ctx, "room:"+roomId).Result()

	if err != nil {
		return fmt.Errorf("failed to retrive room members: %v", err)
	}
	if len(players) != 2 {
		return fmt.Errorf("room %s has %d players, but expected 2", roomId, len(players))
	}
	// Randomly choose the first turn
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	turn := players[rng.Intn(2)]

	gameState := GameState{
		RoomID:    roomId,
		Turn:      turn,
		StartedAt: time.Now().Unix(),
	}
	gameJSON, err := json.Marshal(gameState)

	if err != nil {
		return fmt.Errorf("failed to marshal game state: %v", err)
	}
	if err := s.Rdb.Set(rdbPkg.Ctx, "room:"+roomId+":game", gameJSON, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store game state: %v", err)
	}
	log.Printf("Initialized game for room %s with first turn: %s", roomId, turn)
	return nil
}

// handles a player's attack and returns the result
func (s *Service) ProcessAttack(roomID, playerID, coordinate string) (*Attack, []string, *GameOver, error) {
	// check if the room exists and have players
	isMember, err := s.Rdb.SIsMember(rdbPkg.Ctx, "room:"+roomID, playerID).Result()
	if err != nil || !isMember {
		return nil, nil, nil, fmt.Errorf("player %s not in room %s", playerID, roomID)
	}
	// check if valid coordinate
	_, _, err = ParseCoordinate(coordinate)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid coordinate: %v", err)
	}
	// Check if it's the player's turn
	gameJSON, err := s.Rdb.Get(rdbPkg.Ctx, "room:"+roomID+":game").Result()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get game state: %v", err)
	}
	var gameState GameState
	if err := json.Unmarshal([]byte(gameJSON), &gameState); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal game state: %v", err)
	}
	if gameState.Turn != playerID {
		return nil, nil, nil, fmt.Errorf("not your turn")
	}

	//Check if coordinate was already attacked
	attackKey := fmt.Sprintf("room:%s:attacks:%s", roomID, playerID)
	alreadyAttacked, err := s.Rdb.SIsMember(rdbPkg.Ctx, attackKey, coordinate).Result()

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to check attacks: %v", err)
	}
	if alreadyAttacked {
		return nil, nil, nil, fmt.Errorf("coordinate %s already attacked", coordinate)
	}
	// Get opponent's player ID
	players, err := s.Rdb.SMembers(rdbPkg.Ctx, "room:"+roomID).Result()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get room members: %v", err)
	}

	var opponentID string
	for _, p := range players {
		if p != playerID {
			opponentID = p
			break
		}
	}
	// Load opponet's board
	boardKey := fmt.Sprintf("room:%s:board:%s", roomID, opponentID)
	boardJSON, err := s.Rdb.Get(rdbPkg.Ctx, boardKey).Result()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get opponent board: %v", err)
	}
	var opponentBoard Board
	if err := json.Unmarshal([]byte(boardJSON), &opponentBoard); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal opponent board: %v", err)
	}

	// Check for hit or miss
	result := "miss"
	var nextTurn string
	if _, exists := opponentBoard.Grid[coordinate]; exists {
		result = "hit"
		nextTurn = playerID // Keep turn on hit
	} else {
		nextTurn = opponentID // Switch turn on miss
	}

	// Record the attack
	if err := s.Rdb.SAdd(rdbPkg.Ctx, attackKey, coordinate).Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to record attack: %v", err)
	}

	log.Printf("Player %s attacked %s in room %s: %s, next turn: %s", playerID, coordinate, roomID, result, nextTurn)

	// Check for sunk ships
	sunkShips := []string{}
	if result == "hit" {
		attacks, err := s.Rdb.SMembers(rdbPkg.Ctx, attackKey).Result()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get attacks: %v", err)
		}
		for _, ship := range opponentBoard.Ships {
			isSunk := true
			for _, cell := range ship.Cells {
				found := false
				for _, attack := range attacks {
					if attack == cell {
						found = true
						break
					}
				}
				if !found {
					isSunk = false
					break
				}
			}
			if isSunk {
				sunkShips = append(sunkShips, string(ship.Type))
			}
		}
	}

	// Check for victory (17 hits)
	var gameOver *GameOver
	if result == "hit" {
		attacks, err := s.Rdb.SMembers(rdbPkg.Ctx, attackKey).Result()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get attacks: %v", err)
		}
		hitCount := 0
		for _, attack := range attacks {
			if _, exists := opponentBoard.Grid[attack]; exists {
				hitCount++
			}
		}
		if hitCount >= 17 {
			gameOver = &GameOver{
				Winner: playerID,
				Loser:  opponentID,
			}
			// Update stats
			if err := s.updatePlayerStats(playerID, opponentID); err != nil {
				log.Printf("Failed to update player stats: %v", err)
			}
			// Clean up Redis
			keys, err := s.Rdb.Keys(rdbPkg.Ctx, "room:"+roomID+":*").Result()
			if err != nil {
				log.Printf("Failed to get room keys: %v", err)
			} else {
				for _, key := range keys {
					s.Rdb.Del(rdbPkg.Ctx, key)
				}
				log.Printf("Cleared Redis keys for room %s", roomID)
			}
		}
	}

	// Update turn if no game over
	if gameOver == nil {
		gameState.Turn = nextTurn
		updatedGameJSON, err := json.Marshal(gameState)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to marshal updated game state: %v", err)
		}
		if err := s.Rdb.Set(rdbPkg.Ctx, "room:"+roomID+":game", updatedGameJSON, 24*time.Hour).Err(); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update game state: %v", err)
		}
	}

	return &Attack{Coordinate: coordinate, Result: result}, sunkShips, gameOver, nil
}

// validate and store a player's ship placements
func (s *Service) PlaceShips(roomID, playerID string, ships []Ship) (*Board, error) {
	// Verify player is in room
	isMember, err := s.Rdb.SIsMember(rdbPkg.Ctx, "room:"+roomID, playerID).Result()
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
	if err := s.Rdb.Set(rdbPkg.Ctx, key, boardJSON, 24*time.Hour).Err(); err != nil {
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
		if err := s.Rdb.Publish(rdbPkg.Ctx, "notifications", notificationBytes).Err(); err != nil {
			log.Printf("Failed to publish ships_placed notification for %s: %v", playerID, err)
		} else {
			log.Printf("Published ships_placed notification for player %s in room %s", playerID, roomID)
		}
	}
	// Check if opponent has placed ships
	players, err := s.Rdb.SMembers(rdbPkg.Ctx, "room:"+roomID).Result()
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
		exists, err := s.Rdb.Exists(rdbPkg.Ctx, opponentKey).Result()
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

func (s *Service) updatePlayerStats(winnerID, loserID string) error {
	var winnerStats, loserStats PlayerStats
	err := s.db.QueryRow("SELECT player_id, wins, losses, elo FROM stats WHERE player_id = $1", winnerID).
		Scan(&winnerStats.PlayerID, &winnerStats.Wins, &winnerStats.Losses, &winnerStats.Elo)
	if err == sql.ErrNoRows {
		winnerStats = PlayerStats{PlayerID: winnerID, Wins: 0, Losses: 0, Elo: 1500}
	} else if err != nil {
		return fmt.Errorf("failed to get winner stats: %v", err)
	}
	err = s.db.QueryRow("SELECT player_id, wins, losses, elo FROM stats WHERE player_id = $1", loserID).
		Scan(&loserStats.PlayerID, &loserStats.Wins, &loserStats.Losses, &loserStats.Elo)
	if err == sql.ErrNoRows {
		loserStats = PlayerStats{PlayerID: loserID, Wins: 0, Losses: 0, Elo: 1500}
	} else if err != nil {
		return fmt.Errorf("failed to get loser stats: %v", err)
	}

	//  ELO update
	const K = 32
	expectedWinner := 1 / (1 + math.Pow(10, float64(loserStats.Elo-winnerStats.Elo)/400))
	expectedLoser := 1 / (1 + math.Pow(10, float64(winnerStats.Elo-loserStats.Elo)/400))
	newWinnerElo := winnerStats.Elo + int(float64(K)*(1-expectedWinner))
	newLoserElo := loserStats.Elo + int(float64(K)*(0-expectedLoser))

	// Update stats
	_, err = s.db.Exec(
		"INSERT INTO stats (player_id, wins, losses, elo) VALUES ($1, $2, $3, $4) ON CONFLICT (player_id) DO UPDATE SET wins = $2, losses = $3, elo = $4",
		winnerID, winnerStats.Wins+1, winnerStats.Losses, newWinnerElo,
	)
	if err != nil {
		return fmt.Errorf("failed to update winner stats: %v", err)
	}
	_, err = s.db.Exec(
		"INSERT INTO stats (player_id, wins, losses, elo) VALUES ($1, $2, $3, $4) ON CONFLICT (player_id) DO UPDATE SET wins = $2, losses = $3, elo = $4",
		loserID, loserStats.Wins, loserStats.Losses+1, newLoserElo,
	)
	if err != nil {
		return fmt.Errorf("failed to update loser stats: %v", err)
	}
	log.Printf("Updated stats: %s (wins=%d, elo=%d), %s (losses=%d, elo=%d)",
		winnerID, winnerStats.Wins+1, newWinnerElo, loserID, loserStats.Losses+1, newLoserElo)
	return nil
}