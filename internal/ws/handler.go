package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	wsPkg "github.com/krishanu7/battleship-backend/pkg/websocket"
)

type Handler struct {
	Hub *wsPkg.Hub
}

func NewHandler(hub *wsPkg.Hub) *Handler {
	return &Handler{Hub: hub}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsPkg.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade failed: %v", err)
		return
	}

	playerID := r.URL.Query().Get("playerId")
	roomID := r.URL.Query().Get("roomId")

	if playerID == "" || roomID == "" {
		log.Println("Missing playerId or roomId")
		conn.Close()
		return
	}

	room, exists := h.Hub.GetRoom(roomID)
	if !exists {
		log.Printf("Room %s does not exist for player %s", roomID, playerID)
		conn.Close()
		return
	}

	client := &wsPkg.Client{
		ID:   playerID,
		Conn: conn,
		Send: make(chan []byte, 10), // Add buffer
	}

	room.AddClient(client)

	log.Printf("Player %s connected to room %s", playerID, roomID)
	go h.read(client)
	go h.write(client)
}

func (h *Handler) read(c *wsPkg.Client) {
	defer func() {
		if c.Room != nil {
			delete(c.Room.Clients, c.ID)
			log.Printf("Client %s left room %s", c.ID, c.Room.ID)
		}
		c.Conn.Close()
	}()
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Read error for client %s: %v", c.ID, err)
			break
		}
		if c.Room != nil {
			c.Room.Broadcast(c.ID, msg)
		}
	}
}

func (h *Handler) write(c *wsPkg.Client) {
	defer c.Conn.Close()

	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("Write error for client %s: %v", c.ID, err)
			break
		}
		log.Printf("Sent message to client %s: %s", c.ID, string(msg))
	}
}