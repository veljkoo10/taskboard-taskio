import { Component, OnInit } from '@angular/core';
import { AnalyticsService } from '../../services/analytics.service';
import { TaskService } from '../../services/task.service'; // Import TaskService

@Component({
  selector: 'app-analytics',
  templateUrl: './analytics.component.html',
  styleUrls: ['./analytics.component.css']
})
export class AnalyticsComponent implements OnInit {
  taskCount: number | null = null;
  taskCountByStatus: { done: number; pending: number; 'work in progress': number } | null = null;
  userProjects: any = null;
  projectCompletionStatuses: any = null;
  taskAnalytics: any[] | null = null; // Podaci o zadacima

  constructor(
    private analyticsService: AnalyticsService,
    private taskService: TaskService // Inject TaskService
  ) {}

  ngOnInit(): void {
    const userId = localStorage.getItem('user_id');

    if (userId) {
      // Pozivamo funkciju samo ako je userId definisan
      this.loadTaskCount(userId);
      this.loadTaskCountByStatus(userId);
      this.loadUserProjects(userId);
      this.loadProjectCompletionStatuses(userId);
      this.loadTaskAnalytics(userId); // Poziv izdvojene funkcije
    } else {
      console.error('User ID not found in localStorage.');
    }
  }

  private loadTaskCount(userId: string): void {
    this.analyticsService.getUserTaskCount(userId).subscribe({
      next: (data) => {
        this.taskCount = data.task_count;
      },
      error: (err) => {
        console.error('Failed to fetch task count:', err);
      },
    });
  }

  private loadTaskCountByStatus(userId: string): void {
    this.analyticsService.getUserTaskStatusCount(userId).subscribe({
      next: (data) => {
        this.taskCountByStatus = data;
      },
      error: (err) => {
        console.error('Failed to fetch task count by status:', err);
      },
    });
  }

  private loadUserProjects(userId: string): void {
    this.analyticsService.getUserTaskProject(userId).subscribe({
      next: (data) => {
        this.userProjects = data.projects; // Dodajemo projekte i zadatke u promenljivu
      },
      error: (err) => {
        console.error('Failed to fetch user projects:', err);
      },
    });
  }

  // Method to capitalize the first letter of a string
  capitalizeFirstLetter(value: string): string {
    if (!value) return value; // Return the value as-is if it's null or undefined
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  private loadProjectCompletionStatuses(userId: string): void {
    this.analyticsService.getProjectCompletionStatuses(userId).subscribe({
      next: (data) => {
        this.projectCompletionStatuses = data;
      },
      error: (err) => {
        console.error('Failed to fetch project completion statuses:', err);
      },
    });
  }

  // Funkcija za učitavanje analitike zadataka korisnika
  private loadTaskAnalytics(userId: string): void {
    this.analyticsService.getUserTaskAnalytics(userId).subscribe({
      next: (data) => {
        this.taskAnalytics = data; // Postavljanje dobijenih podataka u promenljivu
        this.loadTaskNames(); // Fetch task names after loading analytics
        console.log('Task Analytics:', this.taskAnalytics);
      },
      error: (err) => {
        console.error('Error fetching task analytics:', err);
      }
    });
  }

  // Fetch task names for each task in taskAnalytics
  private loadTaskNames(): void {
    if (this.taskAnalytics) {
      this.taskAnalytics.forEach((task) => {
        this.taskService.getTaskById(task.task_id).subscribe({
          next: (taskData) => {
            // Capitalize the first letter of the task name
            task.task_name = taskData.name.charAt(0).toUpperCase() + taskData.name.slice(1);
          },
          error: (err) => {
            console.error('Error fetching task name:', err);
          }
        });
      });
    }
  }
}
