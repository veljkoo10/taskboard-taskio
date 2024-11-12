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
    const project = this.project as any;
    console.log(project.id);
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
    const project = this.project as any;

    if (!this.taskName || !this.taskDescription) {
      alert('Please fill in all fields before creating the task.');
      return;
    }

    if (project) {
      const newTask = {
        name: this.taskName,
        description: this.taskDescription
      };

      this.projectService.createTask(project.id, newTask).subscribe(
        (response) => {
          console.log('Task successfully created:', response);
          this.cancelCreateTask();
          window.location.reload();
        },
        (error) => {
          console.error('Error creating task:', error);
        }
      );
    }
  }

  showTaskDetails(task: any) {
    console.log("Selected task:", task);
    this.selectedTask = task;
    document.querySelector('#taskModal')?.setAttribute("style", "display:block; opacity: 100%; margin-top: 20px");
    this.isTaskDetailsVisible = true;
  }

  closeTaskDetails() {
    this.isTaskDetailsVisible = false;
    this.selectedTask = null;
  }


}


