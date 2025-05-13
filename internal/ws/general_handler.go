package ws

import (
	"github.com/gorilla/websocket"
	wsPkg "github.com/krishanu7/battleship-backend/pkg/websocket"
	"log"
	"net/http"
)

type GeneralHandler struct {
	Hub *wsPkg.GeneralHub
}

func NewGeneralHandler(hub *wsPkg.GeneralHub) *GeneralHandler {
	return &GeneralHandler{
		Hub: hub,
	}
}

func (h *GeneralHandler) ServeGeneralWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsPkg.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("General WS upgrade failed: %v", err)
		return
	}
	playerID := r.URL.Query().Get("playerId")

	if playerID == "" {
		log.Println("Player ID is missing in the request")
		conn.Close()
		return
	}
	client := &wsPkg.GeneralClient{
		ID:   playerID,
		Conn: conn,
		Send: make(chan []byte),
	}
	h.Hub.AddClient(client)

	go h.read(client)
	go h.write(client)
}

func (h *GeneralHandler) read(c *wsPkg.GeneralClient) {
	defer func() {
		h.Hub.RemoveClient(c)
		c.Conn.Close()
	}()
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		// No need to handle incoming messages for now
	}
}

func (h *GeneralHandler) write(c *wsPkg.GeneralClient) {
	defer c.Conn.Close()

	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("Error writing message: %v", err)
			break
		}
	}
}
