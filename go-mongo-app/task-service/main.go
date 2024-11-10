package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
	"os"
	"task-service/boostrap"
	"task-service/db"
	"task-service/handlers"
	"time"
)

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.DisconnectMongo()

	bootstrap.ClearTasks()
	bootstrap.InsertInitialTasks()

	router := mux.NewRouter()
	router.HandleFunc("/tasks", handlers.GetTasks).Methods("GET")
	router.HandleFunc("/tasks/create/{project_id}", handlers.CreateTaskHandler).Methods("POST")
	router.HandleFunc("/tasks/{taskId}/users/{userId}", handlers.AddUserToTask).Methods("PUT")
	router.HandleFunc("/tasks/{taskId}/users/{userId}", handlers.RemoveUserFromTask).Methods("DELETE")
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

	fmt.Println("Task service started on port 8082")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting task service:", err)
		os.Exit(1)
	}
}
