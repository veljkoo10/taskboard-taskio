package main

import (
	"analytics-service/db"
	"analytics-service/handlers"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_ "log"
	"net/http"
	"os"
	"time"
)

func main() {
	// Connect to MongoDB
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.DisconnectMongo()

	router := mux.NewRouter()

	// A basic example route (you can add more as needed)
	router.HandleFunc("/analytics/countusers/{user_id}", handlers.CountUserTasks).Methods("GET")
	router.HandleFunc("/analytics/countusersbystatus/{user_id}", handlers.CountUserTaskStatusHandler).Methods("GET")
	router.HandleFunc("/analytics/usertaskproject/{user_id}", handlers.UserTasksAndProjectHandler).Methods("GET")

	// CORS setup
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Set up server with timeouts
	server := &http.Server{
		Handler:      c.Handler(router),
		Addr:         ":8080",
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
	}

	// Start the server
	fmt.Println("Server started on port 8088")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
}
