package service

import (
	"analytics-service/models"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// CountUserTasks - Funkcija za brojanje taskova na kojima je korisnik
func CountUserTasks(userID string) (int, error) {
	// Pozivamo task-service da preuzmemo sve taskove
	taskServiceEndpoint := fmt.Sprintf("http://task-service:8080/tasks")
	resp, err := http.Get(taskServiceEndpoint)
	if err != nil {
		return 0, errors.New("failed to fetch tasks from task-service")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("task-service returned status: %d", resp.StatusCode)
	}

	// Čitamo telo odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.New("failed to read response from task-service")
	}

	// Parsiramo JSON u listu taskova
	var tasks []models.Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return 0, errors.New("failed to parse tasks data")
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
func CountUserTasksByStatus(userID string) (map[string]int, error) {
	// Pozivamo task-service da preuzmemo sve taskove
	taskServiceEndpoint := fmt.Sprintf("http://task-service:8080/tasks")
	resp, err := http.Get(taskServiceEndpoint)
	if err != nil {
		return nil, errors.New("failed to fetch tasks from task-service")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task-service returned status: %d", resp.StatusCode)
	}

	// Čitamo telo odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response from task-service")
	}

	// Parsiramo JSON u listu taskova
	var tasks []models.Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, errors.New("failed to parse tasks data")
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
func CheckProjectStatus(projectID string) (bool, error) {
	endpoint := fmt.Sprintf("http://project-service:8080/projects/isActive/%s", projectID)
	resp, err := http.Get(endpoint)
	if err != nil {
		return false, errors.New("failed to fetch project status")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("project-service returned status: %d", resp.StatusCode)
	}

	// Provera JSON odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, errors.New("failed to read project status response")
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
func GetUserProjects(userID string) ([]models.Project, error) {
	endpoint := fmt.Sprintf("http://project-service:8080/projects/user/%s", userID)
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, errors.New("failed to fetch projects")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("project-service returned status: %d", resp.StatusCode)
	}

	var projects []models.Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, errors.New("failed to decode projects")
	}

	return projects, nil
}

// GetUserTasksAndProject - Funkcija koja vraća taskove korisnika i ime projekta
func GetUserTasksAndProject(userID string) (map[string]interface{}, error) {
	// Pozivamo task-service da preuzmemo sve taskove
	taskServiceEndpoint := fmt.Sprintf("http://task-service:8080/tasks")
	resp, err := http.Get(taskServiceEndpoint)
	if err != nil {
		return nil, errors.New("failed to fetch tasks from task-service")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task-service returned status: %d", resp.StatusCode)
	}

	// Čitamo telo odgovora
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response from task-service")
	}

	// Parsiramo JSON u listu taskova
	var tasks []models.Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, errors.New("failed to parse tasks data")
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
			projectResp, err := http.Get(projectServiceEndpoint)
			if err != nil {
				return nil, errors.New("failed to fetch project data from project-service")
			}
			defer projectResp.Body.Close()

			if projectResp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("project-service returned status: %d", projectResp.StatusCode)
			}

			// Čitamo telo odgovora za projekat
			projectBody, err := ioutil.ReadAll(projectResp.Body)
			if err != nil {
				return nil, errors.New("failed to read response from project-service")
			}

			// Parsiramo JSON u strukturu projekta
			var project struct {
				Title string `json:"title"`
			}
			if err := json.Unmarshal(projectBody, &project); err != nil {
				return nil, errors.New("failed to parse project data")
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
