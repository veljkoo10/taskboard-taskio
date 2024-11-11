import {Component, HostListener} from '@angular/core';
import {ProjectService} from "../../services/project.service";
import {Router} from "@angular/router";
import {AuthService} from "../../services/auth.service";
import {Project} from "../../model/project.model";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  logoPath: string = 'assets/trello4.png';
  profilePath: string = 'assets/user3.png';
  selectedProject!: Project | null;
  isProfileMenuOpen: boolean = false;
  project: Project = new Project();
  projects: Project[] = [];
  successMessage: string = '';
  errorMessage: string = '';
  constructor(private projectService: ProjectService, private router: Router, private authService: AuthService) {}
  isLoggedIn() {
    return localStorage.getItem('access_token') != null;
  }
  goToDashboard() {
    if (window.location.pathname === '/dashboard') {
      location.reload();  // Ako si već na dashboard stranici, osveži
    } else {
      this.router.navigate(['/dashboard']);  // Inače, navigiraj na dashboard
    }
  }

  logout(): void {
    this.authService.logout();
    this.isProfileMenuOpen = false;
    this.router.navigate(['/login']);
  }
  isManager(): boolean {
    return localStorage.getItem('role') === 'Manager';
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
    const clickedInside = event.target instanceof HTMLElement && event.target.closest('.profile-menu');
    const clickedProfileIcon = event.target instanceof HTMLElement && event.target.closest('.nav-link.custom-link');
    if (!clickedInside && !clickedProfileIcon) {
      this.isProfileMenuOpen = false;
    }
  }
  createProject(): void {
    if (!this.project.title || !this.project.description ||
      !this.project.expected_end_date || !this.project.min_people || !this.project.max_people) {
      this.errorMessage = 'All fields must be filled!';
      return;
    }

    if (this.project.min_people < 1) {
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

    if (this.project.users.length > this.project.max_people) {
      this.errorMessage = `You can have a maximum of ${this.project.max_people} users!`;
      return;
    }

    const currentDate = new Date();
    const expectedEndDate = new Date(this.project.expected_end_date);
    if (expectedEndDate <= currentDate) {
      this.errorMessage = 'The project completion date must be after today\'s date!';
      return;
    }

    this.errorMessage = '';

    const managerId = localStorage.getItem('user_id');

    if (!managerId) {
      this.errorMessage = 'Manager ID is missing. Please log in again.';
      return;
    }

    this.projectService.checkProjectByTitle(this.project.title).subscribe(
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
            response => {
              console.log('Project created successfully:', response);
              this.project = new Project();
              this.successMessage = 'The project was successfully created!';

              this.projects.push(response);

              this.loadProjects();

              const closeModalButton = document.querySelector('[data-bs-dismiss="modal"]');
              if (closeModalButton) {
                (closeModalButton as HTMLElement).click();
              }
            },
            error => {
              console.error('Error creating project:', error);
              this.errorMessage = 'There was an error creating the project.';
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
