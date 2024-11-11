package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
	"user-service/bootstrap"
	"user-service/db"
	"user-service/handlers"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.DisconnectMongo()

	bootstrap.ClearUsers()
	bootstrap.InsertInitialUsers()

	router := mux.NewRouter()
	router.HandleFunc("/users/active", handlers.GetActiveUsers).Methods("GET")
	router.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	router.HandleFunc("/register", handlers.RegisterUser).Methods("POST", "OPTIONS")
	router.HandleFunc("/login", handlers.LoginUser).Methods("POST", "OPTIONS")
	router.HandleFunc("/confirm", handlers.ConfirmUser).Methods("GET", "OPTIONS")
	router.HandleFunc("/check-email", handlers.CheckEmail).Methods("GET", "OPTIONS")
	router.HandleFunc("/check-username", handlers.CheckUsername).Methods("GET", "OPTIONS")
	router.HandleFunc("/users/{id}", handlers.GetUserByID).Methods("GET", "OPTIONS")
	router.HandleFunc("/reset-password", handlers.HandleResetPassword).Methods("POST", "GET", "OPTIONS")
	router.HandleFunc("/verify-password", handlers.HandleVerifyPassword).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/api/check-user-active", handlers.CheckUserActive).Methods("GET", "OPTIONS")
	router.HandleFunc("/users/{id}/change-password", handlers.ChangePassword).Methods("POST", "OPTIONS")
	router.HandleFunc("/users/{id}/exists", handlers.CheckUserExists).Methods("GET", "OPTIONS")

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

	fmt.Println("User service started on port 8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting user service:", err)
		os.Exit(1)
	}
}
