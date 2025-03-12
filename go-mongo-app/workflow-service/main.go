package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
	"time"
	"workflow-service/handler"
	"workflow-service/repoWorkflow"
)

func main() {
	// Konfiguracija za Neo4j
	neo4jURI := os.Getenv("NEO4J_URI")
	if neo4jURI == "" {
		neo4jURI = "neo4j://localhost:7687"
	}
	username := os.Getenv("NEO4J_USERNAME")
	if username == "" {
		username = "neo4j"
	}
	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		password = "password"
	}

	driver, err := neo4j.NewDriver(neo4jURI, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Error connecting to Neo4j: %v", err)
	}
	defer driver.Close()

	logger := log.New(os.Stdout, "[user-api] ", log.LstdFlags)

	repo := repoWorkflow.NewWorkflowRepository(driver)
	workflowHandler := handler.NewWorkflowHandler(repo, logger)

	log.Println("Clearing database...")
	if err := repo.ClearDatabase(context.Background()); err != nil {
		log.Fatalf("Error clearing database: %v", err)
	}

	r := mux.NewRouter()

	// Dodavanje ruta
	r.HandleFunc("/workflow/createWorkflow", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.CreateWorkflow, "Manager"))).Methods("POST")
	r.HandleFunc("/workflow/getWorkflows", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.GetWorkflowHandler, "Manager", "Member"))).Methods("GET")
	r.HandleFunc("/workflow/getTaskById/{id}", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.GetTaskByIDHandler, "Manager", "Member"))).Methods("GET")
	r.HandleFunc("/workflow/check-dependency/{task_id}", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.CheckDependencyHandler, "Manager"))).Methods("GET")
	r.HandleFunc("/workflow/{task_id}/dependencies", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.GetTaskDependenciesHandler, "Manager", "Member"))).Methods("GET")
	r.HandleFunc("/workflow/project/{project_id}", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.GetFlowByProjectIDHandler, "Manager", "Member"))).Methods("GET")
	r.HandleFunc("/workflow/delete/{task_id}", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.DeleteWorkflowByTaskIDHandler, "Manager"))).Methods("DELETE")
	r.HandleFunc("/workflow/check/{task_id}", workflowHandler.MiddlewareExtractUserFromHeader(workflowHandler.RoleRequired(workflowHandler.GetWorkflowByTaskIDHandler, "Manager", "Member"))).Methods("GET")

	// Konfiguracija CORS-a
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"}, // URL frontend aplikacije
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		Debug:            true, // Pomaže u debagovanju CORS problema
	})

	// Primena CORS middleware-a
	handler := c.Handler(r)

	server := &http.Server{
		Handler:      handler,
		Addr:         ":8080", // Port backend servisa
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Pokretanje servera
	log.Println("Workflow service running on port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting workflow service: %v", err)
	}
}
