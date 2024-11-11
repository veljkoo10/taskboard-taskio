import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import {Project} from "../model/project.model";
import {Task} from "../model/task.model"

@Injectable({
  providedIn: 'root'
})
export class ProjectService {
  private baseUrl = 'http://localhost/taskio/projects';
  private taskUrl = 'http://localhost:8082/tasks'

  constructor(private http: HttpClient) {}

  createProject(managerId: string, project: Project): Observable<Project> {
    return this.http.post<Project>(`${this.baseUrl}/create/${managerId}`, project);
  }

  getProjects(): Observable<Project[]> {
    return this.http.get<Project[]>(this.baseUrl);
  }
  checkProjectByTitle(title: string): Observable<string> {
    return this.http.post<string>(`${this.baseUrl}/title`, { title: title }, { responseType: 'text' as 'json' });
  }

  createTask(projectId: string, task: { name: string, description: string }): Observable<Task> {
    return this.http.post<any>(`${this.taskUrl}/create/${projectId}`, task);
  }

  getTasks(): Observable<any[]> {
    return this.http.get<any[]>(this.taskUrl);
  }


}


