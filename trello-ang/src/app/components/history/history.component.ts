import { Component, OnInit } from '@angular/core';
import { Project } from "../../model/project.model";
import { ProjectService } from "../../services/project.service";
import { Event } from '../../model/event.model';
import { UserService } from "../../services/user.service";
import {TaskService} from "../../services/task.service"; // Importuj UserService

@Component({
  selector: 'app-history',
  templateUrl: './history.component.html',
  styleUrls: ['./history.component.css']
})
export class HistoryComponent implements OnInit {
  projects: Project[] = [];
  events: Event[] = [];
  filteredEvents: Event[] = [];

  managerUsernames: { [key: string]: string } = {}; // Ključ je managerId, vrednost je korisničko ime
  memberUsernames: { [key: string]: string } = {}; // Ključ je memberId, vrednost je korisničko ime
  projectTitles: { [key: string]: string } = {};  // Ključ je projectId, vrednost je title
  taskNames: { [key: string]: string } = {}; // Ključ je taskId, vrednost je ime zadatka

  constructor(
    private projectService: ProjectService,
    private userService: UserService,
    private taskService: TaskService
  ) {}

  ngOnInit() {
    this.loadProjects();
    this.loadEvents(); // Učitaj događaje
  }

  loadProjects() {
    const userId = localStorage.getItem('user_id');
    const token = localStorage.getItem('access_token');

    if (userId && token) {
      this.projectService.getProjectsByUser(userId, token).subscribe(
        (data: Project[]) => {
          this.projects = data;

          // Log the loaded projects in the console
          console.log("Loaded Projects:", this.projects);

        },
        (error) => {
          console.error('Error fetching projects', error);
        }
      );
    } else {
      console.error('User not logged in.');
    }
  }

  loadEvents() {
    this.projectService.getAllEvents().subscribe(
      (data: any) => {
        console.log("Raw API Response:", data);  // Proveri kako podaci zapravo izgledaju

        // Proveri da li podaci odgovaraju Event modelu
        this.events = data.map((event: any) => ({
          type: event.type || '',
          time: event.time || '',
          event: event.event || {},
          projectId: event.projectId || ''
        }));

        console.log("Mapped Events:", this.events);

        this.events.sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime());

        this.filteredEvents = this.projects.length > 0
          ? this.events.filter(event =>
            this.projects.some(project => project.id === event.projectId)
          )
          : [];

        this.filteredEvents.sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime());

        // Učitavanje dodatnih podataka
        this.filteredEvents.forEach(event => {
          if (event.event.managerId) {
            this.loadManagerUsername(event.event.managerId);
          }
          if (event.event.memberId) {
            this.loadMemberUsername(event.event.memberId);
          }
          if (event.event.projectId) {
            this.loadProjectTitle(event.event.projectId);
          }
          if (event.event.taskId) {
            this.loadTaskName(event.event.taskId);
          }
        });

        console.log("Final Processed Events:", this.filteredEvents);
      },
      (error) => {
        console.error('Error fetching events', error);
      }
    );
  }





  // Metoda koja se poziva kada korisnik promeni odabrani projekat
  onProjectChange(event: any) {
    const selectedProjectId = event.target.value;

    if (selectedProjectId) {
      // Ako je odabran projekat, filtriraj događaje prema tom ID-u
      this.filteredEvents = this.events.filter(event => event.projectId === selectedProjectId);
    } else {
      // Ako nije odabran projekat, prikaži sve događaje
      this.filteredEvents = this.events;
    }

    console.log("Filtered Events for project ID:", selectedProjectId);
  }

  loadMemberUsername(memberId: string) {
    if (!this.memberUsernames[memberId]) {
      this.userService.getUserById(memberId).subscribe(
        (user) => {
          this.memberUsernames[memberId] = user.username; // Sprema korisničko ime za člana
        },
        (error) => {
          console.error(`Error fetching username for member ID ${memberId}`, error);
        }
      );
    }
  }

  loadManagerUsername(managerId: string) {
    if (!this.managerUsernames[managerId]) {
      this.userService.getUserById(managerId).subscribe(
        (user) => {
          this.managerUsernames[managerId] = user.username; // Sprema samo korisničko ime
        },
        (error) => {
          console.error('Error fetching user', error);
        }
      );
    }
  }

  loadProjectTitle(projectId: string) {
    if (!this.projectTitles[projectId]) {
      this.projectService.getProjectByID(projectId).subscribe(
        (project) => {
          this.projectTitles[projectId] = project.title; // Sprema naslov projekta
        },
        (error) => {
          console.error(`Error fetching project title for project ID ${projectId}`, error);
        }
      );
    }
  }

  loadTaskName(taskId: string) {
    if (!this.taskNames[taskId]) {
      this.taskService.getTaskById(taskId).subscribe(
        (task) => {
          this.taskNames[taskId] = task.name; // Čuva ime zadatka
        },
        (error) => {
          console.error(`Error fetching task name for task ID ${taskId}`, error);
        }
      );
    }
  }

  adjustTimeByOneHour(dateString: string): string {
    const date = new Date(dateString);
    date.setHours(date.getHours() - 1); // Oduzimanje 1h
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    const day = date.getDate().toString().padStart(2, '0');
    const month = (date.getMonth() + 1).toString().padStart(2, '0');
    const year = date.getFullYear();
    return `${day}.${month}.${year} ${hours}:${minutes}`;
  }
}
