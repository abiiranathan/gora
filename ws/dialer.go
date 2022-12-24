package ws

import (
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

type DefaultDialer struct {
	addr string // server server address e.g ws://localhost:8080/ws
}

func (dialer *DefaultDialer) Addr() string {
	return dialer.addr
}

func NewDialer(addr string) *DefaultDialer {
	return &DefaultDialer{addr: addr}
}

func (dialer *DefaultDialer) dial() (conn *websocket.Conn, err error) {
	c, _, err := websocket.DefaultDialer.Dial(dialer.addr, nil)
	return c, err
}

// Send message in text format to all websocket clients
func (dialer *DefaultDialer) Send(data any) error {
	conn, err := dialer.dial()
	if err != nil {
		return err
	}

	w, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	defer w.Close()
	return json.NewEncoder(w).Encode(data)
}
