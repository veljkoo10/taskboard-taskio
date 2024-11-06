import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AuthService } from '../../services/auth.service';  // Putanja do vašeg servisa

@Component({
  selector: 'app-user-profile',
  templateUrl: './user-profile.component.html',
  styleUrls: ['./user-profile.component.css']
})
export class UserProfileComponent implements OnInit {

  userId: string | undefined;  // Korisnički ID
  user: any = {};  // Podaci o korisniku
  errorMessage: string = '';  // Ako dođe do greške

  constructor(
    private authService: AuthService,
    private route: ActivatedRoute
  ) { }

  ngOnInit(): void {
    // Čitanje 'id' parametra iz URL-a
    this.route.params.subscribe(params => {
      this.userId = params['id'];  // `id` iz URL-a
      console.log(this.userId)
      if (this.userId) {
        this.getUserProfile();  // Pozivamo servis da učitamo profil
      }
    });
  }

  // Metoda koja poziva servis za učitavanje podataka o korisniku
  getUserProfile(): void {
    if (this.userId) {
      this.authService.getUserById(this.userId).subscribe(
        (data) => {
          this.user = data;  // Postavljanje podataka o korisniku
        },
        (error) => {
          this.errorMessage = 'Error fetching user profile.';
          console.error('Error fetching user profile:', error);
        }
      );
    }
  }
}
