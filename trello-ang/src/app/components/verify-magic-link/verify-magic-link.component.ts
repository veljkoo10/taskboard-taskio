import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { AuthService } from '../../services/auth.service';
import { UserService } from '../../services/user.service';
import { CommonModule } from '@angular/common'; // Dodaj CommonModule za korišćenje ngIf, ngFor, itd.
import { NgModule } from '@angular/core';
import * as CryptoJS from 'crypto-js';

@Component({
  selector: 'app-verify-magic-link',
  templateUrl: './verify-magic-link.component.html'
})
export class VerifyMagicLinkComponent implements OnInit {
  token: string | null = null;
  errorMessage: string = '';
  successMessage: string = '';
  email: string = '';
  message: string = '';

  SECRET_KEY = 'my-secret-key-12345';



  constructor(
    private activatedRoute: ActivatedRoute,
    private authService: AuthService,
    private router: Router,
    private userService: UserService,
    private http: HttpClient,
  ) {}

  ngOnInit() {
    this.activatedRoute.queryParams.subscribe(params => {
      this.token = params['token'];
      console.log('Preuzet token iz URL-a:', this.token);

      if (this.token) {
        this.loginMagicLink(this.token);
      } else {
        this.errorMessage = 'Neispravan token.';
        console.error('Token nije pronađen u URL-u.');
      }
    });
  }
  loginMagicLink(token: string) {
    this.userService.loginMagicLink(token).subscribe(
      response => {
        console.log('Odgovor sa servera:', response);

        console.log('Dobijeni access_token:', response.access_token);
        console.log('Dobijeni role:', response.role);
        console.log('Dobijeni user_id:', response.user_id);

        if (response.user_id) {
          const encryptedUserId = CryptoJS.AES.encrypt(response.user_id, this.SECRET_KEY).toString();
          localStorage.setItem('user_id', encryptedUserId);
        } else {
          console.error('User ID nije dostavljen od servera!');
        }

        const encryptedRole = CryptoJS.AES.encrypt(response.role, this.SECRET_KEY).toString();
        //this.authService.saveToken(response.access_token);
        localStorage.setItem('role', encryptedRole);
        const encryptedToken = CryptoJS.AES.encrypt(response.access_token, this.SECRET_KEY).toString();
        localStorage.setItem('access_token', encryptedToken);

        console.log('Proveravam localStorage:');
        console.log('access_token:', localStorage.getItem('access_token'));
        console.log('role:', localStorage.getItem('role'));
        console.log('user_id:', localStorage.getItem('user_id'));

        setTimeout(() => {
          this.router.navigate(['/dashboard']);
        }, 100);
      },
      error => {
        this.errorMessage = 'Greška prilikom prijave. Pokušajte ponovo.';
        console.error('Greška prilikom prijave:', error);
      }
    );
  }

}
