import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-magic-link',
  templateUrl: './magic-login.component.html',
  styleUrls: ['./magic-login.component.css']
})
export class MagicLinkComponent {

  email: string = '';
  message: string = '';

  constructor(
    private authService: AuthService,
    private router: Router,
    private http: HttpClient
  ) {}

  sendMagicLink() {
    if (!this.email) {
      this.message = 'Molimo vas da unesete email adresu.';
      return;
    }

    this.authService.sendMagicLink(this.email).subscribe({
      next: (response) => {
        this.message = 'Magic link je uspešno poslat na vašu email adresu. Proverite svoj inbox.';
        console.log('Magic link poslat:', response);
      },
      error: (error) => {
        console.error('Greška prilikom slanja magic link-a:', error);
        this.message = 'Došlo je do greške pri slanju magic link-a. Pokušajte ponovo.';
      }
    });
  }
}
