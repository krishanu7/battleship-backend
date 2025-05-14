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
	log.Printf("General client %s connected, total clients: %d", c.ID, len(h.Clients))
}

func (h *GeneralHub) RemoveClient(c *GeneralClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.Clients, c.ID)
	log.Printf("General client %s disconnected, total clients: %d", c.ID, len(h.Clients))
}

func (h *GeneralHub) SendToClient(playerID string, message []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.Clients[playerID]
	if !exists {
		log.Printf("Client %s not found in GeneralHub", playerID)
		return false
	}

	log.Printf("Sending message to client %s: %s", playerID, string(message))
	select {
	case client.Send <- message:
		log.Printf("Message sent to client %s", playerID)
		return true
	default:
		log.Printf("Failed to send message to client %s: channel blocked", playerID)
		return false
	}
}