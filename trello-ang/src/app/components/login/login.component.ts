import { Component } from '@angular/core';
import {Router} from "@angular/router";
import {AuthService} from "../../services/auth.service";

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.css']
})
export class LoginComponent {
  imagePath: string = 'assets/loginSlika.svg';
  resetEmail: string = '';
  resetMessage: string = '';
  constructor(private router: Router,private authService: AuthService) {}
  navigateToRegister() {
    this.router.navigate(['/register']);
  }
  isEmailValid(email: string): boolean {
    return email.endsWith('@gmail.com');
  }
  checkEmailAndResetPassword() {
    if (!this.resetEmail) { // Proverava da li je polje prazno
      this.resetMessage = 'Email mora biti popunjen.';
      return;
    }
    if (!this.isEmailValid(this.resetEmail)) {
      this.resetMessage = 'Email mora biti u formatu @gmail.com';
      return;
    }

    // Provera da li je email aktivan
    this.authService.checkUserActive(this.resetEmail).subscribe(
      (response) => {
        if (response.active) {
          // Ako je email aktivan, šaljemo zahtev za reset lozinke
          this.authService.requestPasswordReset(this.resetEmail).subscribe(
            () => {
              this.resetMessage = 'Link za resetovanje lozinke je poslat na vašu email adresu.';
            },
            (error) => {
              this.resetMessage = 'Došlo je do greške prilikom slanja linka za reset lozinke. Pokušajte ponovo.';
            }
          );
        } else {
          // Ako email nije aktivan, prikazujemo poruku
          this.resetMessage = 'Email nije aktivan.';
        }
      },
      (error) => {
        this.resetMessage = 'Došlo je do greške prilikom provere statusa korisnika.';
      }
    );
  }

}
