import {Component, Input, SimpleChanges} from '@angular/core';
import { Project } from '../../model/project.model';
import { ProjectService } from 'src/app/services/project.service';
import { UserService } from 'src/app/services/user.service';
import { ChangeDetectorRef } from '@angular/core';


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
  isAddMemberFormVisible:boolean=false;
  pendingTasks: string[] = [];
  users: any[] = [];
  selectedUsers: any[] = [];
  selectedTask: any = null;
  isTaskDetailsVisible: boolean = false;
  projectId: string | null = null;
  projectUsers: any[] = [];
  constructor(private projectService: ProjectService,private userService: UserService,private cdRef: ChangeDetectorRef) {}

  ngOnChanges(changes: SimpleChanges) {
    if (changes['project'] && changes['project'].currentValue) {
      this.pendingTasks = [];
      this.loadPendingTasks();
    }
  }
  ngOnInit() {
    this.loadPendingTasks();
    this.loadActiveUsers()
    if (this.project && this.project.title) {
      this.getProjectIDByTitle(this.project.title);
    }
  }
  getProjectIDByTitle(title: string) {
    this.projectService.getProjectIDByTitle(title).subscribe(
      (response: any) => {
        const projectId = response?.projectId;

        if (typeof projectId === 'string') {
          console.log('Project ID:', projectId);
          if (this.project) {
            this.project.id = projectId;
            this.projectId = projectId;
            this.loadUsersForProject(projectId);
          }
        } else {
          console.error('Project ID nije string:', response);
        }
      },
      (error) => {
        console.error('Error fetching project ID:', error);
      }
    );
  }
  loadUsersForProject(projectId: string) {
    this.projectService.getUsersForProject(projectId).subscribe(
      (users) => {
        // Filtriranje korisnika sa ulogom "Member" (bilo "Member" ili "member")
        this.projectUsers = users.filter(user => user.role.toLowerCase() === 'member');
      },
      (error) => {
        console.error('Error loading users for project:', error);
      }
    );
  }




  isManager(): boolean {
    return localStorage.getItem('role') === 'Manager';
  }
  loadPendingTasks() {
    const project = this.project as any;
    this.pendingTasks = [];
    if (this.project) {
      const projectIdStr = String(project.id);
      this.projectService.getTasks().subscribe(tasks => {
        this.pendingTasks = tasks
          .filter(task => task.status === 'pending' && String(task.project_id) === projectIdStr)
          .map(task => task.name);
      });
    }
  }

  loadActiveUsers() {
    this.userService.getActiveUsers().subscribe(
      (data) => {
        this.users = data;
      },
      (error) => {
        console.error('Error fetching active users:', error);
      }
    );
  }


  toggleSelection(user: any) {
    const index = this.selectedUsers.indexOf(user);
    if (index === -1) {
      this.selectedUsers.push(user);
    } else {
      this.selectedUsers.splice(index, 1);
    }
  }

  addSelectedUsersToProject() {
    const project = this.project as any;
    if (project && this.selectedUsers.length > 0) {
      const userIds = this.selectedUsers.map(user => user.id);

      this.projectService.addMemberToProject(project.id, userIds).subscribe(
        (response) => {
          console.log('Users successfully added:', response);
          this.isAddMemberFormVisible = false;
        },
        (error) => {
          console.error('Error adding users to project:', error);
        }
      );
    } else {
      alert('No users selected.');
    }
  }
  showCreateTaskForm() {
    const projectId = this.project as any;
    console.log(projectId);
    this.isCreateTaskFormVisible = true;
    this.cdRef.detectChanges();
    document.querySelector('#mm')?.setAttribute("style", "display:block; opacity: 100%; margin-top: 20px");
  }

  showAddMemberToProject() {
    const project = this.project as any;
    console.log(project.id);
    this.isAddMemberFormVisible=true
    this.cdRef.detectChanges();
    document.querySelector('#addMemberModal')?.setAttribute("style", "display:block; opacity: 100%; margin-top: 20px");
  }
  addMemberToProject(user: any) {
    const project = this.project as any;
    if (project && user) {
      console.log(`Dodavanje korisnika ${user.name} u projekat ${project.name}`);

      this.projectService.addMemberToProject(project.id, user.id).subscribe(
        (response) => {
          console.log('User added to project:', response);
          this.loadActiveUsers();
          this.isAddMemberFormVisible = false;
        },
        (error) => {
          console.error('Error adding user to project:', error);
        }
      );
    }
  }
  toggleUserSelection(user: any) {
    if (this.selectedUsers.includes(user)) {
      this.selectedUsers = this.selectedUsers.filter(u => u !== user);
    } else {
      this.selectedUsers.push(user);
    }
  }
  cancelCreateTask() {
    this.isCreateTaskFormVisible = false;
    this.taskName = '';
    this.taskDescription = '';
  }
  cancelAddMember() {
    this.isAddMemberFormVisible = false;
  }


  createTask() {
    if (!this.taskName || !this.taskDescription) {
      alert('Please fill in all fields before creating the task.');
      return;
    }

    if (this.projectId) {
      const projectIdStr = String(this.projectId);
      console.log('Project ID je:', projectIdStr);

      const newTask = {
        name: this.taskName,
        description: this.taskDescription
      };

      this.projectService.createTask(projectIdStr, newTask).subscribe(
        (response) => {
          console.log('Task successfully created:', response);
          this.pendingTasks.push(response.name);
          this.cancelCreateTask();
        },
        (error) => {
          console.error('Error creating task:', error);
        }
      );
    } else {
      console.error('Project ID is missing');
      alert('Project ID is missing. Could not create task.');
    }
  }




  showTaskDetails(task: any) {
    console.log("Selected task:", task);
    this.selectedTask = task;
    this.isTaskDetailsVisible = true;
    this.cdRef.detectChanges();
    document.querySelector('#taskModal')?.setAttribute("style", "display:block; opacity: 100%; margin-top:20px");

  }

  closeTaskDetails() {
    this.isTaskDetailsVisible = false;
    this.selectedTask = null;
  }


}


