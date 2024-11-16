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
  private readonly verifyUrl = `${this.baseUrl}/verify-magic-link`;
  private readonly magicUrl = `${this.baseUrl}/send-magic-link`;

  constructor(private http: HttpClient) {}

  // Registracija korisnika
  register(user: any): Observable<any> {
    return this.http.post<any>(this.registerUrl, user, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    });
  }

  // Funkcija za prijavu korisnika i čuvanje tokena
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

  // Čuvanje tokena u localStorage
  saveToken(token: string): void {
    localStorage.setItem('access_token', token);
  }

  // Resetovanje lozinke
  resetPassword(email: string): Observable<any> {
    return this.http.post<any>(this.resetPasswordUrl, { email }, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    }).pipe(
      catchError(error => {
        console.error('Error sending reset password email:', error);
        return throwError('Failed to send reset password email. Please try again.');
      })
    );
  }

  // Odlazak korisnika (logout)
  logout(): void {
    localStorage.removeItem('access_token');
  }

  sendMagicLink(email: string, username: string): Observable<any> {
    return this.http.post<any>(this.magicUrl, { email, username }, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    }).pipe(
      catchError(error => {
        console.error('Error sending magic link email:', error);
        return throwError('Failed to send magic link email. Please try again.');
      })
    );
  }

  isAuthenticated(): boolean {
    return !!localStorage.getItem('authToken');
  }
}
