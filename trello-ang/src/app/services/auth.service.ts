import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { catchError, Observable, throwError } from 'rxjs';
import * as CryptoJS from 'crypto-js';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly baseUrl = 'https://localhost/taskio';
  private readonly registerUrl = `${this.baseUrl}/register`;
  private readonly loginUrl = `${this.baseUrl}/login`;
  private readonly resetPasswordUrl = `${this.baseUrl}/reset-password`;
  private readonly profileUrl = `${this.baseUrl}/profile`;
  private readonly verifyUrl = `${this.baseUrl}/verify-magic-link`;
  private readonly magicUrl = `${this.baseUrl}/send-magic-link`;

  SECRET_KEY = 'my-secret-key-12345';

  constructor(private http: HttpClient) {}

  // Helper funkcija za sanitizaciju unosa
  private sanitizeInput(input: string): string {
    return input.replace(/<[^>]*>/g, ''); // Uklanja HTML tagove
  }

  // Registracija korisnika sa sanitizacijom
  register(user: any): Observable<any> {
    console.log(user)
    const sanitizedUser = {
      username: this.sanitizeInput(user.username),
      email: this.sanitizeInput(user.email),
      password: user.password, // Lozinka se ne sanitizuje zbog mogućnosti specijalnih karaktera
      name: user.name,
      surname: user.surname,
      role: user.role,
      id: user.id,
      isActive: user.isActive
    };

    return this.http.post<any>(this.registerUrl, sanitizedUser, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    }).pipe(
      catchError(error => {
        console.error('Registration error:', error);
        return throwError('Registration failed. Please try again.');
      })
    );
  }

  // Funkcija za prijavu korisnika sa sanitizacijom
  login(credentials: { username: string, password: string }): Observable<any> {
    const sanitizedCredentials = {
      username: this.sanitizeInput(credentials.username),
      password: credentials.password
    };

    return this.http.post<any>(this.loginUrl, sanitizedCredentials, {
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

  // Čuvanje tokena u localStorage sa dodatnim koracima
  saveToken(token: string): void {
    // Preporučuje se enkripcija tokena pre čuvanja
    localStorage.setItem('access_token', token);
  }

  // Resetovanje lozinke sa sanitizacijom email-a
  resetPassword(email: string): Observable<any> {
    const sanitizedEmail = this.sanitizeInput(email);

    return this.http.post<any>(this.resetPasswordUrl, { email: sanitizedEmail }, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    }).pipe(
      catchError(error => {
        console.error('Error sending reset password email:', error);
        return throwError('Failed to send reset password email. Please try again.');
      })
    );
  }

  // Slanje magic linka sa sanitizacijom
  sendMagicLink(email: string, username: string): Observable<any> {
    const sanitizedEmail = this.sanitizeInput(email);
    const sanitizedUsername = this.sanitizeInput(username);

    return this.http.post<any>(this.magicUrl, { email: sanitizedEmail, username: sanitizedUsername }, {
      headers: new HttpHeaders({'Content-Type': 'application/json'})
    }).pipe(
      catchError(error => {
        console.error('Error sending magic link email:', error);
        return throwError('Failed to send magic link email. Please try again.');
      })
    );
  }

  // Provera da li je korisnik autentifikovan
  isAuthenticated(): boolean {
    return !!this.getDecryptedData('access_token');
  }

  // Logout korisnika
  logout(): void {
    localStorage.removeItem('access_token');
    localStorage.removeItem('role');
    localStorage.removeItem('user_id');
    localStorage.removeItem('_grecaptcha');
  }

  // Funkcija za dekriptovanje podataka iz localStorage
getDecryptedData(key: string): string {
  const encryptedData = localStorage.getItem(key);
  if (encryptedData) {
    const bytes = CryptoJS.AES.decrypt(encryptedData, this.SECRET_KEY);
    return bytes.toString(CryptoJS.enc.Utf8); // Dekodira u originalni string
  }
  return '';
}
}
