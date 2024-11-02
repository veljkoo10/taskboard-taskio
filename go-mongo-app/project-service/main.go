package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	bootstrap "project-service/boostrap"
	"time"

	"project-service/db"
	"project-service/handlers"

	"github.com/gorilla/mux"
)

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.Client.Disconnect(context.TODO())

	bootstrap.ClearProjects()
	bootstrap.InsertInitialProjects()

	router := mux.NewRouter()
	router.HandleFunc("/projects", handlers.GetProjects).Methods("GET")
	router.HandleFunc("/projects/create", handlers.CreateProject).Methods("POST")
	router.HandleFunc("/projects/{projectId}", handlers.GetProjectByID).Methods("GET")
	router.HandleFunc("/projects/{projectId}/users/{userId}", handlers.AddUserToProject).Methods("PUT")

	server := &http.Server{
		Handler:      router,
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
