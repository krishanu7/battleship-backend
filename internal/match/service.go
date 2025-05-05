package match

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	redisClient *redis.Client
	ctx         context.Context
	queueName   string
	setName     string
}

func NewService(rdb *redis.Client) *Service {
	return &Service{
		redisClient: rdb,
		ctx:         context.Background(),
		queueName:   "matchmaking_queue",
		setName:     "queued_players",
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
	if err := s.redisClient.LPush(s.ctx, s.queueName, playerID).Err(); err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}
	if err := s.redisClient.SAdd(s.ctx, s.setName, playerID).Err(); err != nil {
		// Rollback list addition on set failure
		s.redisClient.LRem(s.ctx, s.queueName, 0, playerID)
		return fmt.Errorf("failed to add to queue set: %w", err)
	}
	return nil
}

func (s *Service) RemoveFromQueue(playerID string) error {
	// Remove from list (first occurrence) and from set
	if err := s.redisClient.LRem(s.ctx, s.queueName, 0, playerID).Err(); err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}
	if err := s.redisClient.SRem(s.ctx, s.setName, playerID).Err(); err != nil {
		return fmt.Errorf("failed to remove from set: %w", err)
	}
	return nil
}

func (s *Service) MatchPlayers() (string, string, error) {
	p1, err := s.redisClient.RPop(s.ctx, s.queueName).Result()
	if err != nil {
		return "", "", fmt.Errorf("not enough players")
	}
	p2, err := s.redisClient.RPop(s.ctx, s.queueName).Result()
	if err != nil {
		s.redisClient.LPush(s.ctx, s.queueName, p1)
		return "", "", fmt.Errorf("not enough players")
	}
	s.redisClient.SRem(s.ctx, s.setName, p1, p2)
	return p1, p2, nil
}

func (s *Service) QueueLength() (int64, error) {
	return s.redisClient.LLen(s.ctx, s.queueName).Result()
}
