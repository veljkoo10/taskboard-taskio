package service

import (
	"analytics-service/db"
	"analytics-service/models"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type AnalyticsService struct {
	Collection *mongo.Collection
}

// NewAnalyticsService creates a new AnalyticsService
func NewAnalyticsService(client *mongo.Client) *AnalyticsService {
	return &AnalyticsService{
		Collection: client.Database("analytics").Collection("task_analytics"),
	}
}

// CountUserTasks - Funkcija za brojanje taskova na kojima je korisnik
func CountUserTasks(userID string, token string) (int, error) {
	// Pozivamo task-service da preuzmemo sve taskove
	taskServiceEndpoint := fmt.Sprintf("http://task-service:8080/tasks")

	// Kreiranje HTTP GET zahteva
	req, err := http.NewRequest("GET", taskServiceEndpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	// Postavi Authorization header sa Bearer tokenom
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Slanje HTTP GET zahteva
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch tasks from task-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("task-service returned status: %d", resp.StatusCode)
	}

	// Čitamo telo odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response from task-service: %v", err)
	}

	// Parsiramo JSON u listu taskova
	var tasks []models.Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return 0, fmt.Errorf("failed to parse tasks data: %v", err)
	}

	// Brojanje taskova na kojima je userID dodat
	count := 0
	for _, task := range tasks {
		for _, user := range task.Users {
			if user == userID {
				count++
				break
			}
		}
	}

	return count, nil
}

// CountUserTasksByStatus - Funkcija za brojanje taskova po statusima za korisnika
func CountUserTasksByStatus(userID string, token string) (map[string]int, error) {
	// Pozivamo task-service da preuzmemo sve taskove
	taskServiceEndpoint := fmt.Sprintf("http://task-service:8080/tasks")

	// Kreiranje HTTP GET zahteva
	req, err := http.NewRequest("GET", taskServiceEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Postavi Authorization header sa Bearer tokenom
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Slanje HTTP GET zahteva
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks from task-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task-service returned status: %d", resp.StatusCode)
	}

	// Čitamo telo odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from task-service: %v", err)
	}

	// Parsiramo JSON u listu taskova
	var tasks []models.Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks data: %v", err)
	}

	// Inicijalizujemo mapu za brojanje taskova po statusu
	statusCount := map[string]int{
		"pending":          0,
		"work in progress": 0,
		"done":             0,
	}

	// Brojanje taskova na kojima je userID dodat, po statusima
	for _, task := range tasks {
		for _, user := range task.Users {
			if user == userID {
				// Increment status count based on the task status
				switch task.Status {
				case "pending":
					statusCount["pending"]++
				case "work in progress":
					statusCount["work in progress"]++
				case "done":
					statusCount["done"]++
				}
				break
			}
		}
	}

	return statusCount, nil
}

// CheckProjectStatus - Proverava da li je projekat završen
func CheckProjectStatus(projectID string, token string) (bool, error) {
	endpoint := fmt.Sprintf("http://project-service:8080/projects/isActive/%s", projectID)

	// Kreiranje HTTP GET zahteva
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	// Postavi Authorization header sa Bearer tokenom
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Slanje HTTP GET zahteva
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to fetch project status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("project-service returned status: %d", resp.StatusCode)
	}

	// Provera JSON odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read project status response: %v", err)
	}

	fmt.Printf("Response body: %s\n", string(body))

	var response struct {
		Result bool `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return false, fmt.Errorf("failed to parse project status response: %v", err)
	}

	return response.Result, nil
}

// GetUserProjects - Dohvata sve projekte korisnika
func GetUserProjects(userID string, token string) ([]models.Project, error) {
	endpoint := fmt.Sprintf("http://project-service:8080/projects/user/%s", userID)

	// Kreiranje HTTP GET zahteva
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Postavi Authorization header sa Bearer tokenom
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Slanje HTTP GET zahteva
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("project-service returned status: %d", resp.StatusCode)
	}

	var projects []models.Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %v", err)
	}

	return projects, nil
}

// GetUserTasksAndProject - Funkcija koja vraća taskove korisnika i ime projekta
func GetUserTasksAndProject(userID string, token string) (map[string]interface{}, error) {
	// Pozivamo task-service da preuzmemo sve taskove
	taskServiceEndpoint := fmt.Sprintf("http://task-service:8080/tasks")

	// Kreiranje HTTP GET zahteva za taskove
	taskReq, err := http.NewRequest("GET", taskServiceEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for tasks: %v", err)
	}

	// Postavi Authorization header sa Bearer tokenom
	taskReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Slanje HTTP GET zahteva za taskove
	client := &http.Client{}
	taskResp, err := client.Do(taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks from task-service: %v", err)
	}
	defer taskResp.Body.Close()

	if taskResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task-service returned status: %d", taskResp.StatusCode)
	}

	// Čitamo telo odgovora za taskove
	taskBody, err := ioutil.ReadAll(taskResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from task-service: %v", err)
	}

	// Parsiramo JSON u listu taskova
	var tasks []models.Task
	if err := json.Unmarshal(taskBody, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks data: %v", err)
	}

	// Inicijalizujemo mapu za rezultat
	result := make(map[string]interface{})
	var userTasks []string
	projectMap := make(map[string][]string)

	// Za svaki task proveravamo da li je korisnik dodeljen tom tasku
	for _, task := range tasks {
		for _, user := range task.Users {
			if user == userID {
				// Dodajemo ime taska u listu ako je korisnik dodeljen
				userTasks = append(userTasks, task.Name)

				// Dodajemo task u odgovarajući projekat
				if task.Project_ID != "" {
					projectMap[task.Project_ID] = append(projectMap[task.Project_ID], task.Name)
				}
				break
			}
		}
	}

	// Ako korisnik ima taskove, pozivamo project-service za ime projekta
	if len(userTasks) > 0 {
		// Iteriramo kroz projekte i pozivamo project-service za ime svakog projekta
		var projectTitles []map[string]interface{}
		for projectID, taskNames := range projectMap {
			projectServiceEndpoint := fmt.Sprintf("http://project-service:8080/projects/%s", projectID)

			// Kreiranje HTTP GET zahteva za projekat
			projectReq, err := http.NewRequest("GET", projectServiceEndpoint, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request for project: %v", err)
			}

			// Postavi Authorization header sa Bearer tokenom
			projectReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			// Slanje HTTP GET zahteva za projekat
			projectResp, err := client.Do(projectReq)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch project data from project-service: %v", err)
			}
			defer projectResp.Body.Close()

			if projectResp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("project-service returned status: %d", projectResp.StatusCode)
			}

			// Čitamo telo odgovora za projekat
			projectBody, err := ioutil.ReadAll(projectResp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response from project-service: %v", err)
			}

			// Parsiramo JSON u strukturu projekta
			var project struct {
				Title string `json:"title"`
			}
			if err := json.Unmarshal(projectBody, &project); err != nil {
				return nil, fmt.Errorf("failed to parse project data: %v", err)
			}

			// Dodajemo projekat i njegove taskove u rezultat
			projectInfo := map[string]interface{}{
				"project": project.Title,
				"tasks":   taskNames,
			}
			projectTitles = append(projectTitles, projectInfo)
		}

		// Dodajemo sve projekte i taskove u rezultat
		result["projects"] = projectTitles
	}

	return result, nil
}

// RecordStatusChange - Zapisuje vreme provedeno u svakom statusu
func RecordStatusChange(taskID, previousStatus, newStatus string, timestamp time.Time) error {
	collection := db.Client.Database("testdb").Collection("analytics")
	// Pronađi postojeći dokument sa analizom
	var analytics models.TaskAnalytics
	err := collection.FindOne(context.TODO(), bson.M{"task_id": taskID}).Decode(&analytics)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Ako dokument ne postoji, kreiraj novi
			analytics = models.TaskAnalytics{
				TaskID:           taskID,
				StatusTimes:      map[string]int64{},
				LastStatusChange: timestamp,
			}
		} else {
			log.Printf("Failed to find task analytics: %v", err)
			return err
		}
	}

	// Izračunaj vreme provedeno u prethodnom statusu
	duration := timestamp.Sub(analytics.LastStatusChange).Seconds()
	if duration > 0 && previousStatus != "" {
		analytics.StatusTimes[previousStatus] += int64(duration)
	}

	// Ažuriraj poslednju promenu statusa i sačuvaj
	analytics.LastStatusChange = timestamp

	// Definiši opcije za upsert
	opts := options.Update().SetUpsert(true)

	// Ažuriraj dokument u bazi
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"task_id": taskID},
		bson.M{"$set": bson.M{
			"status_times":       analytics.StatusTimes,
			"last_status_change": analytics.LastStatusChange,
		}},
		opts,
	)
	if err != nil {
		log.Printf("Failed to update task analytics: %v", err)
		return err
	}

	return nil
}

// GetTaskAnalytics - Dohvata analitiku za određeni task
func GetTaskAnalytics(taskID string) (*models.TaskAnalytics, error) {
	collection := db.Client.Database("testdb").Collection("analytics")
	var analytics models.TaskAnalytics
	err := collection.FindOne(context.TODO(), bson.M{"task_id": taskID}).Decode(&analytics)
	if err != nil {
		return nil, err
	}
	return &analytics, nil
}

// Helper za opcije upserta
func mongoOptionsForUpsert() *options.UpdateOptions { // Ispravljeno sa "mongo.UpdateOptions"
	upsert := true
	return &options.UpdateOptions{Upsert: &upsert}
}

func GetUserTaskAnalytics(userID string, token string) ([]models.TaskAnalytics, error) {
	collection := db.Client.Database("testdb").Collection("analytics")

	// Pozivamo task-service da preuzmemo sve taskove
	taskServiceEndpoint := "http://task-service:8080/tasks"

	// Kreiranje HTTP GET zahteva za taskove
	req, err := http.NewRequest("GET", taskServiceEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for tasks: %v", err)
	}

	// Postavi Authorization header sa Bearer tokenom
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Slanje HTTP GET zahteva za taskove
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks from task-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task-service returned status: %d", resp.StatusCode)
	}

	// Čitamo telo odgovora za taskove
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from task-service: %v", err)
	}

	// Parsiramo JSON u listu taskova
	var tasks []models.Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks data: %v", err)
	}

	// Filtriramo taskove gde je userID u listi Users
	var userTasks []string
	for _, task := range tasks {
		for _, user := range task.Users {
			if user == userID {
				userTasks = append(userTasks, task.ID.Hex())
				break
			}
		}
	}

	// Ako nema taskova, vraćamo praznu listu
	if len(userTasks) == 0 {
		return []models.TaskAnalytics{}, nil
	}

	// Dohvatamo analitiku za sve taskove gde je userID zadužen
	var analyticsList []models.TaskAnalytics
	for _, taskID := range userTasks {
		var analytics models.TaskAnalytics
		err := collection.FindOne(context.TODO(), bson.M{"task_id": taskID}).Decode(&analytics)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// Ako nema analitike za task, samo ga preskočimo
				continue
			}
			return nil, fmt.Errorf("error fetching analytics for task %s: %v", taskID, err)
		}
		analyticsList = append(analyticsList, analytics)
	}

	return analyticsList, nil
}
