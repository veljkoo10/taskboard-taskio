import { Injectable } from '@angular/core';
import {HttpClient, HttpHeaders} from '@angular/common/http';
import {Observable, Subject} from 'rxjs';
import { catchError } from 'rxjs/operators';
import { Project } from "../model/project.model";
import { Task } from "../model/task.model";

@Injectable({
  providedIn: 'root'
})
export class ProjectService {
  private baseUrl = 'http://localhost/taskio/projects';
  private taskUrl = 'http://localhost:8082/tasks';
  private projectCreated = new Subject<Project>();

  constructor(private http: HttpClient) {}
  getProjectsByUser(userId: string, token: string): Observable<Project[]> {
    const headers = new HttpHeaders({
      'Authorization': `Bearer ${token}`
    });

    return this.http.get<Project[]>(`${this.baseUrl}/user/${userId}`, { headers });
  }
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
    return this.http.post<any>(`${this.taskUrl}/create/${projectId}`, task).pipe(
      catchError((error) => {
        if (error.status === 409) {
          alert('Task name must be unique!');
        }
        throw error;
      })
    );
  }

  getTasks(): Observable<any[]> {
    return this.http.get<any[]>(this.taskUrl);
  }

  addMemberToProject(projectId: string, userIds: string[]): Observable<any> {
    const url = `${this.baseUrl}/${projectId}/users`;
    return this.http.put<any>(url, { userIds }, { responseType: 'text' as 'json' });
  }
  get projectCreated$() {
    return this.projectCreated.asObservable();
  }

  notifyProjectCreated(project: Project) {
    this.projectCreated.next(project);
  }
}
