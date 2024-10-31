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

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.DisconnectMongo()
	bootstrap.InsertInitialUsers()

	http.HandleFunc("/users", handlers.GetUsers)
	http.HandleFunc("/register", handlers.RegisterUser)
	http.HandleFunc("/login", handlers.LoginUser)

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
