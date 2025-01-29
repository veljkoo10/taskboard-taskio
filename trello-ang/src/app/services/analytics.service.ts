import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { catchError, Observable, throwError } from 'rxjs';

@Injectable({
    providedIn: 'root'
  })

export class AnalyticsService{
    private baseUrl = 'https://localhost/taskio/analytics'; // Osnovni URL

    constructor(private http: HttpClient) {}


    getUserTaskCount(userId: string): Observable<{ task_count: number }> {
        const url = `${this.baseUrl}/countusers/${userId}`;
    
        // Opcioni zaglavlja, ako treba da dodaš token ili drugi header
        const headers = new HttpHeaders({
          'Content-Type': 'application/json',
        });
    
        return this.http
          .get<{ task_count: number }>(url, { headers })
          .pipe(
            catchError((error) => {
              console.error('Error fetching task count:', error);
              return throwError(() => new Error('Failed to fetch task count.'));
            })
          );
    }

    getUserTaskStatusCount(userId: string): Observable<{ done: number; pending: number; 'work in progress': number }> {
        const url = `${this.baseUrl}/countusersbystatus/${userId}`;
        return this.http.get<{ done: number; pending: number; 'work in progress': number }>(url).pipe(
          catchError((error) => {
            console.error('Error fetching task count by status:', error);
            return throwError(() => new Error('Failed to fetch task count by status'));
          })
        );
      }

      // Funkcija koja šalje HTTP zahtev za projekte i zadatke korisnika
  getUserTaskProject(userId: string): Observable<any> {
    const url = `https://localhost/taskio/analytics/usertaskproject/${userId}`;
    return this.http.get<any>(url);
  }

  getProjectCompletionStatuses(userId: string) {
    const url = `https://localhost/taskio/analytics/project-completion-ontime/${userId}`;
    return this.http.get<any[]>(url).pipe(
      catchError((error) => {
        console.error('Error fetching project completion statuses:', error);
        return throwError(() => error);
      })
    );
  }


}