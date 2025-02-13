package main

import (
	"event_sourcing/handlers"
	"event_sourcing/repository"
	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"context"
	"fmt"
	"github.com/EventStore/EventStore-Client-Go/esdb"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	// Konfigurisanje logger-a
	logger := log.New(os.Stdout, "ANALYTICS-SERVICE: ", log.LstdFlags)
	logger.Println("Starting analytics service...")

	// Učitavanje konfiguracije
	config := loadConfig()
	logger.Println("Configuration loaded:", config)

	// Timeout context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// EventStore povezivanje
	connString := fmt.Sprintf("esdb://admin:changeit@esdb:2113?tls=false")
	logger.Println("Connecting to EventStore with connection string:", connString)

	settings, err := esdb.ParseConnectionString(connString)
	if err != nil {
		logger.Fatalf("Failed to parse EventStore connection string: %v", err)
	}
	client, err := esdb.NewClient(settings)
	if err != nil {
		logger.Fatalf("Error initializing EventStore client: %v", err)
	}
	logger.Println("Successfully connected to EventStore!")

	// Inicijalizacija klijenta
	esdbClient, err := repository.NewESDBClient(client, "analytics-group")
	if err != nil {
		logger.Fatalf("Error initializing ESDBClient: %v", err)
	}
	logger.Println("ESDBClient initialized successfully.")

	// Delete all events before starting the server
	err = esdbClient.DeleteAllEvents()
	if err != nil {
		logger.Fatalf("Error deleting events: %v", err)
	} else {
		logger.Println("All events have been deleted successfully.")
	}

	// Konfigurisanje HTTP ruta
	eventHandler := handlers.NewEventHandler(esdbClient)
	r := mux.NewRouter()
	r.HandleFunc("/event/append", eventHandler.ProcessEventHandler).Methods("POST")
	r.HandleFunc("/events", eventHandler.GetAllEventsHandler).Methods("GET") // Sada je kraće
	logger.Println("Routes configured successfully.")

	// CORS konfiguracija
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	handler := corsHandler.Handler(r)

	// Konfiguracija HTTP servera
	server := &http.Server{
		Addr:         config["address"],
		Handler:      handler,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Pokretanje servera u gorutini
	go func() {
		logger.Println("Server is starting on", config["address"])
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on %s: %v\n", config["address"], err)
		}
	}()

	// Signal za gašenje servera
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	sig := <-sigCh
	logger.Println("Received terminate signal, shutting down gracefully:", sig)

	if err := server.Shutdown(timeoutContext); err != nil {
		logger.Fatalf("Error during graceful shutdown: %v", err)
	}
	logger.Println("Server gracefully stopped")
}

func loadConfig() map[string]string {
	config := make(map[string]string)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	config["address"] = fmt.Sprintf(":%s", port)

	return config
}
