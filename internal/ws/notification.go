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

		if !w.GeneralHub.SendToClient(notification.Player, []byte(msg.Payload)) {
			log.Printf("Failed to send notification to player %s", notification.Player)
		} else {
			log.Printf("Successfully sent notification to player %s", notification.Player)
		}
	}
}