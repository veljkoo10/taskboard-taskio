import { Injectable } from '@angular/core';
import {HttpClient, HttpHeaders} from '@angular/common/http';
import {Observable, Subject, throwError} from 'rxjs';
import { catchError } from 'rxjs/operators';
import { Project } from "../model/project.model";
import { Task } from "../model/task.model";
import { map } from 'rxjs/operators';
import { lastValueFrom } from 'rxjs';


@Injectable({
  providedIn: 'root'
})
export class ProjectService {
  private baseUrl = 'https://localhost/taskio/projects';
  private taskUrl = 'https://localhost/taskio/tasks';
  private projectCreated = new Subject<Project>();
  private newProject = {}

  constructor(private http: HttpClient) {}
  getUsersForProject(projectId: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/${projectId}/users`);
  }
  getProjectIDByTitle(title: string): Observable<string> {
    return this.http.post<string>(`${this.baseUrl}/title/id`, { title })
      .pipe(
        catchError((error) => {
          console.error('Error fetching project ID:', error);
          throw error;
        })
      );
  }
  getProjectsByUser(userId: string, token: string): Observable<Project[]> {
    const headers = new HttpHeaders({
      'Authorization': `Bearer ${token}`
    });

    return this.http.get<Project[]>(`${this.baseUrl}/user/${userId}`, { headers });
  }
  createProject(managerId: string, project: Project): Observable<Project> {
    console.log(project)
    this.newProject = project
    return this.http.post<Project>(`${this.baseUrl}/create/${managerId}`, project);
  }

  getNewProject(){
    return this.newProject
  }

  getProjects(): Observable<Project[]> {
    return this.http.get<Project[]>(this.baseUrl);
  }

  checkProjectByTitle(title: string, managerId: string): Observable<string> {
    console.log('Manager ID:', managerId); // Provera ID-a menadžera
    console.log('Title:', title);         // Provera naslova projekta

    const url = `${this.baseUrl}/title/${managerId}`; // Dodaj managerId u URL

    return this.http.post<string>(
      url,
      { title },                          // Telo zahteva sadrži samo title
      { responseType: 'text' as 'json' } // Specifikacija tipa odgovora
    );
  }


  createTask(projectId: string, task: { name: string, description: string }): Observable<Task> {
    console.log('Project ID u servisu:', projectId);
    return this.http.post<any>(`${this.taskUrl}/create/${projectId}`, task).pipe(
    );
  }




  getTasks(): Observable<any[]> {
    return this.http.get<any[]>(this.taskUrl);
  }

  addMemberToProject(projectId: string, userIds: string[]): Observable<any> {
    const url = `${this.baseUrl}/${projectId}/add-users`;
    return this.http.put<any>(url, { userIds }, { responseType: 'text' as 'json' });
  }

  removeMemberToProject(projectId: string, userIds: string[]): Observable<any> {
    const url = `${this.baseUrl}/${projectId}/remove-users`;
    return this.http.put<any>(url, { userIds }, { responseType: 'text' as 'json' });
  }

  get projectCreated$() {
    return this.projectCreated.asObservable();
  }

  notifyProjectCreated(project: Project) {
    this.projectCreated.next(project);
  }

  getPeojectById(projectId: string): Observable<any[]> {
    const url = `${this.baseUrl}/${projectId}`;
    return this.http.get<any>(url).pipe(
      catchError(error => {
        console.error('Error fetching user profile:', error);
        return throwError(error);
      })
    );
  }

  isProjectActive(projectId: string): Observable<boolean> {
    const url = `${this.baseUrl}/isActive/${projectId}`;
    console.log(url);

    return this.http.get<{ result: boolean }>(url).pipe(
      // Ekstraktovanje vrednosti result direktno
      map(response => response.result),  // direktno dobijanje true/false iz odgovora
      catchError((error) => {
        console.error('Error checking project active status:', error);
        return throwError(error);
      })
    );
  }

}
