import { Component, OnInit } from '@angular/core';
import { AnalyticsService } from '../../services/analytics.service';

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

  constructor(private analyticsService: AnalyticsService) {}

  ngOnInit(): void {
    const userId = localStorage.getItem('user_id');


    if (userId) {
      // Pozivamo funkciju samo ako je userId definisan
      this.loadTaskCount(userId);
      this.loadTaskCountByStatus(userId)
      this.loadUserProjects(userId)
      this.loadProjectCompletionStatuses(userId)
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

  // Funkcija koja učitava projekte i zadatke korisnika
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
        console.log('Task Analytics:', this.taskAnalytics);
      },
      error: (err) => {
        console.error('Error fetching task analytics:', err);
      }
    });
  }
}
