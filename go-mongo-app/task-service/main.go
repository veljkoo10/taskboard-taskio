package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
	bootstrap "task-service/boostrap"
	"task-service/db"
	"task-service/handlers"
	"time"
)

func main() {
	// Učitaj .env fajl
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using default values")
	}

	// Čitanje vrednosti iz environment varijabli
	hdfsNameNodeAddress := os.Getenv("HDFS_NAMENODE_ADDRESS")
	if hdfsNameNodeAddress == "" {
		hdfsNameNodeAddress = "hdfs://localhost:9000"
	}

	// Logovanje HDFS NameNode adrese
	log.Println("HDFS NameNode Address: ", hdfsNameNodeAddress)

	// Veza sa MongoDB
	err = db.ConnectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	defer db.DisconnectMongo()

	bootstrap.ClearTasks()
	bootstrap.InsertInitialTasks()

	// Veza sa NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	// Postavke loggera
	logger := log.New(os.Stdout, "[product-api] ", log.LstdFlags)
	taskRepo := db.NewTaskRepo(db.Client)

	tasksHandler := handlers.NewTasksHandler(logger, taskRepo, nc)

	// Postavke routera
	router := mux.NewRouter()
	router.HandleFunc("/tasks", tasksHandler.GetTasks).Methods("GET")
	router.HandleFunc("/tasks/{taskId}", tasksHandler.GetTaskByID).Methods("GET", "OPTIONS")
	router.HandleFunc("/tasks/create/{project_id}", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.CreateTaskHandler, "Manager"))).Methods("POST")
	router.HandleFunc("/tasks/{taskId}/users/{userId}", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.AddUserToTaskHandler, "Manager"))).Methods("PUT")
	router.HandleFunc("/tasks/{taskId}/users/{userId}", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.RemoveUserFromTaskHandler, "Manager"))).Methods("DELETE")
	router.HandleFunc("/tasks/{taskID}/users", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.GetUsersForTaskHandler, "Manager", "Member"))).Methods("GET")
	router.HandleFunc("/tasks/{taskId}", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.UpdateTaskHandler, "Member"))).Methods("PUT")
	router.HandleFunc("/tasks/{taskId}/member-of/{userId}", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.CheckUserInTaskHandler, "Manager", "Member"))).Methods("GET")
	router.HandleFunc("/tasks/{task_id}/dependencies/{dependency_id}", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.AddDependencyHandler, "Manager"))).Methods("PUT")
	router.HandleFunc("/tasks/projects/{project_id}/tasks", tasksHandler.MiddlewareExtractUserFromHeader(tasksHandler.RoleRequired(tasksHandler.GetTasksForProjectHandler, "Manager"))).Methods("GET")
	router.HandleFunc("/tasks/{task_id}/dependenciesWork", tasksHandler.GetDependenciesForTaskHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/tasks/upload", tasksHandler.UploadFileHandler).Methods("POST")
	router.HandleFunc("/tasks/{taskID}/download/{fileName:.+}", tasksHandler.DownloadFileHandler).Methods("GET")
	router.HandleFunc("/tasks/files/{taskID}", tasksHandler.GetTaskFilesHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/tasks/exists", tasksHandler.TaskExistsHandler).Methods("POST")
	router.HandleFunc("/tasks/delete/{taskID}", tasksHandler.DeleteTaskByIDHandler).Methods("DELETE")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	server := &http.Server{
		Handler:      c.Handler(router),
		Addr:         ":8080",
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
	}

	fmt.Println("Task service started on port 8082")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting task service:", err)
		os.Exit(1)
	}
}
