import {Component, Input, SimpleChanges} from '@angular/core';
import { Project } from '../../model/project.model';
import { ProjectService } from 'src/app/services/project.service';
import { UserService } from 'src/app/services/user.service';
import { ChangeDetectorRef } from '@angular/core';
import { TaskService } from 'src/app/services/task.service'; // Import TaskService
import {AuthService} from "../../services/auth.service";
import {forkJoin} from "rxjs";

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
  users: any[] = [];
  selectedUsers: any[] = [];
  selectedTask: any = null;
  isTaskDetailsVisible: boolean = false;
  projectId: string | null = null;
  projectUsers: any[] = [];
  projectManagers: any[] = [];
  availableSlots: number = 0;
  pendingTasks: any[] = [];
  inProgressTasks: any[] = [];
  doneTasks: any[] = [];
  taskUsers: any[] = [];
  user: any;
  taskFormError: string = '';
  message: string | null = null;
  isSuccessMessage: boolean = true;
  taskAvUsers: any[] = []
  selectedDependencies: string[] = [];
  existingTasks: any[] = [];
  dependencyMessage: string | null = null;
  originalStatus: string | null = null;
  taskDependencies: any[]=[];
  constructor(
    private taskService: TaskService,
    private projectService: ProjectService,
    private userService: UserService,
    private cdRef: ChangeDetectorRef,
    private authService: AuthService
  ) {}
  ngOnChanges(changes: SimpleChanges) {
    if (changes['project'] && changes['project'].currentValue) {
      const newProject = changes['project'].currentValue;

      if (newProject.title) {
        this.getProjectIDByTitle(newProject.title);
      }

      this.resetAddMemberForm();
      this.resetCreateTaskForm();
      this.loadTasks();
      this.loadUsersForProject();
    }
  }

  getAvailableSpots(): number {
    if (!this.project) {
      return 0;
    }
    return this.project.max_people - this.projectUsers.length-1;
  }

  resetAddMemberForm() {
    this.isAddMemberFormVisible = false;
    this.selectedUsers = [];
  }

  resetCreateTaskForm() {
    this.isCreateTaskFormVisible = false;
    this.taskName = '';
    this.taskDescription = '';
  }
  ngOnInit() {
    // Dobijanje korisničkih podataka iz tokena
    this.user = this.getUserInfoFromToken();
    console.log('User Info:', this.user);

    // Učitavanje zadataka i aktivnih korisnika
    this.loadTasks();
    this.loadActiveUsers();

    // Provera da li projekt postoji i ima validan ID
    if (this.project && this.project.id) {
      this.projectId = this.project.id;  // Postavi projectId
      console.log('Project ID set to:', this.projectId);
    }


    // Provera da li projekt ima title
    if (this.project && this.project.title) {
      console.log('Project Title:', this.project.title);
      this.getProjectIDByTitle(this.project.title);
    } else {
      console.error('Project Title is missing!');
    }
  }

  getUserInfoFromToken(): any {
    const token = this.authService.getDecryptedData('access_token');
    if (token) {
      try {
        const payloadBase64 = token.split('.')[1];
        const payloadJson = atob(payloadBase64);
        return JSON.parse(payloadJson);
      } catch (error) {
        console.error('Invalid token format:', error);
        return null;
      }
    }
    return null;
  }

  isUserInProject(userId: string): boolean {
    return this.projectUsers.some(user => user.id === userId);
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
            this.loadUsersForProject();
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

  loadUsersForProject() {
    if (this.project && this.project.id) {
      this.projectService.getUsersForProject(this.project.id).subscribe(
        (users) => {
          this.projectUsers = this.sortUsersAlphabetically(users.filter(user => user.role.toLowerCase() === 'member'));
          this.projectManagers = this.sortUsersAlphabetically(users.filter(user => user.role.toLowerCase() === 'manager'));
          this.loadActiveUsers();
        },
        (error) => {
          console.error('Error loading users for project:', error);
        }
      );
    }
  }


  isManager(): boolean {
    return this.authService.getDecryptedData('role') === 'Manager';
  }
  loadTasks() {
    if (this.project) {
      const projectIdStr = String(this.project.id);
      this.taskService.getTasks().subscribe(tasks => {

        this.pendingTasks = [];
        this.inProgressTasks = [];
        this.doneTasks = [];

        tasks.forEach(task => {

          if (String(task.project_id) === projectIdStr) {
            switch (task.status.toLowerCase()) {
              case 'pending':
                this.pendingTasks.push(task);
                break;
              case 'work in progress':
                this.inProgressTasks.push(task);
                break;
              case 'done':
                this.doneTasks.push(task);
                break;
              default:
                console.warn(`Unrecognized task status: ${task.status}`);
            }
          }
        });

        console.log('Pending Tasks:', this.pendingTasks);
        console.log('In Progress Tasks:', this.inProgressTasks);
        console.log('Done Tasks:', this.doneTasks);
      });
    }
  }

  loadActiveUsers() {
    this.userService.getActiveUsers().subscribe(
      (data) => {
        this.users = this.sortUsersAlphabetically(
          data.filter(user => !this.isUserInProject(user.id))
        );
      },
      (error) => {
        console.error('Error fetching active users:', error);
      }
    );
  }



  isSelected(user: any): boolean {
    return this.selectedUsers.includes(user);
  }

  toggleSelection(user: any) {
    const index = this.selectedUsers.indexOf(user);
    if (index === -1) {
      this.selectedUsers.push(user);
    } else {
      this.selectedUsers.splice(index, 1);
    }
    this.cdRef.detectChanges();
  }
  sortUsersAlphabetically(users: any[]): any[] {
    return users.sort((a, b) => a.username.localeCompare(b.username));
  }

  closeTasksDoneModal(){
    document.querySelector('.all-tasks-done-modal')?.setAttribute("style", "display:none; opacity: 100%; margin-top: 20px")
  }

  showTasksDoneModal(){
    document.querySelector('.all-tasks-done-modal')?.setAttribute("style", "display:flex; opacity: 100%; margin-top: 20px")
  }

  addSelectedUsersToProject() {
    const project = this.project as any;

    if (project && this.selectedUsers.length > 0) {

      // Provera da li je projekat aktivan pre nego što dodate korisnike
      this.projectService.isProjectActive(project.id).subscribe(
        (isActive) => {
          if (!isActive) {
            this.showTasksDoneModal();
            this.selectedUsers = [];
            return;
          }

          // Ako je projekat aktivan, nastavljamo sa dodavanjem korisnika
          const userIds = this.selectedUsers.map(user => user.id);

          this.projectService.addMemberToProject(project.id, userIds).subscribe(
            (response) => {
              console.log('Users successfully added:', response);
              this.loadUsersForProject();
              this.loadActiveUsers();
              this.selectedUsers = [];
              this.availableSlots = this.getAvailableSpots();
            },
            (error) => {
              this.showMaxPeople();
              console.error('Error adding users to project:', error);
            }
          );
        },
        (error) => {
          console.error('Error while checking project status', error);
        }
      );
    } else {
      this.showNoUsersProjectSelectedModal();
    }
  }



  updateTaskStatus(status: string) {
    if (this.selectedTask) {
      const taskId = this.selectedTask.id;
      const userId = this.user.id;

      // Proveri da li je korisnik menadžer
      if (this.user.role === 'Manager') {
        this.dependencyMessage = 'Managers are not allowed to update task status.';
        return; // Prekida izvršavanje ako je korisnik menadžer
      }

      // Proveri da li je korisnik član taska
      this.taskService.isUserOnTask(taskId, userId).subscribe(
        (isMember) => {
          if (isMember) {
            // Ako je korisnik član, ažuriraj status
            this.selectedTask.status = status;

            // Pozivanje servisa za ažuriranje statusa
            this.taskService.updateTaskStatus(taskId, status).subscribe(
              (response) => {
                console.log('Task status updated successfully:', response);
                this.loadTasks();
                // Resetovanje poruke o zavisnostima
                this.dependencyMessage = null;
              },
              (error) => {
                console.error('Error updating task status:', error);

                // Prikazivanje lepše greške u slučaju HTTP greške 400
                if (error.status === 400) {
                  const errorMessage = error.error?.message || 'First, it updates the status of the main task';
                  this.dependencyMessage = `Error: ${errorMessage}`;
                } else {
                  // Prikazivanje opšte greške za ostale status kodove
                  this.dependencyMessage = 'An unexpected error occurred. Please try again later.';
                }
              }
            );
          } else {
            this.dependencyMessage = 'You are not a member of this task and cannot update its status.';
          }
        },
        (error) => {
          console.error('Error checking task membership:', error);
          this.dependencyMessage = 'An error occurred while checking task membership.';
        }
      );
    }
  }


  isStatusDisabled(status: string): boolean {
    if (this.selectedTask) {
      if (status === 'done' && this.selectedTask.status !== 'work in progress') {
        // Može da postavi "done" samo ako je "work in progress"
        return true;
      }
      if (status === 'work in progress' && this.selectedTask.status !== 'pending') {
        // Može da postavi "work in progress" samo ako je "pending"
        return true;
      }
    }
    return false;
  }

  isAddTaskUserVisible: boolean = false;
  selectedTaskUsers: any[] = []; // Stores selected users for the task

  showAddTaskUserModal(task: any) {
    console.log('Opening Add Task User Modal:', task);
    this.selectedTask = task; // Ensure task is valid
    this.isAddTaskUserVisible = true; // Toggle visibility
    this.isTaskDetailsVisible = false;
    this.loadUsersForProject()
    //const C = A.filter(item => !B.i`ncludes(item));
    console.log(this.selectedTask.users)
    this.taskAvUsers = this.projectUsers.filter(
      item => !this.selectedTask.users.includes(item.id)
    );
    console.log(this.taskAvUsers)
  }

  closeAddTaskUserModal() {
    this.isAddTaskUserVisible = false;
    this.selectedTaskUsers = [];
    this.isTaskDetailsVisible=true;
    this.message = null;

  }

  // Toggle User Selection for Task
  toggleTaskUserSelection(user: any) {
    const index = this.selectedTaskUsers.indexOf(user);
    if (index === -1) {
      this.selectedTaskUsers.push(user);
    } else {
      this.selectedTaskUsers.splice(index, 1);
    }
  }

  addSelectedUsersToTask() {
    if (this.selectedTask && this.selectedTaskUsers.length > 0) {
    const taskId = this.selectedTask.id;
    const userIds = this.selectedTaskUsers.map(user => user.id);
    const username = this.selectedTaskUsers.map(user => user.username)

    userIds.forEach(userId => {
      // Proveri da li je korisnik već dodeljen ovom tasku
      const isAlreadyAssigned = this.taskUsers.some(user => user.id === userId);

      if (!isAlreadyAssigned) {
        this.taskService.addUserToTask(taskId, userId).subscribe(
          response => {
            console.log(`User ${userId} added to task ${taskId}:`, response);

            // Ažuriraj lokalni niz taskUsers
            const addedUser = this.taskAvUsers.find(user => user.id === userId);
            if (addedUser) {
              // Dodaj korisnika u taskUsers
              this.taskUsers.push(addedUser);
              // Ukloni korisnika iz projectUsers
              this.taskAvUsers = this.taskAvUsers.filter(user => user.id !== userId);
            }
          },
          error => {
            console.error(`Error adding user ${userId} to task ${taskId}:`, error);
            this.showAddUserErrorModal();
          }
        );
      } else {
          this.showUserAlreadyAddedModal();
      }
    });

  } else {
      this.showNoUsersSelectedModal();
  }
}

  showNoUsersSelectedModal() {
    const modal = document.querySelector('.no-users-selected-modal');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }

  closeNoUsersSelectedModal() {
    const modal = document.querySelector('.no-users-selected-modal');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0;');
    }
  }

  showAddUserErrorModal() {
    const modal = document.querySelector('.add-user-error-modal');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }

  closeAddUserErrorModal() {
    const modal = document.querySelector('.add-user-error-modal');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0;');
    }
  }
  showUserAlreadyAddedModal() {
    const modal = document.querySelector('.user-already-added-modal');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }

  closeUserAlreadyAddedModal() {
    const modal = document.querySelector('.user-already-added-modal');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0;');
    }
  }
  showNoUsersProjectSelectedModal() {
    const modal = document.querySelector('.no-users-selected-modal-project');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }

  closeNoUsersProjectSelectedModal() {
    const modal = document.querySelector('.no-users-selected-modal-project');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0;');
    }
  }


  showCreateTaskForm() {
    const projectId = this.project as any;
    this.loadTasksDepend();
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

  cancelCreateTask() {
    this.isCreateTaskFormVisible = false;
    this.taskName = '';
    this.taskDescription = '';
    this.taskFormError = '';

  }
  cancelAddMember() {
    this.isAddMemberFormVisible = false;
  }


  createTask() {
    if (!this.taskName || !this.taskDescription) {
      this.taskFormError = 'Please fill in all fields before creating the task.';
      return;
    }

    if (this.projectId) {
      const newTask = {
        name: this.taskName,
        description: this.taskDescription,
        dependsOn: this.selectedDependencies, // Dodaj zavisnosti
      };

      this.projectService.createTask(this.projectId, newTask).subscribe(
        (response) => {
          console.log('Task successfully created:', response);

          this.cancelCreateTask();
          this.taskFormError = '';

          this.loadTasks()
        },
        (error) => {
          console.error('Error creating task:', error);
        }
      );
    } else {
      console.error('Project ID is missing');
    }
  }

  showMissingProjectIdModal() {
    const modal = document.querySelector('.exist-task-name');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 1;');
    }
  }
  closeMissingProjectIdModal() {
    const modal = document.querySelector('.exist-task-name');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0;');
    }
  }

  showTaskDetails(task: any) {
    this.dependencyMessage = null;

    // Proveri da li je korisnik član taska
    this.taskService.isUserOnTask(task.id, this.user.id).subscribe(
      (isMember) => {
        if (isMember) {
          document.querySelector('#status-block')?.setAttribute('style', "display: block");
        } else {
          document.querySelector('#status-block')?.setAttribute('style', "display: none");
          // Ako korisnik nije član, prikaži alert
          //alert('You are not a member of this task and cannot update its status.');
        }
      },
      (error) => {
        console.error('Error checking task membership:', error);
        alert('An error occurred while checking task membership.');
      }
    );

    console.log("Selected task:", task);
    this.selectedTask = task;
    this.isTaskDetailsVisible = true;
    this.selectedTask = { ...task };
    this.originalStatus = task.status;
    this.cdRef.detectChanges();
    document.querySelector('#taskModal')?.setAttribute('style', 'display:block; opacity: 100%; margin-top:20px');

    // Učitajte korisnike na tasku
    this.taskService.getUsersForTask(task.id).subscribe(
      (users) => {
        this.taskUsers = this.sortUsersAlphabetically(users);
      },
      (error) => {
        console.error('Error loading users for task:', error);
      }
    );
  }

  removeUserFromTask(userId: string): void {
    if (!this.selectedTask?.id) {
      console.error('Task ID is missing.');
      return;
    }

    if(this.selectedTask.status !== "done"){
    this.taskService.removeUserFromTask(this.selectedTask.id, userId).subscribe(
      (response) => {
        console.log('User removed from task successfully:', response);

        // Ukloni korisnika iz lokalne liste korisnika na tasku
        const removedUser = this.taskUsers.find(user => user.id === userId);
        this.taskUsers = this.taskUsers.filter(user => user.id !== userId);

        // Vraćanje korisnika u listu dostupnih korisnika na projektu
        if (removedUser) {
          this.taskAvUsers.push(removedUser);
          this.taskAvUsers = this.sortUsersAlphabetically(this.taskAvUsers);
        }
      },
      (error) => {
        console.error('Error removing user from task:', error);
      }
    );
  }
    else{
      this.message = "Cant remove member from done task!";
      this.isSuccessMessage = false;

    }
  }

  closeAddMember() {
    this.isAddMemberFormVisible = false;
    this.selectedUsers = [];
  }
  closeCreateTask(){
    this.isCreateTaskFormVisible=false;
  }

  closeTaskDetails() {
    this.isTaskDetailsVisible = false;
    this.selectedTask = null;
    this.dependencyMessage = null;
    if (this.selectedTask) {
      this.selectedTask.status = this.originalStatus; // Vraća status na originalnu vrednost
    }

  }
  closeAddUserToTask(){
    this.isAddTaskUserVisible=false;
    this.selectedUsers=[];
  }
  cancelAddTaskUserModal(){
    this.isTaskDetailsVisible = false;
  }

  removeUserFromProject(userId: string): void {
    if (!this.project?.id) {
      console.error('Project ID is missing.');
      return;
    }

    if(this.project.min_people <= this.projectUsers.length ){
      console.log(this.projectUsers.length - 1)

      this.projectService.removeMemberToProject(this.project.id, [userId]).subscribe(
        (response) => {
          console.log('User removed successfully:', response);
          // Uklanjanje korisnika iz lokalne liste korisnika na projektu
          this.projectUsers = this.projectUsers.filter(user => user.id !== userId);

          // Ažuriraj project.users ručno
          if (this.project?.users) {
            this.project.users = this.project.users.filter(user => user.id !== userId);
          }
          // Osvežavanje liste dostupnih korisnika
          this.loadActiveUsers();
        },
        (error) => {
          console.error('Error removing user:', error);
        }
      );
    }
    else{
      console.log(this.project.users.length )
      this.showDeletePeopleProjectModal();  // Show modal instead of alert
    }
  }
  showDeletePeopleProjectModal() {
    const modal = document.querySelector('.delete-people-project-modal');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }
  closeDeletePeopleProjectModal() {
    const modal = document.querySelector('.delete-people-project-modal');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0;');
    }
  }


  isProjectActive(): boolean {

    if(this.pendingTasks.length === 0 && this.inProgressTasks.length === 0 && this.doneTasks.length !== 0){
      return false;
    }
    else{
      return true
    }
  }

  closeMaxPeople(){
    this.selectedUsers = [];

    document.querySelector(".max-people-error-modal")?.setAttribute("style", "display:none; opacity: 100%; margin-top: 20px")
  }

  showMaxPeople(){
    document.querySelector(".max-people-error-modal")?.setAttribute("style", "display:flex; opacity: 100%; margin-top: 20px")
  }
  loadTasksDepend() {
    if (!this.projectId) {
      console.error('Project ID is missing!');
      return;
    }

    // Resetujemo postojeće zadatke pre nego što učitamo nove
    this.existingTasks = [];

    console.log('Fetching tasks for project ID:', this.projectId);

    // Pozivamo servis za učitavanje ID-ova zadataka vezanih za trenutni projekat
    this.taskService.getTasksByProjectId(this.projectId).subscribe(
      (taskIds) => {
        if (taskIds && taskIds.length > 0) {
          // Pozivamo servis za detalje zadatka prema ID-ovima
          const taskDetailsRequests = taskIds.map((taskId) =>
            this.taskService.getTaskById(taskId)
          );

          // Paralelno izvršavamo sve API pozive za detalje zadataka
          forkJoin(taskDetailsRequests).subscribe(
            (taskDetails) => {
              // Mapiranje i filtriranje samo onih zadataka koji pripadaju trenutnom projektu
              this.existingTasks = taskDetails.map((task) => ({
                id: task.id,
                name: task.name,
              }));
              console.log('Tasks loaded:', this.existingTasks);
              this.cdRef.detectChanges();
            },
            (error) => {
              console.error('Error fetching task details:', error);
            }
          );
        } else {
          console.log('No tasks found for this project.');
        }
      },
      (error) => {
        console.error('Error loading task IDs:', error);
      }
    );
  }


}


