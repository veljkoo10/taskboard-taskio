package main

import (
	"Hdfs/handlers"
	"Hdfs/storage"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	//Reading from environment, if not set we will default it to 8080.
	//This allows flexibility in different environments (for eg. when running multiple docker api's and want to override the default port)
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8086"
	}

	// Initialize context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Initialize the logger we are going to use, with prefix and datetime for every log
	logger := log.New(os.Stdout, "[hdfs-api] ", log.LstdFlags)
	storageLogger := log.New(os.Stdout, "[file-hdfs] ", log.LstdFlags)

	// NoSQL: Initialize File Storage store
	store, err := storage.New(storageLogger)
	if err != nil {
		logger.Fatal(err)
	}
	// Close connection to HDFS on shutdown
	defer store.Close()

	// Create directory tree on HDFS
	_ = store.CreateDirectories()

	//Initialize the handler and inject said logger
	storageHandler := handlers.NewStorageHandler(logger, store)

	//Initialize the router and add a middleware for all the requests
	router := mux.NewRouter()

	router.Use(storageHandler.MiddlewareContentTypeSet)

	copyLocalFile := router.Methods(http.MethodPost).Subrouter()
	copyLocalFile.HandleFunc("/copy", storageHandler.CopyFileToStorage)

	writeFile := router.Methods(http.MethodPost).Subrouter()
	writeFile.HandleFunc("/write", storageHandler.WriteFileToStorage)

	readFile := router.Methods(http.MethodGet).Subrouter()
	readFile.HandleFunc("/read", storageHandler.ReadFileFromStorage)

	walkRootContent := router.Methods(http.MethodGet).Subrouter()
	walkRootContent.HandleFunc("/walk", storageHandler.WalkRoot)

	cors := gorillaHandlers.CORS(gorillaHandlers.AllowedOrigins([]string{"*"}))

	//Initialize the server
	server := http.Server{
		Addr:         ":" + port,
		Handler:      cors(router),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	logger.Println("Server listening on port", port)
	//Distribute all the connections to goroutines
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			logger.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)
	signal.Notify(sigCh, os.Kill)

	sig := <-sigCh
	logger.Println("Received terminate, graceful shutdown", sig)

	//Try to shutdown gracefully
	if server.Shutdown(timeoutContext) != nil {
		logger.Fatal("Cannot gracefully shutdown...")
	}
	logger.Println("Server stopped")
}
