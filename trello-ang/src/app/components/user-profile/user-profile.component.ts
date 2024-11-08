import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AuthService } from 'src/app/services/auth.service';
import { UserService } from 'src/app/services/user.service';

@Component({
  selector: 'app-user-profile',
  templateUrl: './user-profile.component.html',
  styleUrls: ['./user-profile.component.css']
})
export class UserProfileComponent implements OnInit {
  userIcon = 'assets/user.png'; // Placeholder za ikonu korisnika
  user: any; // Korisnički podaci
  userId: string = ''; // ID korisnika koji dobijamo iz tokena

  // Dodajte promenljive za lozinke
  oldPassword: string = '';
  newPassword: string = '';
  confirmPassword: string = '';

  // Poruke za uspeh i grešku
  errorMessage: string = '';
  successMessage: string = '';

  constructor(private authService: AuthService, private userService: UserService) {}

  ngOnInit(): void {
    this.user = this.getUserInfoFromToken();
    if (this.user && this.user.id) {
      console.log("User ID from token:", this.user.id);

      // Poziv funkcije za dobijanje korisničkog profila
      this.userService.getUserById(this.user.id).subscribe({
        next: (data) => {
          this.user = data; // Sačuvaj podatke dobijene sa servera
          console.log("User data from server:", this.user);
        },
        error: (error) => {
          console.error('Error fetching user profile:', error);
        }
      });
    }
  }

  closeModalAndRefresh(){
    window.location.reload();
  }

  // Funkcija koja uzima korisničke informacije iz JWT tokena
  getUserInfoFromToken(): any {
    const token = localStorage.getItem('access_token');
    console.log("Token:", token);
    if (token) {
      try {
        // Podela tokena na tri dela
        const payloadBase64 = token.split('.')[1];
        // Dekodiranje Base64 stringa u JSON
        const payloadJson = atob(payloadBase64);
        // Parsiranje JSON stringa u objekat
        return JSON.parse(payloadJson);
      } catch (error) {
        console.error('Invalid token format:', error);
        return null;
      }
    }
    return null;
  }

  // Funkcija za slanje zahteva za promenu lozinke
  onSubmitChangePassword(): void {
    if (this.newPassword !== this.confirmPassword) {
      this.errorMessage = 'New password and confirm password do not match.';
      return;
    }

    const userId = this.user.id;
    const passwordData = {
      oldPassword: this.oldPassword,
      newPassword: this.newPassword,
      confirmPassword: this.confirmPassword
    };

    this.userService.changePassword(userId, passwordData).subscribe({
      next: (response) => {
        this.successMessage = 'Password changed successfully!';
        console.log('Uspeo')
        this.errorMessage = ''; // Resetovanje greške ako je uspešno
        //location.reload();
      },
      error: (error) => {
        this.errorMessage = 'Failed to change password. Please try again.';
        alert(this.errorMessage)
        this.successMessage = ''; // Resetovanje uspeha ako je greška
      }
    });
  }
}
