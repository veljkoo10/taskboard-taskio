package db

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

type ProjectRepo struct {
	cli *mongo.Client
}

func NewProjectRepo(client *mongo.Client) *ProjectRepo {
	return &ProjectRepo{cli: client}
}
