import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { User } from '../../model/user.model';
import { FormsModule } from "@angular/forms";
import { AuthService } from "../../services/auth.service";
import {filter, map, switchMap} from "rxjs";

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

  constructor(private router: Router, private authService: AuthService) {}

  navigateToLogin() {
    this.router.navigate(['/login']);  }

  isFormValid(): boolean {
    return (
      this.user.username.trim() !== '' &&
      this.user.name.trim() !== '' &&
      this.user.surname.trim() !== '' &&
      this.user.email.trim() !== '' &&
      this.user.password.trim() !== '' &&
      this.user.role.trim() !== ''
    );
  }

  validateEmail(email: string): boolean {
    return email.endsWith('@gmail.com');
  }

  checkEmailExists() {
    return this.authService.checkEmailExists(this.user.email).pipe(
      map(response => response.exists)
    );
  }

  register() {
    if (!this.isFormValid()) {
      alert('Sva polja moraju biti popunjena!');
      return;
    }
    if (!this.validateEmail(this.user.email)) {
      alert('Email mora biti u formatu @gmail.com!');
      return;
    }

    this.checkEmailExists().pipe(
      map(exists => {
        if (exists) {
          alert('Email već postoji! Pokušajte sa drugim emailom.');
          return false;
        }
        return true;
      }),
      switchMap(emailExists => {
        if (!emailExists) return [false];
        return this.authService.checkUsernameExists(this.user.username).pipe(
          map(usernameExists => {
            if (usernameExists.exists) {
              alert('Korisničko ime već postoji! Pokušajte sa drugim korisničkim imenom.');
              return false;
            }
            return true;
          })
        );
      }),
      filter(exists => exists) // Filtrira da nastavi samo ako je korisničko ime novo
    ).subscribe(shouldRegister => {
      if (shouldRegister) {
        this.authService.register(this.user).subscribe({
          next: (response) => {
            console.log('Registration successful:', response);
            alert('Uspešno ste registrovani! Proverite svoj email radi potvrde naloga.');
            this.user = new User('', '', '', '', '', '');
            this.navigateToLogin();
          },
          error: (error) => {
            console.error('Registration failed:', error);
            alert('Podaci nisu tačni.');
          }
        });
      }
    });
  }

}
