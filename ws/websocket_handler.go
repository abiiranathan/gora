// websocket handler built on top of guerilla/websocket
package ws

import (
	"log"
	"net/http"
	"sync"
)

/*
WebsocketHandler maintains the set of active clients and broadcasts messages to the
clients.

WebsocketHandler implements http.Handler interface and can be used directly in your http
routes. Don't forget to call the Run method in a seperate go routine.

	hub, quit := ws.NewHandler()
	defer quit()

	go hub.Run()

	http.Handle("/", http.FileServer(http.Dir("public")))
	http.Handle("/ws", hub)
	log.Fatal(http.ListenAndServe(":8080", nil))
*/
type WebsocketHandler struct {
	// Registered clients.
	clients map[*Client]bool
	// Inbound messages from the clients.
	broadcast chan []byte
	// Register requests from the clients.
	register chan *Client
	// Unregister requests from clients.
	unregister chan *Client
	// onmessage callback
	onmessage func([]byte)
	// channel to signal exit
	done chan struct{}

	// Broadcast Message to all connected clients?.
	broadcastMessages bool
}

type HubOption func(*WebsocketHandler)

func OnMessage(f func(msg []byte)) HubOption {
	return func(h *WebsocketHandler) {
		h.onmessage = f
	}
}

func NoBroadcast() HubOption {
	return func(wh *WebsocketHandler) {
		wh.broadcastMessages = false
	}
}

// Returns a new websocker hundler.
// By default, this handler broadcasts all messages to connected clients
// as in a chat. If you want to handle each message yourself, pass in an OnMessage Option and NoBroadcast option.
// Then call BroadCastMessage on this handler to send the message to all clients.
func NewHandler(options ...HubOption) (handler *WebsocketHandler, quit func()) {
	h := &WebsocketHandler{
		broadcast:         make(chan []byte),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		clients:           make(map[*Client]bool),
		onmessage:         nil,
		done:              make(chan struct{}),
		broadcastMessages: true,
	}

	for _, opt := range options {
		opt(h)
	}

	// Function to close the hub
	var once sync.Once
	closeFunc := func() {
		once.Do(func() {
			close(h.done)
		})
	}
	return h, closeFunc
}

// Infinite loop that runs the hub indefinately.
// Run this is a go routine.
func (h *WebsocketHandler) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.removeClient(client)
			}
		case message := <-h.broadcast:
			h.BroadCastMessage(message)
			if h.onmessage != nil {
				h.onmessage(message)
			}
		case <-h.done:
			// remove all clients and return
			for c := range h.clients {
				h.removeClient(c)
			}
			log.Println("quitting websocket run loop gracefully")
			return
		}
	}
}

// send message to all active clients.
// Client who can't recv are closed and deleted from the client map
func (h *WebsocketHandler) BroadCastMessage(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			h.removeClient(client)
		}
	}
}

func (h *WebsocketHandler) removeClient(client *Client) {
	close(client.send)
	delete(h.clients, client)
}

// Http handler
func (hub *WebsocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// could pass more client specific identifiers from request like client_id, authentication etc
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
