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
}

func NewService(rdb *redis.Client) *Service {
	return &Service{
		redisClient: rdb,
		ctx:         context.Background(),
		queueName:   "matchmaking_queue",
	}
}

func (s *Service) AddToQueue(playerID string) error {
	err := s.redisClient.LPush(s.ctx, s.queueName, playerID).Err()
	if err != nil {
		return fmt.Errorf("failed to add player to queue: %w", err)
	}
	return nil
}

func (s *Service) MatchPlayers() (string, string, error) {
	player1, err := s.redisClient.RPop(s.ctx, s.queueName).Result()

	if err != nil {
		return "", "", fmt.Errorf("failed to pop player from queue: %w", err)
	}
	player2, err := s.redisClient.RPop(s.ctx, s.queueName).Result()

	if err != nil {
		// Put player1 back to the queue
		s.redisClient.LPush(s.ctx, s.queueName, player1)
		return "", "", fmt.Errorf("not enough players to match")
	}
	return player1, player2, nil
}

func (s *Service) QueueLength() (int64, error) {
	length, err := s.redisClient.LLen(s.ctx, s.queueName).Result()
	if err != nil {
		return 0, err
	}
	return length, nil
}
