package db

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

type AnalyticsRepo struct {
	cli *mongo.Client
}

func NewAnalyticsRepo(client *mongo.Client) *AnalyticsRepo {
	return &AnalyticsRepo{cli: client}
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
