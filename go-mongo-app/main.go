package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // Role can be "NK", "M", or "Č"
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
}

var client *mongo.Client

func connectToMongo() error {
	var err error
	mongoURI := os.Getenv("MONGO_URI")
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err = mongo.Connect(context.TODO(), clientOptions)
	return err
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	collection := client.Database("testdb").Collection("users")

	var users []User
	cursor, err := collection.Find(context.TODO(), map[string]interface{}{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = cursor.All(context.TODO(), &users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	collection := client.Database("testdb").Collection("users")
	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := client.Database("testdb").Collection("users")
	var dbUser User
	err := collection.FindOne(context.TODO(), map[string]interface{}{"username": user.Username}).Decode(&dbUser)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dbUser)
}

func insertMockUsers() {
	collection := client.Database("testdb").Collection("users")

	mockUsers := []User{
		{Username: "user1", Password: "password1", Role: "NK", Name: "John", Surname: "Doe", Email: "john.doe@example.com"},
		{Username: "user2", Password: "password2", Role: "M", Name: "Jane", Surname: "Doe", Email: "jane.doe@example.com"},
		{Username: "user3", Password: "password3", Role: "Č", Name: "Alice", Surname: "Smith", Email: "alice.smith@example.com"},
	}

	for _, user := range mockUsers {
		// Hash the password before inserting
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println("Error hashing password:", err)
			continue
		}
		user.Password = string(hashedPassword)

		_, err = collection.InsertOne(context.TODO(), user)
		if err != nil {
			fmt.Println("Error inserting user:", err)
		}
	}
}

func main() {
	err := connectToMongo()
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}

	insertMockUsers() // Insert mock users

	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			fmt.Println("Error disconnecting from MongoDB:", err)
		}
	}()

	http.HandleFunc("/users", getUsers)
	http.HandleFunc("/register", registerUser)
	http.HandleFunc("/login", loginUser)

	server := &http.Server{
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Println("Server started on port 8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
}
