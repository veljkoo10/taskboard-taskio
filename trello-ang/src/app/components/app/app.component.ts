import { Component, HostListener, ChangeDetectorRef, ApplicationRef, ViewChild, ElementRef } from '@angular/core';
import {ProjectService} from "../../services/project.service";
import {Router} from "@angular/router";
import {AuthService} from "../../services/auth.service";
import {Project} from "../../model/project.model";
import {DashboardComponent} from "../dashboard/dashboard.component"
import {NgForm} from "@angular/forms";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  @ViewChild('projectForm') projectForm!: NgForm;
  logoPath: string = 'assets/trello4.png';
  profilePath: string = 'assets/user3.png';
  selectedProject!: Project | null;
  isProfileMenuOpen: boolean = false;
  project: Project = new Project();
  projects: Project[] = [];
  successMessage: string = '';
  errorMessage: string = '';
  constructor(private projectService: ProjectService, private router: Router, private authService: AuthService, private changeDetectorRef: ChangeDetectorRef, private appRef: ApplicationRef) {}
  isLoggedIn() {
    return this.authService.getDecryptedData('access_token') != '';
  }
  goToDashboard() {
    if (window.location.pathname === '/dashboard') {
      location.reload();
    } else {
      this.router.navigate(['/dashboard']);
    }
  }
  goToNotifications(): void {
    this.isProfileMenuOpen = false;
    this.router.navigate(['/notification']);
  }
  logout(): void {
    this.authService.logout();
    this.isProfileMenuOpen = false;
    this.router.navigate(['/login']);
  }
  isManager(): boolean {
    return this.authService.getDecryptedData('role') === 'Manager';
  }
  goToProfile(): void {
    this.isProfileMenuOpen = false;
    this.router.navigate(['/profile']);
  }
  toggleProfileMenu(): void {
    this.isProfileMenuOpen = !this.isProfileMenuOpen;
  }
  @HostListener('document:click', ['$event'])
  onClick(event: MouseEvent): void {
    const clickedInsideProfileMenu = event.target instanceof HTMLElement && event.target.closest('.profile-menu');
    const clickedProfileIcon = event.target instanceof HTMLElement && event.target.closest('.nav-link.custom-link');
    const clickedInsideModal = event.target instanceof HTMLElement && event.target.closest('#addProjectModal');

    if (!clickedInsideProfileMenu && !clickedProfileIcon) {
      this.isProfileMenuOpen = false;
    }
    if (!clickedInsideModal) {
      this.resetForm();
    }
  }


  createProject(): void {
    // Check if all fields are filled
    if (!this.project.title || !this.project.description ||
      !this.project.expected_end_date || !this.project.min_people || !this.project.max_people) {
      this.errorMessage = 'All fields must be filled!';
      return;
    }

    // Validate minimum and maximum number of people
    if (this.project.min_people <1) {
      this.errorMessage = 'Minimum number of people must be at least 1.';
      return;
    }

    if (this.project.max_people < 2) {
      this.errorMessage = 'Maximum number of people must be at least 2.';
      return;
    }

    if (this.project.max_people < this.project.min_people) {
      this.errorMessage = 'The maximum number of people must be greater than or equal to the minimum number!';
      return;
    }


    // Check if the number of users exceeds max_people
    if (this.project.users.length > this.project.max_people) {
      this.errorMessage = `You can have a maximum of ${this.project.max_people} users!`;
      return;
    }

    // Validate the expected end date
    const currentDate = new Date();
    const expectedEndDate = new Date(this.project.expected_end_date);
    if (expectedEndDate <= currentDate) {
      this.errorMessage = 'The project completion date must be after today\'s date!';
      return;
    }

    // Clear any previous error messages
    this.errorMessage = '';

    const managerId = this.authService.getDecryptedData('user_id');
    if (!managerId) {
      this.errorMessage = 'Manager ID is missing. Please log in again.';
      return;
    }

    // Check if a project with the same title already exists
    this.projectService.checkProjectByTitle(this.project.title, managerId).subscribe(
      (response: string) => {
        if (response === 'Project exists') {
          this.errorMessage = 'A project with this title already exists.';
        } else if (response === 'Project not found') {
          const projectPayload = {
            title: this.project.title,
            description: this.project.description,
            expected_end_date: this.project.expected_end_date,
            min_people: this.project.min_people,
            max_people: this.project.max_people,
            users: this.project.users,
            manager_id: managerId
          };

          this.projectService.createProject(managerId, projectPayload).subscribe(
            (response: Project) => {
              console.log('Project created successfully:', response);

              this.projectService.notifyProjectCreated(response);

              this.project = new Project();
              this.successMessage = 'The project was successfully created!';
              this.projects.push(response);

              // Close modal (if applicable)
              const closeModalButton = document.querySelector('[data-bs-dismiss="modal"]');
              if (closeModalButton) {
                (closeModalButton as HTMLElement).click();
              }
            },
            (error) => {
              if (error.status === 500 && error.error === 'Project with this name already exists for the same manager\n') {
                this.errorMessage = 'A project with this title already exists.';
              } else {
                console.error('Error creating project:', error);
                this.errorMessage = 'There was an error creating the project.';
              }
            }
          );
        }
      },
      (error) => {
        console.error('Error checking project title:', error);
        this.errorMessage = 'There was an error checking the project title.';
      }
    );
  }

  resetForm(): void {
    if (this.projectForm) {
      this.projectForm.resetForm(); // Reset form if the reference is available
    }
    this.project = new Project();
    this.errorMessage = '';
    this.successMessage = '';
  }

  loadProjects() {
    this.projectService.getProjects().subscribe(
      (data: Project[]) => {
        this.projects = data;
      },
      (error) => {
        console.error('Error fetching projects', error);
      }
    );
  }
}
