import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { Task } from "../model/task.model";

@Injectable({
  providedIn: 'root'
})
export class TaskService {
  private taskUrl = 'http://localhost:8082/tasks'; // Base URL for task endpoints

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
  
  

  // Remove a user from a task
  removeUserFromTask(taskId: string, userId: string): Observable<any> {
    return this.http.delete<any>(`${this.taskUrl}/${taskId}/users/${userId}`).pipe(
      catchError((error) => {
        console.error('Error removing user from task:', error);
        throw error;
      })
    );
  }
}
