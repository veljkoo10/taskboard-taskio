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
    if (!this.resetEmail) {
      this.resetMessage = 'Email must be filled out.';
      return;
    }
    if (!this.isEmailValid(this.resetEmail)) {
      this.resetMessage = 'Email must be in @gmail.com format';
      return;
    }

    this.authService.checkUserActive(this.resetEmail).subscribe(
      (response) => {
        if (response.active) {
          this.authService.requestPasswordReset(this.resetEmail).subscribe(
            () => {
              this.resetMessage = 'A password reset link has been sent to your email address.';
            },
            (error) => {
              this.resetMessage = 'There was an error sending the password reset link. Try again.';
            }
          );
        } else {
          this.resetMessage = 'Email is not active.';
        }
      },
      (error) => {
        this.resetMessage = 'An error occurred while checking the user\'s status.';
      }
    );
  }

}
