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
	// Port na kojem API treba da sluÅ¡a
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	// Inicijalizacija logera
	logger := log.New(os.Stdout, "[notification-api] ", log.LstdFlags)
	storeLogger := log.New(os.Stdout, "[notification-store] ", log.LstdFlags)

	// Povezivanje sa Cassandra bazom
	store, err := repoNotification.New(storeLogger)
	if err != nil {
		logger.Fatalf("Failed to initialize Cassandra connection: %v", err)
	}
	defer store.CloseSession()

	cluster := gocql.NewCluster("cassandra") // or your cassandra host
	cluster.Keyspace = "notifications"
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal("Error connecting to Cassandra:", err)
	}
	defer session.Close()

	err = session.Query("TRUNCATE notifications").Exec()
	if err != nil {
		log.Fatal("Error truncating notifications table:", err)
	}

	store.CreateTables()

	// Kreiramo NotificationHandler sa logerom i repozitorijumom
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

	r := mux.NewRouter()
	r.HandleFunc("/notifications/user/{id}", notificationHandler.GetNotificationsByUserID).Methods("GET", "OPTIONS")
	r.HandleFunc("/notifications", notificationHandler.CreateNotification).Methods("POST")
	r.HandleFunc("/notifications/all", notificationHandler.GetAllNotifications).Methods("GET")

	r.Use(CORS)

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
		// Set CORS headers
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
