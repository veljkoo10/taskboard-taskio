import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import {catchError, Observable, throwError} from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly baseUrl = 'http://localhost/taskio';
  private readonly registerUrl = `${this.baseUrl}/register`;
  private readonly checkEmailUrl = `${this.baseUrl}/check-email`;
  private readonly resetPasswordUrl = `${this.baseUrl}/reset-password`;

  constructor(private http: HttpClient) {}

  register(user: any): Observable<any> {
    return this.http.post<any>(this.registerUrl, user, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    });
  }

  checkUsernameExists(username: string): Observable<{ exists: boolean }> {
    return this.http.get<{ exists: boolean }>(`${this.baseUrl}/check-username?username=${username}`);
  }

  checkEmailExists(email: string): Observable<{ exists: boolean }> {
    return this.http.get<{ exists: boolean }>(`${this.checkEmailUrl}?email=${email}`);
  }

  requestPasswordReset(email: string): Observable<any> {
    return this.http.post<any>(this.resetPasswordUrl, { email }, {
      headers: new HttpHeaders({
        'Content-Type': 'application/json'
      })
    });
  }
  checkUserActive(email: string): Observable<{ active: boolean }> {
    return this.http.get<{ active: boolean }>(`${this.baseUrl}/api/check-user-active?email=${email}`)
      .pipe(
        catchError(error => {
          console.error('Error checking user active status:', error);
          return throwError(error);
        })
      );
  }



}
