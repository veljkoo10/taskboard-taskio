package main

import (
	"context"
	"fmt"
	bootstrap "go-mongo-app/boostrap"
	"net/http"
	"os"
	"time"

	"go-mongo-app/db"
	"go-mongo-app/handlers"
)

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.Client.Disconnect(context.TODO())
	
	bootstrap.InsertInitialProjects()
	bootstrap.ClearProjects()

	http.HandleFunc("/projects", handlers.GetProjects)
	http.HandleFunc("/projects/create", handlers.CreateProject)

	server := &http.Server{
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Println("Project service started on port 8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting project service:", err)
		os.Exit(1)
	}
}
