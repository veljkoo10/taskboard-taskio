import { Component, HostListener, ChangeDetectorRef, ApplicationRef, ViewChild, ElementRef, OnInit, OnDestroy } from '@angular/core';
import { ProjectService } from "../../services/project.service";
import {NavigationEnd, Router} from "@angular/router";
import { AuthService } from "../../services/auth.service";
import { Project } from "../../model/project.model";
import { DashboardComponent } from "../dashboard/dashboard.component";
import { NgForm } from "@angular/forms";
import { NotificationService } from "../../services/notification.service";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit, OnDestroy {
  @ViewChild('projectForm') projectForm!: NgForm;
  logoPath: string = 'assets/trello4.png';
  profilePath: string = 'assets/user3.png';
  selectedProject!: Project | null;
  isProfileMenuOpen: boolean = false;
  project: Project = new Project();
  projects: Project[] = [];
  successMessage: string = '';
  errorMessage: string = '';
  hasNotifications: boolean = false;
  private notificationCheckInterval: any;

  ngOnInit() {
    this.router.events.subscribe((event) => {
      if (event instanceof NavigationEnd) {
        const currentPath = window.location.pathname;

        // Stop notification checking when navigating to /notification page
        if (currentPath === '/notification') {
          this.stopNotificationCheck();
          this.hasNotifications = false;  // Reset notification dot
        } else {
          // Start notification check when navigating to other pages
          this.startNotificationCheck();
        }
      }
    });

    // If not on /notification, start checking for notifications
    if (!this.isManager() && window.location.pathname !== '/notification') {
      this.startNotificationCheck();
    }
  }

  goToNotifications(): void {
    this.isProfileMenuOpen = false;
    this.router.navigate(['/notification']);
    this.hasNotifications = false;  // Reset the notification dot when navigating to the notifications page

    // Stop notification checking when on /notification page
    if (this.notificationCheckInterval) {
      clearInterval(this.notificationCheckInterval);  // Stop the notification check
    }
  }




  constructor(private projectService: ProjectService, private router: Router, private authService: AuthService,
              private notificationService: NotificationService,
              private changeDetectorRef: ChangeDetectorRef, private appRef: ApplicationRef) {}

  ngOnDestroy() {
    if (this.notificationCheckInterval) {
      clearInterval(this.notificationCheckInterval);
    }
  }

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





  logout(): void {
    this.authService.logout();
    this.isProfileMenuOpen = false;
    this.router.navigate(['/login']);
    this.stopNotificationCheck();
    this.hasNotifications = false;
  }

  isManager(): boolean {
    return this.authService.getDecryptedData('role') === 'Manager';
  }

  isMember(): boolean {
    return this.authService.getDecryptedData('role') === 'Member';
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

  checkForNotifications() {
    const userID = this.authService.getDecryptedData('user_id');

    if (userID) {
      this.notificationService.getNotificationsByUserID(userID).subscribe(
        (notifications) => {
          if (notifications === null || notifications === undefined) {
            this.stopNotificationCheck(); // Stop checking if the data is invalid
          } else {
            this.handleNotifications(notifications);
          }
        },
        (error) => {
          console.error('Error fetching notifications', error);
          this.stopNotificationCheck(); // Stop checking on error
        }
      );
    } else {
      this.stopNotificationCheck();
    }
  }



  handleNotifications(notifications: any[]) {
    if (notifications && Array.isArray(notifications)) {
      const unreadNotifications = notifications.filter(notification => notification.status === 'unread');
      this.hasNotifications = unreadNotifications.length > 0;

      // Trigger change detection to update the view
      this.changeDetectorRef.detectChanges();
    } else {
      console.error('Invalid notifications data:', notifications);
      this.stopNotificationCheck(); // Stop checking on invalid data
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
      this.projectForm.resetForm();
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

  startNotificationCheck() {
    this.notificationCheckInterval = setInterval(() => {
      this.checkForNotifications();
    }, 1000);
  }

  stopNotificationCheck() {
    if (this.notificationCheckInterval) {
      clearInterval(this.notificationCheckInterval);
    }
  }
}
