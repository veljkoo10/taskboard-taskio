package main

import (
	"context"
	"log"
	"net/http"
	"notification-service/handlers"
	"notification-service/repoNotification"
	"os"
	"os/signal"
	"time"

	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	logger := log.New(os.Stdout, "[notification-api] ", log.LstdFlags)
	storeLogger := log.New(os.Stdout, "[notification-store] ", log.LstdFlags)

	store, err := repoNotification.New(storeLogger)
	if err != nil {
		logger.Fatalf("Failed to initialize Cassandra connection: %v", err)
	}
	defer store.CloseSession()

	cluster := gocql.NewCluster("cassandra")
	cluster.Keyspace = "notifications"
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		logger.Fatalf("Error connecting to Cassandra: %v", err)
	}
	defer session.Close()

	if err := repoNotification.EnsureKeyspaceAndTable(session, logger); err != nil {
		logger.Fatalf("Error ensuring keyspace and table: %v", err)
	}

	if err := clearNotifications(session, logger); err != nil {
		logger.Fatalf("Error clearing notifications: %v", err)
	}

	store.CreateTables()

	notificationHandler := handlers.NewNotificationHandler(logger, store)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Println("Recovered in NotificationListener:", r)
			}
		}()
		notificationHandler.NotificationListener()
		logger.Println("NotificationListener invoked successfully")
	}()

	// Set up HTTP router
	r := mux.NewRouter()
	r.HandleFunc("/notifications/user/{id}", notificationHandler.MiddlewareExtractUserFromHeader(notificationHandler.RoleRequired(notificationHandler.GetNotificationsByUserID, "Member", "Manager"))).Methods("GET", "OPTIONS")
	r.HandleFunc("/notifications", notificationHandler.MiddlewareExtractUserFromHeader(notificationHandler.RoleRequired(notificationHandler.CreateNotification, "Member"))).Methods("POST")
	r.HandleFunc("/notifications/all", notificationHandler.MiddlewareExtractUserFromHeader(notificationHandler.RoleRequired(notificationHandler.GetAllNotifications, "Member"))).Methods("GET")
	r.HandleFunc("/notifications/{id}/mark-as-read", notificationHandler.MiddlewareExtractUserFromHeader(notificationHandler.RoleRequired(notificationHandler.MarkAsRead, "Member"))).Methods("PUT", "OPTIONS")

	// Apply CORS middleware
	r.Use(CORS)

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Printf("Starting server on port %s...\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on port %s: %v\n", port, err)
		}
	}()

	// Graceful shutdown handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	sig := <-sigCh
	logger.Printf("Received signal %s, shutting down...\n", sig)

	if err := server.Shutdown(context.Background()); err != nil {
		logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	logger.Println("Server stopped gracefully")
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clearNotifications(session *gocql.Session, logger *log.Logger) error {
	logger.Println("Clearing notifications table...")
	err := session.Query("TRUNCATE notifications.notifications").Exec()
	if err != nil {
		return err
	}
	logger.Println("Notifications table cleared.")
	return nil
}
