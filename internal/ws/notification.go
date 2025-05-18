package ws

import (
	"encoding/json"
	"log"

	rdbPkg "github.com/krishanu7/battleship-backend/pkg/redis"
	wsPkg "github.com/krishanu7/battleship-backend/pkg/websocket"
	"github.com/redis/go-redis/v9"
)

type NotificationWorker struct {
	RedisClient *redis.Client
	GeneralHub  *wsPkg.GeneralHub
}

func NewNotificationWorker(rdb *redis.Client, hub *wsPkg.GeneralHub) *NotificationWorker {
	return &NotificationWorker{
		RedisClient: rdb,
		GeneralHub:  hub,
	}
}

func (w *NotificationWorker) Run() {
	log.Println("Notification worker starting...")
	pubsub := w.RedisClient.Subscribe(rdbPkg.Ctx, "notifications")
	defer pubsub.Close()

	for {
		log.Println("Waiting for notification messages...")
		msg, err := pubsub.ReceiveMessage(rdbPkg.Ctx)
		if err != nil {
			log.Printf("Notification pub/sub error: %v", err)
			continue
		}
		log.Printf("Received notification: %s", msg.Payload)

		var notification struct {
			Type   string `json:"type"`
			RoomID string `json:"roomId"`
			Player string `json:"player"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
			log.Printf("Failed to unmarshal notification: %v", err)
			continue
		}
		// log.Printf("Parsed notification for player %s: type=%s, roomId=%s", notification.Player, notification.Type, notification.RoomID)
		// Forward to the specific player via GeneralHub
		if !w.GeneralHub.SendToClient(notification.Player, []byte(msg.Payload)) {
			log.Printf("Failed to send notification to player %s", notification.Player)
		} else {
			log.Printf("Successfully sent notification to player %s", notification.Player)
		}
		// Check if both players placed ships
		if notification.Type == "ships_placed" {
			players, err := w.RedisClient.SMembers(rdbPkg.Ctx, "room:"+notification.RoomID).Result()

			if err != nil {
				log.Printf("Failed to get room members: %v", err)
				continue
			}
			bothReady := true
			for _, player := range players {
				key := "room:" + notification.RoomID + ":board:" + player

				exists, err := w.RedisClient.Exists(rdbPkg.Ctx, key).Result()

				if err != nil || exists == 0 {
					bothReady = false
					break
				}
			}
			if bothReady {
				// Notify both players that the game can start
				gameStartMsg := struct {
					Type   string `json:"type"`
					RoomID string `json:"roomId"`
				}{
					Type:   "game_start",
					RoomID: notification.RoomID,
				}
				msgBytes, err := json.Marshal(gameStartMsg)
				if err != nil {
					log.Printf("Failed to marshal game_start notification: %v", err)
					continue
				}
				for _, player := range players {
					if !w.GeneralHub.SendToClient(player, msgBytes) {
						log.Printf("Failed to send game_start to player %s", player)
					} else {
						log.Printf("Sent game_start to player %s", player)
					}
				}
			}
		}
	}
}
