package websocket

import (
	"log"
)

type Room struct {
	ID string
	Clients map[string]*Client
}

func NewRoom(id string) *Room {
	return &Room {
		ID: id,
		Clients: make(map[string]*Client),
	}
}

func (r *Room) Broadcast(senderID string, message []byte) {
	for id, client := range r.Clients {
		if id != senderID {
			client.Send <- message
		}
	}
}
func (r *Room) AddClient(c *Client) {
	r.Clients[c.ID] = c
	c.Room = r
	log.Printf("Client %s joined room %s", c.ID, r.ID)
}