package ws

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/krishanu7/battleship-backend/internal/game"
	wsPkg "github.com/krishanu7/battleship-backend/pkg/websocket"
)

type Handler struct {
	Hub         *wsPkg.Hub
	gameService *game.Service
}

func NewHandler(hub *wsPkg.Hub, gameService *game.Service) *Handler {
	return &Handler{
		Hub:         hub,
		gameService: gameService,
	}
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
		var message struct {
			Type       string `json:"type"`
			Coordinate string `json:"coordinate"`
		}
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Failed to unmarshal message from %s: %v", c.ID, err)
			continue
		}
		if message.Type == "attack" && c.Room != nil {
			attack, nextTurn, err := h.gameService.ProcessAttack(c.Room.ID, c.ID, message.Coordinate)
			if err != nil {
				errorMsg := struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{
					Type:    "error",
					Message: err.Error(),
				}
				errorBytes, _ := json.Marshal(errorMsg)
				c.Send <- errorBytes
				continue
			}
			// Broadcast attack result
			resultMsg := struct {
				Type       string `json:"type"`
				Coordinate string `json:"coordinate"`
				Result     string `json:"result"`
				NextTurn   string `json:"nextTurn"`
			}{
				Type:       "attack_result",
				Coordinate: attack.Coordinate,
				Result:     attack.Result,
				NextTurn:   nextTurn,
			}
			resultBytes, err := json.Marshal(resultMsg)

			if err != nil {
				log.Printf("Failed to marshal attack_result: %v", err)
				continue
			}
			c.Room.Broadcast("", resultBytes)
			// Notify next turn
			turnMsg := struct {
				Type     string `json:"type"`
				PlayerID string `json:"playerId"`
			}{
				Type:     "turn",
				PlayerID: nextTurn,
			}
			turnBytes, err := json.Marshal(turnMsg)
			if err != nil {
				log.Printf("Failed to marshal turn notification: %v", err)
				continue
			}
			c.Room.Broadcast("", turnBytes)
		} else if message.Type == "chat" {
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