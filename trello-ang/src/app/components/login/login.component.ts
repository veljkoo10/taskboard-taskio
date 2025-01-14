import {Component, ElementRef} from '@angular/core';
import {Router} from "@angular/router";
import {AuthService} from "../../services/auth.service";
import { User } from '../../model/user.model';
import {UserService} from "../../services/user.service";


@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.css']
})
export class LoginComponent {
  imagePath: string = 'assets/loginSlika.svg';
  resetEmail: string = '';
  resetMessage: string = '';
  username: string = '';
  magicLinkUsername: string = '';
  password: string = '';
  loginError: string = '';
  email: string='';
  resetMessageMagic:string='';
  message:string='';
  isSuccess: boolean = false;


  user: User = new User('', '', '', '', '', '','');
  recaptchaResponse: string = '';
  constructor(private router: Router, private authService: AuthService,private userService:UserService,private elRef: ElementRef) {}


  login() {
    const recaptchaResponse = (window as any).grecaptcha.getResponse();

    // Provera CAPTCHA
    if (!recaptchaResponse) {
      this.loginError = 'Please solve the CAPTCHA.';
      return;
    }

    // Provera username-a i password-a
    if (!this.username || !this.password) {
      this.loginError = 'Please enter both username and password.';
      return;
    }

    const userCredentials = {
      username: this.username,
      password: this.password,
      recaptchaResponse
    };

    this.authService.login(userCredentials).subscribe(
      (response) => {
        const { access_token, role, user_id } = response;

        // Smeštanje podataka u localStorage
        localStorage.setItem('access_token', access_token);
        localStorage.setItem('role', role);
        localStorage.setItem('user_id', user_id.toString());
        this.router.navigate(['/dashboard']);
      },
      (error) => {
        // Proveri grešku sa servera i prikaži odgovarajuću poruku
        if (error.error === 'reCAPTCHA verification failed') {
          this.loginError = 'CAPTCHA verification failed. Please try again.';
        } else {
          this.loginError = 'Invalid username or password. Please try again.';
        }
      }
    );
  }


  sendMagicLink() {
    if (!this.email || !this.magicLinkUsername) {
      this.message = 'Molimo vas da unesete email adresu i korisničko ime.';
      this.isSuccess = false;
      return;
    }

    this.authService.sendMagicLink(this.email, this.magicLinkUsername).subscribe({
      next: (response) => {
        this.message = 'Magic link sent successfully to your email.';
        this.isSuccess = true; // Uspešna poruka
      },
      error: (error) => {

        if (error.status === 403) {
          this.message = 'Your account is not active. Contact support for more information.';
        } else if (error.status === 400 && error.error.includes('Username i email se ne podudaraju')) {
          this.message = 'The email and username entered do not match. Please check your data and try again.';
        } else if (error.status === 404) {
          this.message = 'The user with the entered email was not found. Check the entry.';
        } else {
          this.message = 'The user is not active or does not exist.';
        }
        this.isSuccess = false; // Greška
      },
    });
  }


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

    this.userService.checkUserActive(this.resetEmail).subscribe(
      (response) => {
        if (response.active) {
          this.userService.requestPasswordReset(this.resetEmail).subscribe(
            () => {
              this.resetMessage = 'A password reset link has been sent to your email address.';
              this.isSuccess = true;

            },
            (error) => {
              this.resetMessage = 'There was an error sending the password reset link. Try again.';
              this.isSuccess = false;

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
  ngAfterViewInit() {
    const forgotPasswordModal = this.elRef.nativeElement.querySelector('#forgotPasswordModal');
    if (forgotPasswordModal) {
      forgotPasswordModal.addEventListener('hidden.bs.modal', () => {
        this.resetForgotPasswordForm();
      });
    }
  }
  resetForgotPasswordForm() {
    this.resetEmail = '';
    this.resetMessage = '';
    this.isSuccess = false;
  }
  checkEmailAndUsernameAndSendMagicLink() {


    if (!this.email || !this.username) {
      this.resetMessageMagic = 'Email i Username moraju biti uneti.';
      return;
    }

    this.userService.loginWithMagic(this.email, this.username).subscribe(
      (response) => {
        this.resetMessageMagic = response.message;
      },
      (error) => {
        this.resetMessageMagic = 'Desila se greška pri slanju magic linka.';
      }
    );
  }


}
