package db

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

var Client *mongo.Client

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
