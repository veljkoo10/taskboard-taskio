import { Component, HostListener } from '@angular/core';
import {NgForm} from "@angular/forms";
import {ProjectService} from "../../services/project.service";
import {Project} from "../../model/project.model";

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent {
  logoPath: string = 'assets/trello4.png';
  profilePath: string = 'assets/user3.png';
  isProfileMenuOpen: boolean = false;
  project: Project = new Project();
  successMessage: string = '';
  errorMessage: string = '';
  constructor(private projectService: ProjectService) {}

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
    // Provera da li su sva polja popunjena
    if (!this.project.title || !this.project.description || !this.project.owner ||
      !this.project.expected_end_date || !this.project.min_people || !this.project.max_people) {
      this.errorMessage = 'Sva polja moraju biti popunjena!';
      return;
    }

    // Provera da maksimalan broj bude veći ili jednak minimalnom broju
    if (this.project.max_people < this.project.min_people) {
      this.errorMessage = 'Maksimalan broj ljudi mora biti veći ili jednak minimalnom broju!';
      return;
    }

    // Resetovanje poruke o grešci
    this.errorMessage = '';

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
        this.successMessage = 'Projekat je uspešno kreiran!';

        const closeModalButton = document.querySelector('[data-bs-dismiss="modal"]');
        if (closeModalButton) {
          (closeModalButton as HTMLElement).click();
        }
      },
      error => {
        console.error('Error creating project:', error);
        this.errorMessage = 'Došlo je do greške prilikom kreiranja projekta.';
      }
    );
  }


}
