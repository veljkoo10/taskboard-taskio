import { Component } from '@angular/core';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-forgot-password',
  templateUrl: './forgot-password.component.html',
  styleUrls: ['./forgot-password.component.css']
})
export class ForgotPasswordComponent {
  email: string = '';
  message: string = '';

  constructor(private authService: AuthService) {}

  sendResetPasswordEmail() {
    this.authService.resetPassword(this.email).subscribe({
      next: () => {
        this.message = 'Password reset email sent successfully.';
      },
      error: (error) => {
        console.error('Error sending reset email:', error);
        this.message = 'Failed to send password reset email. Please try again.';
      }
    });
  }
}
