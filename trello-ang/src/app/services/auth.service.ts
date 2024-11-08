import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { catchError, Observable, throwError } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly baseUrl = 'http://localhost/taskio';
  private readonly registerUrl = `${this.baseUrl}/register`;
  private readonly loginUrl = `${this.baseUrl}/login`;
  private readonly resetPasswordUrl = `${this.baseUrl}/reset-password`;
   private readonly profileUrl = `${this.baseUrl}/profile`; 

  constructor(private http: HttpClient) {}

  register(user: any): Observable<any> {
    return this.http.post<any>(this.registerUrl, user, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    });
  }

  login(credentials: { username: string, password: string }): Observable<any> {
    return this.http.post<any>(this.loginUrl, credentials, {
      headers: new HttpHeaders({
        'Content-Type': 'application/json'
      })
    }).pipe(
      catchError(error => {
        console.error('Login error:', error);
        return throwError('Login failed. Please try again.');
      })
    );
  }
  resetPassword(email: string): Observable<any> { // Nova metoda za reset lozinke
    return this.http.post<any>(this.resetPasswordUrl, { email }, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    }).pipe(
      catchError(error => {
        console.error('Error sending reset password email:', error);
        return throwError('Failed to send reset password email. Please try again.');
      })
    );
  }
  
  logout(){
    localStorage.clear()
  }



  getProfileData(): Observable<any> {
    const token = localStorage.getItem('authToken');  // Preuzmi token iz localStorage

    if (!token) {
      throw new Error('User not authenticated');
    }

    // Kreiraj header sa tokenom
    const headers = new HttpHeaders({
      'Authorization': `Bearer ${token}`
    });

    // Pozovi API za korisničke podatke
    return this.http.get<any>(this.profileUrl, { headers });
  }
}