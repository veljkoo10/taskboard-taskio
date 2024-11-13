import { Component, HostListener, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ProjectService } from '../../services/project.service';
import { Project } from '../../model/project.model';
import { Router } from '@angular/router';
import {AuthService} from "../../services/auth.service";

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent implements OnInit {

  isProfileMenuOpen: boolean = false;
  project: Project = new Project();
  projects: Project[] = [];
  selectedProject!: Project | null;
  newProj: any

  constructor(private projectService: ProjectService, private router: Router, private authService: AuthService) {}

  ngOnInit() {
    this.loadProjects();
    
    this.projectService.projectCreated$.subscribe((newProject: Project) => {
      // Učitavamo projekte i postavljamo novi kao selektovan
      this.newProj = this.projectService.getNewProject()

      this.loadProjects();
      this.selectProject(this.newProj);
      console.log(this.newProj)
    });
  }

  selectProject(project: Project): void {
    this.selectedProject = project;
    console.log('Selected project:', this.selectedProject);
  }



  loadProjects() {
    const userId = localStorage.getItem('user_id');
    const token = localStorage.getItem('access_token');

    if (userId && token) {
      this.projectService.getProjectsByUser(userId, token).subscribe(
        (data: Project[]) => {
          this.projects = data;
        },
        (error) => {
          console.error('Error fetching projects', error);
        }
      );
    } else {
      console.error('User not logged in.');
    }
  }
}

