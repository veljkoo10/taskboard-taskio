package db

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

type TaskRepo struct {
	cli *mongo.Client
}

func NewTaskRepo(client *mongo.Client) *TaskRepo {
	return &TaskRepo{cli: client}
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
