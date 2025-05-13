package match

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

type Service struct {
	redisClient *redis.Client
	ctx         context.Context
	mainQueue   string // list of player in queue
	startQueue  string // player who pressed start button
	setName     string // set of players in queue
	channel     string // channel for pub/sub
}

type MatchResult struct {
	Player1 string
	Player2 string
	RoomID  string
}

func NewService(rdb *redis.Client) *Service {
	return &Service{
		redisClient: rdb,
		ctx:         context.Background(),
		mainQueue:   "matchmaking_queue",
		startQueue:  "match_start_queue",
		setName:     "queued_players",
		channel:     "matchmaking_channel",
	}
}

func (s *Service) AddToQueue(playerID string) error {
	// Check if player is already in the set
	exists, err := s.redisClient.SIsMember(s.ctx, s.setName, playerID).Result()
	if err != nil {
		return fmt.Errorf("failed to check queue set: %w", err)
	}
	if exists {
		return fmt.Errorf("player already in queue")
	}

	// Add to list and set
	if err := s.redisClient.LPush(s.ctx, s.mainQueue, playerID).Err(); err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}
	if err := s.redisClient.SAdd(s.ctx, s.setName, playerID).Err(); err != nil {
		// Rollback list addition on set failure
		s.redisClient.LRem(s.ctx, s.mainQueue, 0, playerID)
		return fmt.Errorf("failed to add to queue set: %w", err)
	}
	return nil
}

func (s *Service) RemoveFromQueue(playerID string) error {
	// Remove from list (first occurrence) and from set
	if err := s.redisClient.LRem(s.ctx, s.mainQueue, 0, playerID).Err(); err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}
	if err := s.redisClient.SRem(s.ctx, s.setName, playerID).Err(); err != nil {
		return fmt.Errorf("failed to remove from set: %w", err)
	}
	return nil
}

func (s* Service) StartMatching(playerID string) error {
	// check if player is in the matching_queue
	exists, err := s.redisClient.SIsMember(s.ctx, s.setName, playerID).Result()
	if err != nil || !exists {
		return fmt.Errorf("player not in queue")
	}
	// Remove from matchmaking_queue 
	if err := s.RemoveFromQueue(playerID); err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}
	// Add to match_start_queue
	if err := s.redisClient.LPush(s.ctx, s.startQueue, playerID).Err(); err != nil {
		return fmt.Errorf("failed to add to start queue: %w", err)
	}
	// Publish to matchmaking channel
	if err := s.redisClient.Publish(s.ctx, s.channel, playerID).Err(); err != nil {
		s.redisClient.LRem(s.ctx, s.startQueue, 0, playerID)
		return fmt.Errorf("failed to publish to channel: %w", err)
	}
	return nil
}
// TODO: Think about how to handle this
func (s *Service) CancelMatching(playerID string) error {
	if err := s.redisClient.LRem(s.ctx, s.startQueue, 0, playerID).Err(); err != nil {

		return fmt.Errorf("failed to remove from start queue: %w", err)
	}
	return nil
}

func (s *Service) MatchPlayers() (string, string, string, error) {
	p1, err := s.redisClient.RPop(s.ctx, s.startQueue).Result()
	if err != nil {
		return "", "", "", fmt.Errorf("not enough players")
	}
	p2, err := s.redisClient.RPop(s.ctx, s.startQueue).Result()
	if err != nil {
		s.redisClient.LPush(s.ctx, s.startQueue, p1)
		return "", "", "", fmt.Errorf("not enough players")
	}

	roomID := generateRoomID(p1, p2)

	// Store room-player mapping in Redis
	roomKey := fmt.Sprintf("room:%s", roomID)
	if err := s.redisClient.SAdd(s.ctx, roomKey, p1, p2).Err(); err != nil {
		s.redisClient.LPush(s.ctx, s.startQueue, p1, p2)
		return "", "", "", fmt.Errorf("failed to store room mapping: %w", err)
	}
	// Set expiration for room mapping (e.g., 1 hour)
	s.redisClient.Expire(s.ctx, roomKey, 1*time.Hour)

	return p1, p2, roomID, nil
}

func (s *Service) RunMatchmaker(matchChan chan MatchResult) {
	pubsub := s.redisClient.Subscribe(s.ctx, s.channel)
	defer pubsub.Close()

	for {
		_, err := pubsub.ReceiveMessage(s.ctx)
		if err != nil {
			log.Printf("Pub/sub error: %v", err)
			continue
		}

		// Check if there are enough players
		length, err := s.redisClient.LLen(s.ctx, s.startQueue).Result()
		if err != nil || length < 2 {
			continue
		}

		// Attempt to match players
		p1, p2, roomID, err := s.MatchPlayers()
		if err != nil {
			continue
		}

		matchChan <- MatchResult{
			Player1: p1,
			Player2: p2,
			RoomID:  roomID,
		}
	}
}

func (s *Service) GetMatchStatus(playerID string) (string, string, error) {
	// Check if player is in match_start_queue
	length, err := s.redisClient.LLen(s.ctx, s.startQueue).Result()
	if err != nil {
		return "", "", fmt.Errorf("failed to check start queue: %w", err)
	}
	for i := int64(0); i < length; i++ {
		player, err := s.redisClient.LIndex(s.ctx, s.startQueue, i).Result()
		if err == nil && player == playerID {
			return "waiting", "", nil
		}
	}

	// Check if player is in a room
	keys, err := s.redisClient.Keys(s.ctx, "room:*").Result()
	if err != nil {
		return "", "", fmt.Errorf("failed to scan rooms: %w", err)
	}
	for _, key := range keys {
		exists, err := s.redisClient.SIsMember(s.ctx, key, playerID).Result()
		if err == nil && exists {
			roomID := key[len("room:"):]
			return "matched", roomID, nil
		}
	}

	// Check if player is in matchmaking_queue
	exists, err := s.redisClient.SIsMember(s.ctx, s.setName, playerID).Result()
	if err == nil && exists {
		return "in_queue", "", nil
	}

	return "not_found", "", nil
}

func (s *Service) QueueLength() (int64, error) {
	return s.redisClient.LLen(s.ctx, s.mainQueue).Result()
}

func generateRoomID(player1, player2 string) string {
	hash := sha1.Sum([]byte(player1 + ":" + player2))
	return hex.EncodeToString(hash[:])
}
