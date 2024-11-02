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
)

func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.DisconnectMongo()

	bootstrap.ClearUsers()
	bootstrap.InsertInitialUsers()

	r := mux.NewRouter()
	r.Handle("/users", enableCors(http.HandlerFunc(handlers.GetUsers))).Methods("GET")
	r.Handle("/register", enableCors(http.HandlerFunc(handlers.RegisterUser))).Methods("POST")
	r.Handle("/login", enableCors(http.HandlerFunc(handlers.LoginUser))).Methods("POST")
	r.Handle("/confirm", enableCors(http.HandlerFunc(handlers.ConfirmUser))).Methods("GET")
	r.Handle("/users/{id}", enableCors(http.HandlerFunc(handlers.GetUserByID))).Methods("GET") // Route for GetUserByID

	server := &http.Server{
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		Handler:      r,
	}

	fmt.Println("User service started on port 8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting user service:", err)
		os.Exit(1)
	}
}
