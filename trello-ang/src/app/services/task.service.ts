import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { catchError } from 'rxjs/operators';
import { Task } from "../model/task.model";
import { map } from 'rxjs/operators';
import {Observable, Subject, throwError} from 'rxjs';


@Injectable({
  providedIn: 'root'
})
export class TaskService {
  private taskUrl = 'https://localhost/taskio/tasks'; // Base URL for task endpoints
  private taskUrl2 = 'https://localhost/taskio/tasks'; // Base URL for task endpoints
  private workflow = 'https://localhost/taskio/workflow'
  private taskUrl3 = 'http://localhost:8082/tasks';

  constructor(private http: HttpClient) {}

  // Fetch all tasks
  getTasks(): Observable<Task[]> {
    return this.http.get<Task[]>(`${this.taskUrl}`).pipe(
      catchError((error) => {
        console.error('Error fetching tasks:', error);
        throw error;
      })
    );
  }

  updateTaskStatus(taskId: string, status: string): Observable<any> {
    return this.http.put<any>(`${this.taskUrl}/${taskId}`, { status }).pipe(
      catchError((error) => {
        console.error('Error updating task status:', error);
        throw error;
      })
    );
  }


  // Create a new task for a specific project
  createTask(projectId: string, task: { name: string; description: string }): Observable<Task> {
    console.log(`ID PROJEKTA U SERVISU JE: ${projectId}`);
    return this.http.post<Task>(`${this.taskUrl}/create/${projectId}`, task).pipe(
      catchError((error) => {
        if (error.status === 409) {
          alert('A task with that name already exists!');
        }
        console.error('Error creating task:', error);
        throw error;
      })
    );
  }

  addUserToTask(taskId: string, userId: string): Observable<any> {
    // Log the request to see the taskId and userId being passed
    console.log(`Adding user ${userId} to task ${taskId}`);

    // Send the PUT request with taskId and userId as URL parameters
    return this.http.put<any>(`${this.taskUrl}/${taskId}/users/${userId}`, {}).pipe(
      catchError((error) => {
        console.error('Error adding user to task:', error);
        throw error;
      })
    );
  }

  getUsersForTask(taskId: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.taskUrl}/${taskId}/users`);
  }

  getTaskById(taskId: string): Observable<{ id: string; name: string }> {
    return this.http.get<{ id: string; name: string }>(`${this.taskUrl}/${taskId}`);
  }

  // Remove a user from a task
  removeUserFromTask(taskId: string, userId: string): Observable<any> {
    return this.http.delete<any>(`${this.taskUrl}/${taskId}/users/${userId}`).pipe(
      catchError((error) => {
        console.error('Error removing user from task:', error);
        throw error;
      })
    );
  }

  getTasksByProjectId(projectId: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.taskUrl2}/projects/${projectId}/tasks`);
  }

  isUserOnTask(taskId: string, userId: string): Observable<boolean> {
    const url = `${this.taskUrl}/${taskId}/member-of/${userId}`;
    console.log(url);

    return this.http.get<{ isMember: boolean }>(url).pipe(
      // Ekstraktovanje vrednosti result direktno
      map(response => response.isMember),  // direktno dobijanje true/false iz odgovora
      catchError((error) => {
        console.error('Error checking project active status:', error);
        return throwError(error);
      })
    );
  }
  uploadFile(formData: FormData): Observable<any> {

    return this.http.post(`${this.taskUrl}/upload`, formData);
  }
  downloadFile(taskId: string, fileNamee: string): Observable<Blob> {
    const fileName = encodeURIComponent(fileNamee); // Enkodiranje imena fajla
    const url = `${this.taskUrl3}/${taskId}/download/${fileName}`;  // Prilagodi URL
    return this.http.get(url, { responseType: 'blob' });
  }
  // Create a workflow by assigning dependencies to a task
  createWorkflow(taskId: string, dependencyTasks: string[], projectId: string): Observable<any> {
    const url = `${this.workflow}/createWorkflow`;

    const payload = {
      task_id: taskId,
      dependency_task: dependencyTasks,
      project_id: projectId
    };

    return this.http.post<any>(url, payload).pipe(
      catchError((error) => {
        console.error('Error creating workflow:', error);
        throw error;
      })
    );
  }

  getTaskFiles(taskId: string): Observable<{ fileName: string, content: string }[]> {
    return this.http.get<{ fileName: string, content: string }[]>(`${this.taskUrl}/files/${taskId}`);
  }
  getAllWorkflows() {
    return this.http.get<any[]>(`${this.workflow}/getWorkflows`);
  }

  getWorkflowByProjectId(projectId: string): Observable<any[]> {
      const url = `${this.workflow}/project/${projectId}`;
      return this.http.get<any[]>(url).pipe(
        catchError((error) => {
          console.error(`Error fetching workflows for project ID ${projectId}:`, error);
          return throwError(error);
        }));
  }


}


