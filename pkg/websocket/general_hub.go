package websocket

import (
	"log"
	"sync"
)

type GeneralHub struct {
	Clients map[string]*GeneralClient
	mu      sync.Mutex
}

func NewGeneralHub() *GeneralHub {
	return &GeneralHub{
		Clients: make(map[string]*GeneralClient),
	}
}

func (h *GeneralHub) AddClient(c *GeneralClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Clients[c.ID] = c
	log.Printf("General client %s connected", c.ID)
}

func (h *GeneralHub) RemoveClient(c *GeneralClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.Clients, c.ID)
	log.Printf("General client %s disconnected", c.ID)
}

func (h *GeneralHub) SendToClient(playerID string, message []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.Clients[playerID]

	if !exists {
		return false
	}

	select {
	case client.Send <- message:
		return true
	default:
		return false
	}
}
