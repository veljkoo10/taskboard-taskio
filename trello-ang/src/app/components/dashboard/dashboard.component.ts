import { Component, HostListener, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ProjectService } from '../../services/project.service';
import { Project } from '../../model/project.model';
import { Router } from '@angular/router';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent implements OnInit {
  logoPath: string = 'assets/trello4.png';
  profilePath: string = 'assets/user3.png';
  isProfileMenuOpen: boolean = false;
  project: Project = new Project();
  successMessage: string = '';
  errorMessage: string = '';
  projects: Project[] = [];
  selectedProject!: Project | null;

  constructor(private projectService: ProjectService, private router: Router) {}

  ngOnInit() {
    this.loadProjects();
  }

  selectProject(project: Project) {
    this.selectedProject = project;
  }

  goToDashboard(): void {
    this.selectedProject = null;
    this.router.navigate(['/dashboard']);
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

  toggleProfileMenu(): void {
    this.isProfileMenuOpen = !this.isProfileMenuOpen;
  }

  @HostListener('document:click', ['$event'])
  closeProfileMenu(event: Event): void {
    const target = event.target as HTMLElement;
    const isClickInsideMenu = target.closest('.profile-menu') || target.closest('.nav-link');

    if (!isClickInsideMenu) {
      this.isProfileMenuOpen = false;
    }
  }

  createProject(): void {
    if (!this.project.title || !this.project.description || !this.project.owner ||
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

    this.projectService.checkProjectByTitle(this.project.title).subscribe(
      (response: string) => {
        if (response === 'Project exists') {
          this.errorMessage = 'A project with this title already exists.';
        } else if (response === 'Project not found') {
          const projectPayload = {
            title: this.project.title,
            description: this.project.description,
            owner: this.project.owner,
            expected_end_date: this.project.expected_end_date,
            min_people: this.project.min_people,
            max_people: this.project.max_people,
            users: this.project.users
          };

          this.projectService.createProject(projectPayload).subscribe(
            response => {
              console.log('Project created successfully:', response);
              this.project = new Project();
              this.successMessage = 'The project was successfully created!';
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


}
