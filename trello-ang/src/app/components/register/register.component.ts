import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { User } from '../../model/user.model';
import { FormsModule } from "@angular/forms";
import { AuthService } from "../../services/auth.service";
import { filter, map, switchMap } from "rxjs";
import { UserService } from "../../services/user.service";
import {NgClass, NgIf} from "@angular/common";

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [
    FormsModule,
    NgIf,
    NgClass
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
  user: User = new User('', '', '', '', '', '', '');
  passwordError: string = '';
  modalTitle: string = '';
  modalMessage: string = '';
  isModalVisible: boolean = false;
  onCloseCallback: (() => void) | null = null;
  modalType: string = 'success';

  constructor(private router: Router, private authService: AuthService, private userService: UserService) {}

  navigateToLogin() {
    this.router.navigate(['/login']);
  }

  showModal(title: string, message: string, type: string = 'success', onClose?: () => void) {
    this.modalTitle = title;
    this.modalMessage = message;
    this.modalType = type;
    this.isModalVisible = true;
    this.onCloseCallback = onClose || null;
  }

  closeModal() {
    this.isModalVisible = false;
    if (this.onCloseCallback) {
      this.onCloseCallback();
      this.onCloseCallback = null;
    }
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
    return this.userService.checkEmailExists(this.user.email).pipe(
      map(response => response.exists)
    );
  }

  register() {
    if (!this.isFormValid()) {
      if (this.passwordError) {
        this.showModal('Password Error', this.passwordError, 'error');
      } else {
        this.showModal('Validation Error', 'All fields must be filled!', 'error');
      }
      return;
    }

    if (!this.validateEmail(this.user.email)) {
      this.showModal('Email Error', 'Email must be in @gmail.com format!', 'error');
      return;
    }

    this.checkEmailExists().pipe(
      map(exists => {
        if (exists) {
          this.showModal('Email Error', 'Email already exists! Please try another email.', 'error');
          return false;
        }
        return true;
      }),
      switchMap(emailExists => {
        if (!emailExists) return [false];
        return this.userService.checkUsernameExists(this.user.username).pipe(
          map(usernameExists => {
            if (usernameExists.exists) {
              this.showModal('Username Error', 'Username already exists! Please try another username.', 'error');
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
            this.showModal(
              'Success',
              'You are successfully registered! Check your email to confirm your account.',
              'success',
              () => this.navigateToLogin()
            );
            this.user = new User('', '', '', '', '', '', '');
          },
          error: (error) => {
            this.showModal('Password Error', 'Password is used too often.', 'error');
          }
        });
      }
    });
  }

}
