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

	config := loadConfig()
	logger := log.New(os.Stdout, "ANALYTICS-SERVICE: ", log.LstdFlags)

	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connString := fmt.Sprintf("esdb://admin:changeit@esdb:2113?tls=false")
	settings, err := esdb.ParseConnectionString(connString)
	if err != nil {
		log.Fatal(err)
	}
	client, err := esdb.NewClient(settings)
	if err != nil {
		logger.Fatal("Error initializing EventStore client: ", err)
	} else {
		logger.Println("Successfully connected to EventStore!")
	}

	esdbClient, err := repository.NewESDBClient(client, "analytics-group")
	if err != nil {
		log.Fatal("Error initializing ESDBClient:", err)
	}

	eventHandler := handlers.NewEventHandler(esdbClient)
	r := mux.NewRouter()

	// Define routes with mux variables
	r.HandleFunc("/event/append", eventHandler.ProcessEventHandler).Methods("POST")   // POST method to process event
	r.HandleFunc("/events/{projectID}", eventHandler.GetEventsHandler).Methods("GET") // GET method to retrieve events

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := corsHandler.Handler(r)

	server := &http.Server{
		Addr:         config["address"],
		Handler:      handler,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Println("Server listening on", config["address"])
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	sig := <-sigCh
	logger.Println("Received terminate signal, shutting down gracefully...", sig)

	if err := server.Shutdown(timeoutContext); err != nil {
		logger.Fatal("Error during graceful shutdown:", err)
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
