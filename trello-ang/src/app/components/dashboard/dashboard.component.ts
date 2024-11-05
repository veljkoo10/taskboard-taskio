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
    // Prepare the project object to match the backend structure
    const projectPayload = {
      title: this.project.title,
      description: this.project.description,
      owner: this.project.owner,
      expected_end_date: this.project.expected_end_date, // Use the updated property name
      min_people: this.project.min_people,               // Use the updated property name
      max_people: this.project.max_people,               // Use the updated property name
      users: this.project.users                           // Assuming this is an array
    };
  
    this.projectService.createProject(projectPayload).subscribe(
      response => {
        console.log('Project created successfully:', response);
        // Clear the form after successful project creation
        this.project = new Project();
        // Close the modal
        let modal = document.getElementById('addProjectModal');
        if (modal) modal.click();
      },
      error => {
        console.error('Error creating project:', error);
      }
    );
  }
  

}
