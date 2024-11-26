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

	"github.com/gorilla/mux"
)

func main() {
	// Port na kojem API treba da sluša
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8084"
	}

	// Kontekst za graceful shutdown
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Inicijalizacija logera
	logger := log.New(os.Stdout, "[notification-api] ", log.LstdFlags)
	storeLogger := log.New(os.Stdout, "[notification-store] ", log.LstdFlags)

	// Povezivanje sa Cassandra bazom
	store, err := repoNotification.New(storeLogger)
	if err != nil {
		logger.Fatalf("Failed to initialize Cassandra connection: %v", err)
	}
	defer store.CloseSession()

	// Kreiranje potrebnih tabela
	store.CreateTables()

	// Kreiramo NotificationHandler sa logerom i repozitorijumom
	notificationHandler := handlers.NewNotificationHandler(logger, store)

	// Postavljamo rute za API
	router := mux.NewRouter()

	// Dodajemo rute
	router.HandleFunc("/notifications/GetAll", notificationHandler.GetAllNotificationsHandler).Methods("GET")
	router.HandleFunc("/notifications/create", notificationHandler.CreateNotificationHandler).Methods("POST")

	// Dodajemo CORS middleware
	router.Use(CORS)

	// Kreiranje HTTP servera
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Pokretanje servera u gorutini
	go func() {
		logger.Printf("Starting server on port %s...\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on port %s: %v\n", port, err)
		}
	}()

	// Čekamo na signal za prekid
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	sig := <-sigCh
	logger.Printf("Received signal %s, shutting down...\n", sig)

	// Graceful shutdown
	if err := server.Shutdown(timeoutContext); err != nil {
		logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	logger.Println("Server stopped gracefully")
}

// CORS middleware
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Postavljanje CORS headera
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
