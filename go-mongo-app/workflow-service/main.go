package main

import (
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

	r := mux.NewRouter()

	r.HandleFunc("/workflow/createWorkflow", workflowHandler.CreateTask).Methods("POST")
	r.HandleFunc("/workflow/getWorkflows", workflowHandler.GetWorkflowHandler).Methods("GET")
	
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"}, // Frontend Angular
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	server := &http.Server{
		Handler:      handler,
		Addr:         ":8084",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Pokretanje servera
	log.Println("Workflow service running on port 8084")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting workflow service: %v", err)
	}
}
