package main

import (
	"encoding/json"
	"log"
	"net/http"
	"vuDataSim/src/handlers"

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
	handlers.AppState.Mutex.Lock()
	handlers.AppState.Clients[conn] = true
	handlers.AppState.Mutex.Unlock()

	// Send initial state
	initialState, _ := json.Marshal(handlers.AppState)
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
	handlers.AppState.Mutex.Lock()
	delete(handlers.AppState.Clients, conn)
	handlers.AppState.Mutex.Unlock()
}
