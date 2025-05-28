package ws

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/krishanu7/battleship-backend/internal/game"
	"github.com/krishanu7/battleship-backend/pkg/redis"
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
		Send: make(chan []byte, 10),
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
			Message    string `json:"message"`
		}
		if err := json.Unmarshal(msg, &message); err == nil {
			log.Printf("Received JSON message from %s: type=%s", c.ID, message.Type)
			if message.Type == "attack" && c.Room != nil {
				log.Printf("Processing attack from %s: %s", c.ID, message.Coordinate)
				attack, sunkShips, gameOver, err := h.gameService.ProcessAttack(c.Room.ID, c.ID, message.Coordinate)
				if err != nil {
					log.Printf("Attack error for %s: %v", c.ID, err)
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
				nextTurn := ""
				if gameOver == nil {
					nextTurn = h.getCurrentTurn(c.Room.ID)
				}
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
				log.Printf("Broadcasting attack_result: %s", string(resultBytes))
				c.Room.Broadcast("", resultBytes)

				// Broadcast sunk ships
				for _, ship := range sunkShips {
					sunkMsg := struct {
						Type     string `json:"type"`
						Ship     string `json:"ship"`
						PlayerID string `json:"playerId"`
					}{
						Type:     "ship_sunk",
						Ship:     ship,
						PlayerID: c.ID,
					}
					sunkBytes, err := json.Marshal(sunkMsg)
					if err != nil {
						log.Printf("Failed to marshal ship_sunk: %v", err)
						continue
					}
					log.Printf("Broadcasting ship_sunk: %s", string(sunkBytes))
					c.Room.Broadcast("", sunkBytes)
				}

				// Broadcast game over
				if gameOver != nil {
					gameOverMsg := struct {
						Type   string `json:"type"`
						Winner string `json:"winner"`
						Loser  string `json:"loser"`
					}{
						Type:   "game_over",
						Winner: gameOver.Winner,
						Loser:  gameOver.Loser,
					}
					gameOverBytes, err := json.Marshal(gameOverMsg)
					if err != nil {
						log.Printf("Failed to marshal game_over: %v", err)
						continue
					}
					log.Printf("Broadcasting game_over: %s", string(gameOverBytes))
					c.Room.Broadcast("", gameOverBytes)
				} else {
					// Notify next turn
					turnMsg := struct {
						Type     string `json:"type"`
						PlayerID string `json:"playerId"`
					}{
						Type:     "turn",
						PlayerID: h.getCurrentTurn(c.Room.ID),
					}
					turnBytes, err := json.Marshal(turnMsg)
					if err != nil {
						log.Printf("Failed to marshal turn notification: %v", err)
						continue
					}
					log.Printf("Broadcasting turn: %s", string(turnBytes))
					c.Room.Broadcast("", turnBytes)
				}
			} else if message.Type == "chat" && c.Room != nil {
				chatMsg := struct {
					Type    string `json:"type"`
					Sender  string `json:"sender"`
					Message string `json:"message"`
				}{
					Type:    "chat",
					Sender:  c.ID,
					Message: message.Message,
				}
				chatBytes, err := json.Marshal(chatMsg)
				if err != nil {
					log.Printf("Failed to marshal chat message: %v", err)
					continue
				}
				log.Printf("Broadcasting chat from %s: %s", c.ID, string(chatBytes))
				c.Room.Broadcast(c.ID, chatBytes)
			}
		} else {
			log.Printf("Received plain text from %s: %s", c.ID, string(msg))
			if c.Room != nil {
				chatMsg := struct {
					Type    string `json:"type"`
					Sender  string `json:"sender"`
					Message string `json:"message"`
				}{
					Type:    "chat",
					Sender:  c.ID,
					Message: string(msg),
				}
				chatBytes, err := json.Marshal(chatMsg)
				if err != nil {
					log.Printf("Failed to marshal plain text chat message: %v", err)
					continue
				}
				log.Printf("Broadcasting plain text chat from %s: %s", c.ID, string(chatBytes))
				c.Room.Broadcast(c.ID, chatBytes)
			}
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

// getCurrentTurn retrieves the current turn from game state.
func (h *Handler) getCurrentTurn(roomID string) string {
	gameJSON, err := h.gameService.Rdb.Get(redis.Ctx, "room:"+roomID+":game").Result()
	if err != nil {
		log.Printf("Failed to get game state for turn: %v", err)
		return ""
	}
	var gameState game.GameState
	if err := json.Unmarshal([]byte(gameJSON), &gameState); err != nil {
		log.Printf("Failed to unmarshal game state for turn: %v", err)
		return ""
	}
	return gameState.Turn
}