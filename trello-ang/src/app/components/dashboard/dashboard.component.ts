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
    this.projectService.createProject(this.project).subscribe(response => {
      console.log('Project created:', response);
      this.project = new Project(); // reset project
    }, error => {
      console.error('Error creating project:', error);
    });
  }
}
