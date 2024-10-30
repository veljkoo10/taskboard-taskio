package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Project struct {
	ID          string `json:"id,omitempty"` // Optional ID field for response
	Title       string `json:"title"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
}

var client *mongo.Client

func connectToMongo() error {
	mongoURI := os.Getenv("MONGO_URI")
	clientOptions := options.Client().ApplyURI(mongoURI)
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	return err
}

func getProjects(w http.ResponseWriter, r *http.Request) {
	collection := client.Database("testdb").Collection("projects")

	var projects []Project
	cursor, err := collection.Find(context.TODO(), map[string]interface{}{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = cursor.All(context.TODO(), &projects); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func createProject(w http.ResponseWriter, r *http.Request) {
	var project Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := client.Database("testdb").Collection("projects")
	_, err := collection.InsertOne(context.TODO(), project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func main() {
	err := connectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer client.Disconnect(context.TODO())

	http.HandleFunc("/projects", getProjects)
	http.HandleFunc("/projects/create", createProject)

	server := &http.Server{
		Addr:         ":8080", // Internal port for service
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Println("Project service started on port 8081")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting project service:", err)
		os.Exit(1)
	}
}
