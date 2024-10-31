package service

import (
	"context"
	"go-mongo-app/db"
	"go-mongo-app/models"
	"go.mongodb.org/mongo-driver/bson"
)

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

func CreateProject(project models.Project) (models.Project, error) {
	collection := db.Client.Database("testdb").Collection("projects")
	_, err := collection.InsertOne(context.TODO(), project)
	if err != nil {
		return models.Project{}, err
	}

	return project, nil
}
