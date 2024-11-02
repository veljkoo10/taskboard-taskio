package service

import (
	"context"
	"errors"
	"project-service/db"
	"project-service/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// userExists proverava da li korisnik sa datim ID-jem postoji u bazi
func userExists(userID string) (bool, error) {
	userCollection := db.Client.Database("testdb").Collection("users")
	var user models.User

	// Convert userID string to MongoDB ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID format")
	}

	err = userCollection.FindOne(context.TODO(), bson.M{"_id": userObjectID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return false, nil // User not found
	} else if err != nil {
		return false, err // Other errors
	}

	return true, nil // User exists
}

func GetAllProjects() ([]models.Project, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	var projects []models.Project

	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if err = cursor.All(context.TODO(), &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func CreateProject(project models.Project) (string, error) {
	expectedEndDate, err := time.Parse("2006-01-02", project.ExpectedEndDate)
	if err != nil {
		return "", errors.New("invalid expected end date format, must be YYYY-MM-DD")
	}

	if err := validateProject(project, expectedEndDate); err != nil {
		return "", err
	}

	collection := db.Client.Database("testdb").Collection("projects")
	_, err = collection.InsertOne(context.TODO(), project)
	if err != nil {
		return "", err
	}

	return "Successfully saved to the database", nil
}

func validateProject(project models.Project, expectedEndDate time.Time) error {
	if project.MinPeople < 1 || project.MaxPeople < project.MinPeople {
		return errors.New("invalid min/max people values")
	}
	if expectedEndDate.Before(time.Now()) {
		return errors.New("Expected date must be in the future")
	}
	exists, err := projectExists(project.Title)
	if err != nil {
		return err // Return any error encountered during the check
	}
	if exists {
		return errors.New("Project with this name already exists")
	}

	return nil
}
func projectExists(title string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project

	err := collection.FindOne(context.TODO(), bson.M{"title": title}).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // No project found
		}
		return false, err
	}
	return true, nil
}

// AddUserToProject dodaje korisnika na projekat nakon što proveri validacije
func AddUserToProject(projectID string, userID string) error {

	// Check if the user exists in the users collection
	userExists, err := userExists(userID)
	if err != nil {
		return err
	}
	if !userExists {
		return errors.New("user not found")
	}

	// Convert projectID string to MongoDB ObjectID
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return errors.New("invalid project ID format")
	}

	// Retrieve the project from the collection by its ID
	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return errors.New("project not found")
	} else if err != nil {
		return err
	}

	// Check if the number of users has reached the maximum allowed for the project
	userCount := len(project.Users)
	if userCount >= project.MaxPeople {
		return errors.New("maximum number of users reached for this project")
	}

	// Add the user to the project's users array
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": projectObjectID},
		bson.M{"$addToSet": bson.M{"users": userID}}, // Use $addToSet to prevent duplicate users
	)
	if err != nil {
		return err
	}

	return nil
}

// countProjectUsers vraća broj korisnika u projektu
func countProjectUsers(projectID string) (int, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return 0, errors.New("invalid project ID")
	}

	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err != nil {
		return 0, err
	}

	return len(project.Users), nil
}

func GetProjectByID(projectID string) (*models.Project, error) {
	projectObjectID, err := primitive.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, errors.New("invalid project ID")
	}

	collection := db.Client.Database("testdb").Collection("projects")
	var project models.Project
	err = collection.FindOne(context.TODO(), bson.M{"_id": projectObjectID}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("project not found")
	} else if err != nil {
		return nil, err
	}

	return &project, nil
}
