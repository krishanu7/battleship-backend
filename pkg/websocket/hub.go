package websocket

import (
	"log"
	"sync"

	redisM "github.com/redis/go-redis/v9"
	"github.com/krishanu7/battleship-backend/pkg/redis"
)


type Hub struct {
	Rooms map[string]*Room
	mu    sync.Mutex
	rdb   *redisM.Client
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[string]*Room),
		rdb:   redis.NewRedisClient(),
	}
}

func (h *Hub) GetRoom(roomID string) (*Room, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check in-memory rooms first
	if room, exists := h.Rooms[roomID]; exists {
		return room, true
	}

	players, err := h.rdb.SMembers(redis.Ctx, "room:"+roomID).Result()
	if err != nil {
		log.Printf("Failed to check room %s in Redis: %v", roomID, err)
		return nil, false
	}
	if len(players) == 0 {
		log.Printf("Room %s not found in Redis", roomID)
		return nil, false
	}

	room := NewRoom(roomID)
	h.Rooms[roomID] = room
	log.Printf("Initialized room %s in Hub with players: %v", roomID, players)
	return room, true
}