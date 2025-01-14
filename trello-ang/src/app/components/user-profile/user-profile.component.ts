import { Component, OnInit } from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import { AuthService } from 'src/app/services/auth.service';
import { UserService } from 'src/app/services/user.service';
import{ ProjectService } from 'src/app/services/project.service';
import { Project } from 'src/app/model/project.model'
import { forkJoin, of } from 'rxjs';

@Component({
  selector: 'app-user-profile',
  templateUrl: './user-profile.component.html',
  styleUrls: ['./user-profile.component.css']
})
export class UserProfileComponent implements OnInit {
  userIcon = 'assets/user.png';
  user: any;
  userId: string = '';
  accountDeleteMessage: string = 'Are you sure you want to delete your account?';


  project: Project = new Project();
  projects: Project[] = [];

  oldPassword: string = '';
  newPassword: string = '';
  confirmPassword: string = '';

  errorMessage: string = '';
  successMessage: string = '';
  passwordError: string = '';
  deleteAccountModalVisible: boolean = false;

  constructor( private router: Router,private authService: AuthService, private userService: UserService, private projectService: ProjectService) {}

  ngOnInit(): void {
    this.user = this.getUserInfoFromToken();
    if (this.user && this.user.id) {
      console.log("User ID from token:", this.user.id);

      this.userService.getUserById(this.user.id).subscribe({
        next: (data) => {
          this.user = data;
          console.log("User data from server:", this.user);
        },
        error: (error) => {
          console.error('Error fetching user profile:', error);
        }
      });
    }
    this.loadProjects()
  }
  onDeleteAccount(): void {
    this.accountDeleteMessage = 'Are you sure you want to delete your account?';
    this.deleteAccountModalVisible = true;
  }
  resetPasswordFields(): void {
    this.oldPassword = '';
    this.newPassword = '';
    this.confirmPassword = '';
  }

  canDeleteAccount(): boolean {
    return this.accountDeleteMessage === 'Are you sure you want to delete your account?';
  }

  closeDeleteAccountModal(): void {
    this.deleteAccountModalVisible = false;
    this.accountDeleteMessage = 'Are you sure you want to delete your account?';
  }

  closeModalAndRefresh(): void {
    this.resetPasswordFields();
    this.successMessage = '';
    this.errorMessage = '';
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
    // Provera da li su sva polja popunjena
    if (!this.oldPassword || !this.newPassword || !this.confirmPassword) {
      this.errorMessage = 'All fields must be filled out.';
      return;
    }

    // Provera da li se nova lozinka i potvrda lozinke podudaraju
    if (this.newPassword !== this.confirmPassword) {
      this.errorMessage = 'New password and confirm password do not match.';
      return;
    }

    // Provera validnosti nove lozinke (ako imaš neku specifičnu validaciju)
    if (this.isPasswordValid(this.newPassword)) {
      const userId = this.user.id;
      const passwordData = {
        oldPassword: this.oldPassword,
        newPassword: this.newPassword,
        confirmPassword: this.confirmPassword,
      };

      // Poziv servisa za promenu lozinke
      this.userService.changePassword(userId, passwordData).subscribe({
        next: () => {
          this.successMessage = 'Password changed successfully!';
          this.errorMessage = '';
          this.resetPasswordFields();

        },
        error: (error) => {
          // Dinamičko preuzimanje i prikazivanje greške sa backend-a
          if (error.error && typeof error.error === 'string') {
            this.errorMessage = error.error; // Prikazuje stvarnu grešku koju vrati backend
          } else {
            this.errorMessage = 'An unexpected error occurred. Please try again.';
          }
          this.successMessage = '';
        },
      });
    }

    // Ako postoji greška u validaciji lozinke
    if (this.passwordError) {
      this.errorMessage = this.passwordError;
    }
  }



  deleteUserAccount(): void {
    console.log(this.projects)
    if(this.projects === null){
      this.userService.deactivateUser(this.user.id).subscribe({
        next: (response) => {
          this.accountDeleteMessage = 'Your account has been deleted.';
          this.authService.logout();
          this.router.navigate(['/login']);
        },
        error: (error) => {
          console.error('Error deleting user:', error);
          this.accountDeleteMessage = 'There was an error deleting your account.';
        }
      });

      this.closeDeleteAccountModal();
    }
    const projectStatusChecks = this.projects.map(project => {
      if (project.id) {
        return this.projectService.isProjectActive(project.id);
      } else {
        return of(false);
      }
    });


    forkJoin(projectStatusChecks).subscribe(
      (results) => {
        const anyProjectActive = results.some(isActive => isActive);
        if (anyProjectActive) {
          this.accountDeleteMessage = 'You cannot delete your account because some projects are still active.';
          return;
        }

        this.userService.deactivateUser(this.user.id).subscribe({
          next: (response) => {
            this.accountDeleteMessage = 'Your account has been deleted.';
            this.authService.logout();
            this.router.navigate(['/login']);
          },
          error: (error) => {
            console.error('Error deleting user:', error);
            this.accountDeleteMessage = 'There was an error deleting your account.';
          }
        });

        this.closeDeleteAccountModal();
      },
      (error) => {
        console.error('Error checking project status:', error);
        this.accountDeleteMessage = 'There was an error checking project status.';
      }
    );
  }


  loadProjects() {
    const userId = localStorage.getItem('user_id');
    const token = localStorage.getItem('access_token');

    if (userId && token) {
      this.projectService.getProjectsByUser(userId, token).subscribe(
        (data: Project[]) => {
          this.projects = data;
          console.log(this.projects)
        },
        (error) => {
          console.error('Error fetching projects', error);
        }
      );
    } else {
      console.error('User not logged in.');
    }
  }
  closeResultModal(): void {
    this.successMessage = '';
    this.errorMessage = '';
    this.resetPasswordFields();

  }

}
