import { Component, HostListener, ChangeDetectorRef, ApplicationRef, ViewChild, ElementRef, OnInit, OnDestroy } from '@angular/core';
import { ProjectService } from "../../services/project.service";
import { NavigationEnd, Router } from "@angular/router";
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
  userPath :string = 'assets/usernav.png';
  notificationPath: string = 'assets/bell.png';
  activityPath: string = 'assets/activity.png';
  logoutPath: string = 'assets/logout.png';
  analiticsPath: string = 'assets/data-analytics.png';
  selectedProject!: Project | null;
  isProfileMenuOpen: boolean = false;
  project: Project = new Project();
  projects: Project[] = [];
  successMessage: string = '';
  errorMessage: string = '';
  hasNotifications: boolean = false;
  private notificationCheckInterval: any;
  private timeoutModalInterval: any;
  private interval: any;
  private secondsPassed: number = 0;

  ngOnInit() {
    // Logovanje trenutnog URL-a pri inicijalizaciji
    console.log(`Current component URL: ${this.router.url}`);

    if (!['/login', '/register'].includes(this.router.url)) {
      this.startTimer();
    }

    // Provera notifikacija se ne pokreće za menadžera
    if (!this.isManager() && this.router.url !== '/notification') {
      this.startNotificationCheck();
    }

    this.router.events.subscribe((event) => {
      if (event instanceof NavigationEnd) {
        const currentPath = this.router.url;

        // Logovanje trenutnog URL-a nakon navigacije
        console.log(`Navigated to URL: ${currentPath}`);

        // Provera notifikacija se ne pokreće za menadžera
        if (!this.isManager()) {
          if (currentPath === '/notification') {
            this.stopNotificationCheck();
            this.hasNotifications = false; // Postavite hasNotifications na false kada ste na /notification
          } else {
            this.startNotificationCheck();
          }
        }

        // Proveri da li trenutna ruta nije /login ili /register pre pokretanja tajmera
        if (!['/login', '/register'].includes(currentPath)) {
          this.startTimer();
        } else {
          // Zaustavi tajmer ako je na isključenim rutama
          if (this.interval) {
            clearInterval(this.interval);
          }
        }
      }
    });
  }


  goToNotifications(): void {
    this.isProfileMenuOpen = false;
    this.stopNotificationCheck();
    this.hasNotifications = false; // Postavite hasNotifications na false
    this.router.navigate(['/notification']);
  }

  constructor(private projectService: ProjectService, private router: Router, private authService: AuthService,
              private notificationService: NotificationService,
              private changeDetectorRef: ChangeDetectorRef, private appRef: ApplicationRef) {}

  isLoggedIn(): boolean {
    return localStorage.getItem('access_token') != null;
  }

  goToDashboard() {
    if (window.location.pathname === '/dashboard') {
      location.reload();
    } else {
      this.router.navigate(['/dashboard']);
    }
  }

  startTimer() {
    const excludedRoutes = ['/login', '/register'];

    // Provera da li trenutni URL pripada isključenim rutama
    if (excludedRoutes.includes(this.router.url)) {
      return;
    }

    // Dohvatanje prethodno sačuvanog vremena iz localStorage
    const savedTime = localStorage.getItem('secondsPassed');
    this.secondsPassed = savedTime ? parseInt(savedTime, 10) : 0;

    // Ako interval već postoji, resetuj ga
    if (this.interval) {
      clearInterval(this.interval);
    }

    // Postavljanje novog intervala koji povećava broj sekundi
    this.interval = setInterval(() => {
      this.secondsPassed++;

      // Sačuvaj trenutno vreme u localStorage
      localStorage.setItem('secondsPassed', this.secondsPassed.toString());
    }, 1000);

    // Ako postoji prethodni timeout za modal, resetuj ga
    if (this.timeoutModalInterval) {
      clearTimeout(this.timeoutModalInterval);
    }

    // Postavljanje timeouta za prikaz modalnog prozora nakon 12 minuta
    const remainingTime = (12 * 60 * 1000) - (this.secondsPassed * 1000);
    this.timeoutModalInterval = setTimeout(() => {
      clearInterval(this.interval);
      this.openLogoutModal();
    }, remainingTime > 0 ? remainingTime : 0);
  }

  clearTimers() {
    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }
    if (this.timeoutModalInterval) {
      clearTimeout(this.timeoutModalInterval);
      this.timeoutModalInterval = null;
    }

    // Resetuj vrednost u localStorage
    localStorage.removeItem('secondsPassed');
  }


  logoutToken() {
    this.authService.logout();
    this.isProfileMenuOpen = false;
    this.router.navigate(['/login']);
    this.stopNotificationCheck();
    this.hasNotifications = false;
    this.closeLogoutModal();
    this.clearTimers();
  }

  goToHistory(){
    this.isProfileMenuOpen = false;
    this.router.navigate(['/history']);
  }
  goToAnalytics(){
    this.isProfileMenuOpen = false;
    this.router.navigate(['/analytics']);
  }
  isDashboard(): boolean {
    return this.router.url === '/dashboard';
  }
  ngOnDestroy() {
    if (this.notificationCheckInterval) {
      clearInterval(this.notificationCheckInterval);
    }
    this.clearTimers();
  }

  openLogoutModal(): void {
    const modal = document.querySelector('.modal-wrapper');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }

  closeLogoutModal() {
    const modal = document.querySelector('.modal-wrapper');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0%');
    }
  }

  logout(): void {
    this.authService.logout();
    this.isProfileMenuOpen = false;
    this.router.navigate(['/login']);
    this.stopNotificationCheck();
    this.hasNotifications = false;
    this.clearTimers();
  }

  isManager(): boolean {
    return localStorage.getItem('role') === 'Manager';
  }

  isMember(): boolean {
    return localStorage.getItem('role') === 'Member';
  }

  goToProfile(): void {
    this.isProfileMenuOpen = false;
    this.router.navigate(['/profile']);
  }
  onHover() {
    this.isProfileMenuOpen = !this.isProfileMenuOpen;
  }

  onHoverExit() {
    this.isProfileMenuOpen = false;
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
    const userID = localStorage.getItem('user_id');

    // Ako je korisnik menadžer, preskoči proveru notifikacija
    if (this.isManager()) {
      this.hasNotifications = false;
      return;
    }

    if (userID) {
      this.notificationService.getNotificationsByUserID(userID).subscribe(
        (notifications) => {
          if (notifications === null || notifications === undefined) {
            this.stopNotificationCheck();
          } else {
            this.handleNotifications(notifications);
          }
        },
        () => {
          this.stopNotificationCheck();
        }
      );
    } else {
      this.stopNotificationCheck();
    }
  }

  handleNotifications(notifications: any[]) {
    // Ako je korisnik na /notification ruti, nemoj prikazivati tačku za notifikacije
    if (this.router.url === '/notification') {
      this.hasNotifications = false;
      return;
    }

    // Ako je korisnik menadžer, nemoj prikazivati tačku za notifikacije
    if (this.isManager()) {
      this.hasNotifications = false;
      return;
    }

    if (notifications && Array.isArray(notifications)) {
      const unreadNotifications = notifications.filter(notification => notification.status === 'unread');
      this.hasNotifications = unreadNotifications.length > 0;
      this.changeDetectorRef.detectChanges();
    } else {
      this.stopNotificationCheck();
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

              const closeModalButton = document.querySelector('[data-bs-dismiss="modal"]');
              if (closeModalButton) {
                (closeModalButton as HTMLElement).click();
              }
            },
            (error) => {
              if (error.status === 500 && error.error === 'Project with this name already exists for the same manager\n') {
                this.errorMessage = 'A project with this title already exists.';
              } else {
                this.errorMessage = 'There was an error creating the project.';
              }
            }
          );
        }
      },
      () => {
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
