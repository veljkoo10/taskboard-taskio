import { Component, Input } from '@angular/core';
import { Project } from '../../model/project.model';
import { ViewChild, AfterViewInit } from '@angular/core';
import { Modal } from 'bootstrap';
import {DashboardComponent} from '../dashboard/dashboard.component'
import { ProjectService } from 'src/app/services/project.service';
import { SimpleChanges } from '@angular/core';

@Component({
  selector: 'app-project-details',
  templateUrl: './project-details.component.html',
  styleUrls: ['./project-details.component.css']
})
export class ProjectDetailsComponent {
  @Input() project: Project | null = null;
  taskName: string = '';
  taskDescription: string = '';
  isCreateTaskFormVisible: boolean = false;
  pendingTasks: string[] = [];  // Lista imena zadataka koji su pending

  constructor(private projectService: ProjectService) {}

  ngOnChanges(changes: SimpleChanges) {
    if (changes['project'] && changes['project'].currentValue) {
      this.pendingTasks = []; // Resetujemo listu pre nego što učitamo nove podatke
      this.loadPendingTasks(); // Učitavamo pending taskove za novi projekat
    }
  }
  // Metoda koja se poziva prilikom inicijalizacije komponente
  ngOnInit() {
    this.loadPendingTasks();  // Pozivamo metodu za učitavanje pending taskova
  }

  loadPendingTasks() {
    const project = this.project as any; 
    this.pendingTasks = []; // Resetujemo listu pending taskova pre svakog učitavanja
    if (this.project) {
      const projectIdStr = String(project.id);
      this.projectService.getTasks().subscribe(tasks => {
        this.pendingTasks = tasks
          .filter(task => task.status === 'pending' && String(task.project_id) === projectIdStr)
          .map(task => task.name);
      });
    }
  }

  // Ova metoda prikazuje modal
  showCreateTaskForm() {
    const project = this.project as any;  // Pretvori 'project' u 'any' tip
    console.log(project.id);  // Sada možeš pristupiti 'id' bez greške
    this.isCreateTaskFormVisible = true;
    document.querySelector('#mm')?.setAttribute("style", "display:block; opacity: 100%; margin-top: 20px");
  }


  // Ova metoda zatvara modal
  cancelCreateTask() {
    this.isCreateTaskFormVisible = false; // Modal postaje nevidljiv
    this.taskName = '';  // Resetuje task name
    this.taskDescription = '';  // Resetuje task description
  }

 // Funkcija za kreiranje taska
 createTask() {
  const project = this.project as any;
  if (project) {
    const newTask = {
      name: this.taskName,
      description: this.taskDescription
    };

    // Pozivamo servis, a projectId šaljemo kao parametar u URL-u
    this.projectService.createTask(project.id, newTask).subscribe(
      (response) => {
        console.log('Task successfully created:', response);
        this.cancelCreateTask();  // Zatvaranje modala nakon kreiranja taska
        window.location.reload();
      },
      (error) => {
        console.error('Error creating task:', error);
      }
    );
  }
}

}


