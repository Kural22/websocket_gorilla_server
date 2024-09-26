package mongodb

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var chatCollection *mongo.Collection

// ChatMessage represents the structure of a chat message
type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username  string             `bson:"username" json:"username"`
	Message   string             `bson:"message" json:"message"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
}

// ConnectMongoDB establishes a connection to MongoDB
func ConnectMongoDB() {
	clientOptions := options.Client().ApplyURI("mongodb://mongo3:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("MongoDB ping error:", err)
	}

	chatCollection = client.Database("chat_app").Collection("gorilla_messages")
	log.Println("Connected to MongoDB!")
}

// StoreMessage saves a chat message to MongoDB
func StoreMessage(username string, message string) {
	chatMessage := ChatMessage{
		Username:  username,
		Message:   message,
		Timestamp: time.Now(),
	}

	_, err := chatCollection.InsertOne(context.TODO(), chatMessage)
	if err != nil {
		log.Println("Error inserting message:", err)
	}
}

// RetrieveMessages fetches the last 100 chat messages from MongoDB
func RetrieveMessages() ([]ChatMessage, error) {
	var messages []ChatMessage
	cursor, err := chatCollection.Find(context.TODO(), bson.M{}, options.Find().SetLimit(100))
	if err != nil {
		return nil, err
	}

	if err := cursor.All(context.TODO(), &messages); err != nil {
		return nil, err
	}

	return messages, nil
}
