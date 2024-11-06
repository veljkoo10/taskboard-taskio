import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { User } from '../../model/user.model';
import { FormsModule } from "@angular/forms";
import { AuthService } from "../../services/auth.service";
import { filter, map, switchMap } from "rxjs";

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [
    FormsModule,
  ],
  templateUrl: './register.component.html',
  styleUrls: ['./register.component.css']
})
export class RegisterComponent {
  userIcon = 'assets/user2.png';
  emailIcon = 'assets/email.png';
  padlockIcon = 'assets/padlock.png';
  settingIcon = 'assets/setting.png';
  backgroundIcon = 'assets/Login-rafiki.svg';
  usernameIcon = 'assets/id-card.png';
  user: User = new User('', '', '', '', '', '');
  passwordError: string = '';

  constructor(private router: Router, private authService: AuthService) {}

  navigateToLogin() {
    this.router.navigate(['/login']);
  }

  isFormValid(): boolean {
    return (
      this.user.username.trim() !== '' &&
      this.user.name.trim() !== '' &&
      this.user.surname.trim() !== '' &&
      this.user.email.trim() !== '' &&
      this.user.password.trim() !== '' &&
      this.user.role.trim() !== '' &&
      this.isPasswordValid(this.user.password)
    );
  }

  validateEmail(email: string): boolean {
    return email.endsWith('@gmail.com');
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

  checkEmailExists() {
    return this.authService.checkEmailExists(this.user.email).pipe(
      map(response => response.exists)
    );
  }

  register() {
    if (!this.isFormValid()) {
      if (this.passwordError) {
        alert(this.passwordError);
      } else {
        alert('All fields must be filled!');
      }
      return;
    }
    if (!this.validateEmail(this.user.email)) {
      alert('Email must be in @gmail.com format!');
      return;
    }

    this.checkEmailExists().pipe(
      map(exists => {
        if (exists) {
          alert('Email already exists! Please try another email.');
          return false;
        }
        return true;
      }),
      switchMap(emailExists => {
        if (!emailExists) return [false];
        return this.authService.checkUsernameExists(this.user.username).pipe(
          map(usernameExists => {
            if (usernameExists.exists) {
              alert('Username already exists! Please try another username.');
              return false;
            }
            return true;
          })
        );
      }),
      filter(exists => exists)
    ).subscribe(shouldRegister => {
      if (shouldRegister) {
        this.authService.register(this.user).subscribe({
          next: (response) => {
            console.log('Registration successful:', response);
            alert('You are successfully registered! Check your email to confirm your account.');
            this.user = new User('', '', '', '', '', '');
            this.navigateToLogin();
          },
          error: (error) => {
            console.error('Registration failed:', error);
            alert('The data is not correct.');
          }
        });
      }
    });
  }
}
