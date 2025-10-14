package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin in development
	},
}

// WebSocket handler for real-time updates
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	appState.mutex.Lock()
	appState.clients[conn] = true
	appState.mutex.Unlock()

	// Send initial state
	initialState, _ := json.Marshal(appState)
	conn.WriteMessage(websocket.TextMessage, initialState)

	// Listen for client messages
	for {
		var msg []byte
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received WebSocket message: %s", msg)
	}

	// Unregister client
	appState.mutex.Lock()
	delete(appState.clients, conn)
	appState.mutex.Unlock()
}

// Broadcast updates to all WebSocket clients
func (state *AppState) broadcastUpdate() {
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	state.mutex.RLock()
	clients := make([]*websocket.Conn, 0, len(state.clients))
	for client := range state.clients {
		clients = append(clients, client)
	}
	state.mutex.RUnlock()

	for _, client := range clients {
		go func(c *websocket.Conn) {
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error: %v", err)
				state.mutex.Lock()
				delete(state.clients, c)
				state.mutex.Unlock()
				c.Close()
			}
		}(client)
	}
}
