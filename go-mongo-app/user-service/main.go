package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"user-service/bootstrap"
	"user-service/db"
	"user-service/handlers"
	"user-service/service"

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
	db.CreateTTLIndex()
	db.CreateTTLIndex2()

	bootstrap.ClearUsers()
	bootstrap.InsertInitialUsers()

	logger := log.New(os.Stdout, "[user-api] ", log.LstdFlags)
	mongoInstance := db.New(db.Client, logger)

	userService := service.NewUserService(mongoInstance, logger)

	userHandler := handlers.NewUserHandler(logger, userService)

	router := mux.NewRouter()

	// Wrap specific routes with the MiddlewareExtractUserFromHeader
	router.HandleFunc("/users/{id}/deactivate", userHandler.MiddlewareExtractUserFromHeader(userHandler.RoleRequired(userHandler.DeactivateUser, "Manager", "Member"))).Methods("PUT", "OPTIONS")
	router.HandleFunc("/users/active", userHandler.MiddlewareExtractUserFromHeader(userHandler.RoleRequired(userHandler.GetActiveUsers, "Manager", "Member"))).Methods("GET")
	router.HandleFunc("/users", userHandler.MiddlewareExtractUserFromHeader(userHandler.RoleRequired(userHandler.GetUsers, "Manager", "Member"))).Methods("GET")
	router.HandleFunc("/users/{id}", userHandler.GetUserByID).Methods("GET", "OPTIONS")
	router.HandleFunc("/reset-password", userHandler.HandleResetPassword).Methods("POST", "GET", "OPTIONS")
	router.HandleFunc("/verify-password", userHandler.HandleVerifyPassword).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/users/{id}/change-password", userHandler.MiddlewareExtractUserFromHeader(userHandler.RoleRequired(userHandler.ChangePassword, "Manager", "Member"))).Methods("POST", "OPTIONS")

	// Other routes without the middleware
	router.HandleFunc("/check-email", handlers.CheckEmail).Methods("GET", "OPTIONS")
	router.HandleFunc("/login", userHandler.LoginUser).Methods("POST", "OPTIONS")
	router.HandleFunc("/register", handlers.RegisterUser).Methods("POST", "OPTIONS")
	router.HandleFunc("/confirm", userHandler.ConfirmUser).Methods("GET", "OPTIONS")
	router.HandleFunc("/check-username", userHandler.CheckUsername).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/check-user-active", userHandler.CheckUserActive).Methods("GET", "OPTIONS")
	router.HandleFunc("/users/{id}/exists", userHandler.CheckUserExists).Methods("GET", "OPTIONS") //check this
	router.HandleFunc("/send-magic-link", userHandler.SendMagicLinkHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/verify-magic-link", userHandler.VerifyMagicLinkHandler).Methods("GET", "OPTIONS")

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
