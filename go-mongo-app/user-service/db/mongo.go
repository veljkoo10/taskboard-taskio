package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
)

var Client *mongo.Client

type Mongo struct {
	cli    *mongo.Client
	logger *log.Logger
}

func New(client *mongo.Client, logger *log.Logger) *Mongo {
	return &Mongo{cli: client, logger: logger}
}

func ConnectToMongo() error {
	mongoURI := os.Getenv("MONGO_URI")
	clientOptions := options.Client().ApplyURI(mongoURI)
	var err error
	Client, err = mongo.Connect(context.TODO(), clientOptions)
	return err
}

func DisconnectMongo() error {
	return Client.Disconnect(context.TODO())
}
func CreateTTLIndex() {
	// Pristupanje kolekciji password_resets u bazi testdb
	collection := Client.Database("testdb").Collection("password_resets")

	// Kreiranje TTL indeksa na polju expiresAt
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "expiresAt", Value: 1}, // Indeksiraj prema expiresAt polju
		},
		Options: options.Index().SetExpireAfterSeconds(0), // Automatsko brisanje kada vreme istekne
	}

	// Kreiranje indeksa
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Fatal("Failed to create TTL index:", err)
	}
}
func CreateTTLIndex2() {
	// Pristupanje kolekciji password_resets u bazi testdb
	collection := Client.Database("testdb").Collection("confirmations")

	// Kreiranje TTL indeksa na polju expiresAt
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "expiresAt", Value: 1}, // Indeksiraj prema expiresAt polju
		},
		Options: options.Index().SetExpireAfterSeconds(0), // Automatsko brisanje kada vreme istekne
	}

	// Kreiranje indeksa
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Fatal("Failed to create TTL index:", err)
	}
}
