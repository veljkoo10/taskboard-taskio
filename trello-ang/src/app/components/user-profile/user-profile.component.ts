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
  userIcon = 'assets/user.png';
  user: any;
  userId: string = '';

  oldPassword: string = '';
  newPassword: string = '';
  confirmPassword: string = '';

  errorMessage: string = '';
  successMessage: string = '';
  passwordError: string = '';

  constructor(private authService: AuthService, private userService: UserService) {}

  ngOnInit(): void {
    this.user = this.getUserInfoFromToken();
    if (this.user && this.user.id) {
      console.log("User ID from token:", this.user.id);

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

  getUserInfoFromToken(): any {
    const token = localStorage.getItem('access_token');
    console.log("Token:", token);
    if (token) {
      try {
        const payloadBase64 = token.split('.')[1];
        const payloadJson = atob(payloadBase64);
        return JSON.parse(payloadJson);
      } catch (error) {
        console.error('Invalid token format:', error);
        return null;
      }
    }
    return null;
  }

  isPasswordValid(password: string): boolean {
    this.passwordError = '';

    if (password.length < 8) {
      this.passwordError = 'Password must have at least 8 characters.';
      return false;
    }
    if (!/[A-Z]/.test(password)) {
      this.passwordError = 'Password must have at least one capital letter.';
      return false;
    }
    if (!/[a-z]/.test(password)) {
      this.passwordError = 'Password must have at least one lowercase letter.';
      return false;
    }
    if (!/[0-9]/.test(password)) {
      this.passwordError = 'The password must have at least one number.';
      return false;
    }
    if (!/[!@#~$%^&*(),.?":{}|<>]/.test(password)) {
      this.passwordError = 'Password must have at least one special character.';
      return false;
    }

    return true;
  }


  onSubmitChangePassword(): void {
    if (!this.oldPassword || !this.newPassword || !this.confirmPassword) {
      this.errorMessage = 'All fields must be filled out.';
      alert(this.errorMessage);
      return;
    }

    if (this.newPassword !== this.confirmPassword) {
      this.errorMessage = 'New password and confirm password do not match.';
      alert(this.errorMessage);
      return;
    }

    if (this.isPasswordValid(this.newPassword)) {
      const userId = this.user.id;
      const passwordData = {
        oldPassword: this.oldPassword,
        newPassword: this.newPassword,
        confirmPassword: this.confirmPassword
      };

      this.userService.changePassword(userId, passwordData).subscribe({
        next: (response) => {
          this.successMessage = 'Password changed successfully!';
          console.log('Success');
          this.errorMessage = '';
        },
        error: (error) => {
          this.errorMessage = 'Wrong old password.';
          alert(this.errorMessage);
          this.successMessage = '';
        }
      });
    }
    if (this.passwordError) {
      alert(this.passwordError);
    }
  }

}
