package main

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"os"
	"time"

	bootstrap "project-service/boostrap"
	"project-service/db"
	"project-service/handlers"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	err := db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.Client.Disconnect(context.TODO())

	bootstrap.ClearProjects()
	bootstrap.InsertInitialProjects()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	projectRepo := db.NewProjectRepo(db.Client)
	logger := log.New(os.Stdout, "[product-api] ", log.LstdFlags)
	projectsHandler := handlers.NewProjectsHandler(logger, projectRepo, nc)

	router := mux.NewRouter()
	router.HandleFunc("/projects/{projectId}/users", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.GetUsersForProjectHandler, "Manager", "Member"))).Methods("GET")
	router.HandleFunc("/projects/title/id", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.GetProjectIDByTitle, "Manager", "Member"))).Methods("POST")
	router.HandleFunc("/projects/user/{userId}", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.GetProjectsByUserID, "Member", "Manager"))).Methods("GET")
	router.HandleFunc("/projects", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.GetProjects, "Manager"))).Methods("GET")
	router.HandleFunc("/projects/create/{managerId}", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.CreateProject, "Manager"))).Methods("POST")
	router.HandleFunc("/projects/{projectId}", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.GetProjectByID, "Manager", "Member"))).Methods("GET", "OPTIONS")
	router.HandleFunc("/projects/{projectId}/add-users", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.AddUsersToProject, "Manager"))).Methods("PUT")
	router.HandleFunc("/projects/{projectId}/remove-users", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.RemoveUsersFromProject, "Manager"))).Methods("PUT")
	router.HandleFunc("/projects/title/{managerId}", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.HandleCheckProjectByTitle, "Manager"))).Methods("POST")
	router.HandleFunc("/projects/{projectID}/tasks/{taskID}", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.AddTaskToProjectHandler, "Manager"))).Methods("PUT", "OPTIONS")
	router.HandleFunc("/projects/isActive/{projectId}", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.IsActiveProject, "Manager", "Member"))).Methods("GET")
	router.HandleFunc("/projects/delete/{projectID}", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.DeleteProjectByIDHandler, "Manager"))).Methods("DELETE")
	router.HandleFunc("/projects/{projectID}/task-order", projectsHandler.MiddlewareExtractUserFromHeader(projectsHandler.RoleRequired(projectsHandler.UpdateTaskOrder, "Member", "Manager"))).Methods("PUT")

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
	fmt.Println("Project service started on port 8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting project service:", err)
		os.Exit(1)
	}
}
