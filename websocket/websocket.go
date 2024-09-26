package websocket

import (
	"gorilla_chat/mongodb"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	_websocket "github.com/gorilla/websocket"
)

// Client struct to store the WebSocket connection and Mutex for safe writes
type Client struct {
	Conn  *_websocket.Conn
	Mutex sync.Mutex
}

var clients = make(map[*Client]bool)
var broadcast = make(chan mongodb.ChatMessage, 200)
var mutex sync.Mutex

var batchMutex sync.Mutex

var upgrader = _websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWebSocket manages new WebSocket connections
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {

	startTime := time.Now()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer func() {
		conn.Close()
		disconnectTime := time.Now()
		elapsedTime := disconnectTime.Sub(startTime)
		log.Printf("Connection closed. Time connected: %v\n", elapsedTime)
	}()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		userID = uuid.New().String()
	}

	log.Printf("Connection established with user-id: %s. Time taken to connect: %v\n", userID, time.Since(startTime))

	client := &Client{Conn: conn}

	// Add client to the clients map
	mutex.Lock()
	clients[client] = true
	mutex.Unlock()

	// Send chat history to the new connection
	messages, _ := mongodb.RetrieveMessages()
	for _, msg := range messages {
		client.Mutex.Lock()
		err = client.Conn.WriteJSON(msg)
		client.Mutex.Unlock()
		if err != nil {
			// log.Println("WebSocket write error:", err)
			conn.Close()
			mutex.Lock()
			delete(clients, client)
			mutex.Unlock()
			return
		}
	}
	log.Println("The total number of clients will be ", len(clients))
	// Read messages from this connection and broadcast them
	for {
		var msg mongodb.ChatMessage
		err := conn.ReadJSON(&msg)
		// log.Printf("The new msg received from user will be %v to %v", conn, msg)
		if err != nil {
			// log.Println("WebSocket read error:", err)
			mutex.Lock()
			delete(clients, client)
			mutex.Unlock()
			break
		}

		// Add the current time as a timestamp and store the message
		msg.Username = userID
		msg.Timestamp = time.Now()
		mongodb.StoreMessage(msg.Username, msg.Message)

		// Broadcast the message to all clients
		broadcast <- msg
	}
}

// HandleMessages broadcasts messages to all connected clients
// func HandleMessages() {
// 	for {
// 		msg := <-broadcast

// 		// Send the message to all connected clients
// 		mutex.Lock()
// 		for client := range clients {
// 			go func(client *Client) {
// 				client.Mutex.Lock()
// 				err := client.Conn.WriteJSON(msg)
// 				// log.Printf("The broadcasting msg will be %v to %v", client.Conn, msg)
// 				client.Mutex.Unlock()

// 				if err != nil {
// 					// log.Println("WebSocket write error:", err)
// 					client.Conn.Close()
// 					mutex.Lock()
// 					delete(clients, client)
// 					mutex.Unlock()
// 				}
// 			}(client)
// 		}
// 		mutex.Unlock()
// 	}
// }

func HandleMessages(batchSize int, thresholdTime time.Duration) {
	broadcastBatch := make([]mongodb.ChatMessage, 0, batchSize)
	ticker := time.NewTicker(thresholdTime)
	defer ticker.Stop()

	for {
		select {
		case msg := <-broadcast:
			batchMutex.Lock()
			broadcastBatch = append(broadcastBatch, msg)
			batchMutex.Unlock()

		case <-ticker.C:
			batchMutex.Lock()
			if len(broadcastBatch) > 0 {
				sendBatchSize := batchSize
				if len(broadcastBatch) < batchSize {
					sendBatchSize = len(broadcastBatch)
				}
				sendBatchMessages(broadcastBatch[:sendBatchSize])

				broadcastBatch = broadcastBatch[sendBatchSize:]
			}
			batchMutex.Unlock()
		}
	}
}

func sendBatchMessages(broadcastBatch []mongodb.ChatMessage) {

	log.Println("The total number of connected clients will be", len(clients))
	log.Printf("Broadcasting %d messages", len(broadcastBatch))

	// // Send the batch to all clients
	// for conn := range clients {
	// 	err := wsjson.Write(context.Background(), conn, broadcastBatch)
	// 	if err != nil {
	// 		conn.Close(websocket.StatusInternalError, "Internal Error")
	// 		mutex.Lock()
	// 		delete(clients, conn)
	// 		mutex.Unlock()
	// 	}
	// }

	mutex.Lock()
	for client := range clients {
		go func(client *Client) {
			client.Mutex.Lock()
			err := client.Conn.WriteJSON(broadcastBatch)
			// log.Printf("The broadcasting msg will be %v to %v", client.Conn, msg)
			client.Mutex.Unlock()

			if err != nil {
				// log.Println("WebSocket write error:", err)
				client.Conn.Close()
				mutex.Lock()
				delete(clients, client)
				mutex.Unlock()
			}
		}(client)
	}
	mutex.Unlock()

	// Clear the batch after broadcasting
	broadcastBatch = nil
}
