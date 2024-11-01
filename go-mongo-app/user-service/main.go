package main

import (
	"fmt"
	"go-mongo-app/bootstrap"
	"net/http"
	"os"
	"time"

	"go-mongo-app/db"
	"go-mongo-app/handlers"
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

	bootstrap.InsertInitialUsers()
	bootstrap.ClearUsers()

	http.Handle("/users", enableCors(http.HandlerFunc(handlers.GetUsers)))
	http.Handle("/register", enableCors(http.HandlerFunc(handlers.RegisterUser)))
	http.Handle("/login", enableCors(http.HandlerFunc(handlers.LoginUser)))

	server := &http.Server{
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
