package main

import (
	"log"
	"net/http"
	"time"

	"gorilla_chat/mongodb"
	"gorilla_chat/websocket"
)

const (
	batchSize     = 200
	thresholdTime = 5 * time.Second
)

func main() {
	// Connect to MongoDB
	mongodb.ConnectMongoDB()

	// Start the WebSocket message handler in a goroutine
	go websocket.HandleMessages(batchSize, thresholdTime)

	// Serve the frontend files from the static directory
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Handle WebSocket connections at /ws
	http.HandleFunc("/ws", websocket.HandleWebSocket)

	// Start the HTTP server
	log.Println("Server started at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server error:", err)
	}
}
