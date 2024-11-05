import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import {Project} from "../model/project.model";

@Injectable({
  providedIn: 'root'
})
export class ProjectService {
  private baseUrl = 'http://localhost:8081/projects'; // Update to the correct port

  constructor(private http: HttpClient) {}

  createProject(project: Project): Observable<Project> {
    return this.http.post<Project>(`${this.baseUrl}/create`, project); // This sends the POST request
  }
  
  getProjects(): Observable<Project[]> {
    return this.http.get<Project[]>(this.baseUrl);
  }
}


