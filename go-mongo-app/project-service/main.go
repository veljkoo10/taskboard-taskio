package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	bootstrap "project-service/boostrap"
	"project-service/db"
	"project-service/handlers"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
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
	router.HandleFunc("/projects/{projectId}/users/{userId}", handlers.RemoveUserFromProject).Methods("DELETE")
	router.HandleFunc("/projects/title/{title}", handlers.HandleCheckProjectByTitle).Methods("GET")
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	server := &http.Server{
		Handler:      c.Handler(router),
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
