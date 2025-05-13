package websocket

import (
	"sync"
)

type Hub struct {
	Rooms map[string]*Room
	mu    sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[string]*Room),
	}
}

func (h *Hub) CreateRoom(id string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	room := NewRoom(id)
	h.Rooms[id] = room
	return room
}

func (h *Hub) GetRoom(id string) (*Room, bool ){
	h.mu.Lock()
	defer h.mu.Unlock()
	
	room, exists := h.Rooms[id]
	return room, exists
}
func (h *Hub) RemoveRoom(id string) {
	h.mu.Lock()

	defer h.mu.Unlock()
	delete(h.Rooms,id)
}
