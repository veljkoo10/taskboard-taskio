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

	repo := repoWorkflow.NewWorkflowRepository(driver)
	workflowHandler := handler.NewWorkflowHandler(repo)

	log.Println("Clearing database...")
	if err := repo.ClearDatabase(context.Background()); err != nil {
		log.Fatalf("Error clearing database: %v", err)
	}

	r := mux.NewRouter()

	// Dodavanje ruta
	r.HandleFunc("/workflow/createWorkflow", workflowHandler.CreateWorkflow).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/workflow/getWorkflows", workflowHandler.GetWorkflowHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/workflow/getTaskById/{id}", workflowHandler.GetTaskByIDHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/workflow/check-dependency/{task_id}", workflowHandler.CheckDependencyHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/workflow/{task_id}/dependencies", workflowHandler.GetTaskDependenciesHandler).Methods(http.MethodGet, http.MethodOptions)

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
