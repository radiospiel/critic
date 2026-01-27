package webui

import (
	"net/http"
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
	"golang.org/x/net/websocket"
)

// Hub manages WebSocket connections
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logger.Info("WebSocket client connected, total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			logger.Info("WebSocket client disconnected, total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					close(client.send)
					delete(h.clients, client)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(func(conn *websocket.Conn) {
		client := &Client{
			hub:  s.hub,
			conn: conn,
			send: make(chan []byte, 256),
		}

		s.hub.register <- client

		// Start goroutines for reading and writing
		go client.writePump()
		client.readPump()
	}).ServeHTTP(w, r)
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		var msg []byte
		if err := websocket.Message.Receive(c.conn, &msg); err != nil {
			break
		}
		// We don't process incoming messages for now
		// The WebSocket is primarily for server-to-client notifications
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			if err := websocket.Message.Send(c.conn, string(message)); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := websocket.Message.Send(c.conn, `{"type":"ping"}`); err != nil {
				return
			}
		}
	}
}
