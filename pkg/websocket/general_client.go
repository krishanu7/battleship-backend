package websocket

import (
	"github.com/gorilla/websocket"
)

type GeneralClient struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}
