package main

import (
	bootstrap "analytics-service/bootstrap"
	"analytics-service/db"
	"analytics-service/handlers"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/rs/cors"
	"log"
	_ "log"
	"net/http"
	"os"
	"time"
)

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.Client.Disconnect(context.TODO())
	bootstrap.ClearAnalytics()
	logger := log.New(os.Stdout, "[analytics-service] ", log.LstdFlags)
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	repo := db.NewAnalyticsRepo(db.Client)
	analyticsHandler := handlers.NewAnalyticsHandler(logger, repo, nc)

	router := mux.NewRouter()

	// A basic example route (you can add more as needed)
	router.HandleFunc("/analytics/countusers/{user_id}", analyticsHandler.MiddlewareExtractUserFromHeader(analyticsHandler.RoleRequired(analyticsHandler.CountUserTasks, "Member", "Manager"))).Methods("GET")
	router.HandleFunc("/analytics/countusersbystatus/{user_id}", analyticsHandler.MiddlewareExtractUserFromHeader(analyticsHandler.RoleRequired(analyticsHandler.CountUserTaskStatusHandler, "Member", "Manager"))).Methods("GET")
	router.HandleFunc("/analytics/usertaskproject/{user_id}", analyticsHandler.MiddlewareExtractUserFromHeader(analyticsHandler.RoleRequired(analyticsHandler.UserTasksAndProjectHandler, "Member", "Manager"))).Methods("GET")
	router.HandleFunc("/analytics/project-completion-ontime/{userId}", analyticsHandler.MiddlewareExtractUserFromHeader(analyticsHandler.RoleRequired(analyticsHandler.CheckIfProjectCompletedOnTime, "Member", "Manager"))).Methods("GET")
	router.HandleFunc("/analytics/status-change", analyticsHandler.MiddlewareExtractUserFromHeader(analyticsHandler.RoleRequired(analyticsHandler.HandleStatusChange, "Member", "Manager"))).Methods("POST")
	router.HandleFunc("/analytics/tasks", analyticsHandler.MiddlewareExtractUserFromHeader(analyticsHandler.RoleRequired(analyticsHandler.HandleGetTaskAnalytics, "Member", "Manager"))).Methods("GET")
	router.HandleFunc("/analytics/user/{userID}", analyticsHandler.MiddlewareExtractUserFromHeader(analyticsHandler.RoleRequired(analyticsHandler.GetUserTaskAnalyticsHandler, "Member", "Manager"))).Methods("GET")

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
