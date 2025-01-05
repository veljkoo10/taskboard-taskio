import {Component, Input, SimpleChanges,ElementRef,  ChangeDetectionStrategy} from '@angular/core';
import { Project } from '../../model/project.model';
import { ProjectService } from 'src/app/services/project.service';
import { UserService } from 'src/app/services/user.service';
import { ChangeDetectorRef } from '@angular/core';
import { TaskService } from 'src/app/services/task.service'; // Import TaskService
import {AuthService} from "../../services/auth.service";
import {forkJoin} from "rxjs";
import * as d3 from 'd3';

@Component({
  selector: 'app-project-details',
  templateUrl: './project-details.component.html',
  styleUrls: ['./project-details.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
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
  selectedDependencies: any[] = [];
  existingTasks: any[] = [];
  dependencyMessage: string | null = null;
  originalStatus: string | null = null;
  taskDependencies: any[]=[];
  taskFiles: any[] = [];
  isAddDependencyModalVisible: boolean = false;
  selectedDependency: any;
  workflows: any[] = [];
  dependencyFormError: string = '';

  constructor(
    private taskService: TaskService,
    private projectService: ProjectService,
    private userService: UserService,
    private cdRef: ChangeDetectorRef,
    private authService: AuthService,
    private el: ElementRef
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
      this.loadExistingTasks();
      this.loadFlows()
      this.renderGraph();
      console.log(this.project)
      console.log(this.workflows)


    }
  }

  getAvailableSpots(): number {
    if (!this.project) {
      return 0;
    }
    return this.project.max_people - this.projectUsers.length-1;
  }


  showAddDependencyModal(): void {
    // Učitaj taskove i ažuriraj existingTasks
    this.loadExistingTasks();

    // Proveri da li su taskovi ažurirani
    console.log('Updated Existing Tasks:', this.existingTasks);

    // Postavi modal kao vidljiv
    this.isAddDependencyModalVisible = true;
    this.isTaskDetailsVisible=false
    // Osveži prikaz ako je potrebno
    this.cdRef.detectChanges();
  }

  loadExistingTasks() {
    if (!this.selectedTask || !this.selectedTask.id) {
      return;
    }

    // Ispisujemo ID selektovanog taska u konzoli
    console.log('Selected Task ID:', this.selectedTask.id);

    // Filtriramo sve zadatke tako da ne uključimo onaj koji je selektovan
    this.existingTasks = this.existingTasks.filter(task => task.id !== this.selectedTask.id);
    // Detekcija promene (ako je potrebno)
    this.cdRef.detectChanges();
  }



  trackByTaskId(index: number, task: any): string {
    return task.id; // Assumes each task has a unique id
  }


closeAddDependecyModalAll():void{
  this.isAddDependencyModalVisible = false;
  this.selectedDependencies = []; // Resetovanje selektovanih zavisnosti
  this.dependencyFormError = '';
}

// Zatvaranje modala
closeAddDependencyModal(): void {
  this.isAddDependencyModalVisible = false;
  this.isTaskDetailsVisible=true;
  this.selectedDependencies = []; // Resetovanje selektovanih zavisnosti
  this.dependencyFormError = '';
}

// Upravljanje selektovanim zavisnostima
toggleTaskDependency(task: any): void {
  const index = this.selectedDependencies.findIndex(t => t === task.id); // Tražimo ID taska u listi
  if (index === -1) {
    this.selectedDependencies.push(task.id); // Dodajemo ID zavisnosti
  } else {
    this.selectedDependencies.splice(index, 1); // Uklanjamo ID ako je već selektovan
  }
}

// Potvrda i dodavanje zavisnosti
confirmDependencies(): void {
  if (this.selectedDependencies.length === 0) {
    this.dependencyFormError = 'Please select at least one task.';
    return;
  }

  this.createWorkflow();
  this.closeAddDependencyModal();
  this.cdRef.detectChanges();
  this.renderGraph();


}

addDependency(): void {
  if (!this.selectedDependency) {
    this.dependencyFormError = 'Please select a task.';
    return;
  }

  this.selectedDependencies.push(this.selectedDependency.id);
  this.closeAddDependencyModal();
  this.renderGraph()
}

// Funkcija koja poziva createWorkflow iz TaskService
createWorkflow(): void {
  if (!this.selectedTask.id || this.selectedDependencies.length === 0) {
    console.error('Invalid data for creating workflow');
    return;
  }

  console.log(this.selectedDependencies);
  // Pozivanje funkcije createWorkflow sa trenutnim taskId i zavisnostima
  this.taskService.createWorkflow(this.selectedTask.id, this.selectedDependencies).subscribe(
    (response) => {
      console.log('Workflow created successfully: uspesno', response);


      this.loadFlows();
      this.renderGraph();
    },
    (error) => {
      console.error('Error creating workflow:', error);
    }
  );
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

    // Učitavanje zadataka i aktivnih korisnika
    this.loadTasks();
    this.loadActiveUsers();

    // Provera da li projekt postoji i ima validan ID
    if (this.project && this.project.id) {
      this.projectId = this.project.id;  // Postavi projectId
    }


    // Provera da li projekt ima title
    if (this.project && this.project.title) {
      this.getProjectIDByTitle(this.project.title);
    } else {
      console.error('Project Title is missing!');
    }

    this.loadFlows();
    this.renderGraph()
  }

  loadFlows(){
    this.taskService.getAllWorkflows().subscribe((data) => {
      this.workflows = data;

      if (!this.workflows || this.workflows.length === 0) {
        console.log('No workflows found!');
        return;
      }

      console.log(this.workflows)
      console.log(this.existingTasks)
      for (let i = 0; i < this.workflows.length; i++) {
        const taskExists = this.existingTasks.some(task => task.id === this.workflows[i].task_id);
    }

    this.renderGraph();

    });
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
        this.existingTasks = [];  // Resetovanje pre novog učitavanja

        tasks.forEach(task => {
          if (String(task.project_id) === projectIdStr) {
            switch (task.status.toLowerCase()) {
              case 'pending':
                this.pendingTasks.push(task);
                this.existingTasks.push(task);
                break;
              case 'work in progress':
                this.inProgressTasks.push(task);
                this.existingTasks.push(task);
                break;
              case 'done':
                this.doneTasks.push(task);
                this.existingTasks.push(task);
                break;
              default:
                console.warn(`Unrecognized task status: ${task.status}`);
            }
          }
        });

        // Poziv za detekciju promena u slučaju da postoji problem sa UI
        this.cdRef.detectChanges();

      }, (error) => {
        console.error('Error loading tasks:', error);
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
    this.selectedTask = task; // Ensure task is valid
    this.isAddTaskUserVisible = true; // Toggle visibility
    this.isTaskDetailsVisible = false;
    this.loadUsersForProject()
    //const C = A.filter(item => !B.i`ncludes(item));
    this.taskAvUsers = this.projectUsers.filter(
      item => !this.selectedTask.users.includes(item.id)
    );
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
    this.isCreateTaskFormVisible = true;
    this.cdRef.detectChanges();
    document.querySelector('#mm')?.setAttribute("style", "display:block; opacity: 100%; margin-top: 20px");
  }

  showAddMemberToProject() {
    const project = this.project as any;
    this.isAddMemberFormVisible=true
    this.cdRef.detectChanges();
    document.querySelector('#addMemberModal')?.setAttribute("style", "display:block; opacity: 100%; margin-top: 20px");
  }
  addMemberToProject(user: any) {
    const project = this.project as any;
    if (project && user) {

      this.projectService.addMemberToProject(project.id, user.id).subscribe(
        (response) => {
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
    this.loadTaskFiles(task.id);

  }
  loadTaskFiles(taskId: string): void {
  this.taskFiles = [];
  this.cdRef.detectChanges();
  this.taskService.getTaskFiles(taskId).subscribe(
    (files) => {
      console.log('Files loaded for task:', files);  // Proverite da li fajlovi stižu
      if (files && Array.isArray(files)) {
        this.taskFiles = files;  // Direktno dodelite niz fajlova
      } else {
        console.error('Invalid file data format');
      }
    },
    (error) => {
      console.error('Error loading task files:', error);
    }
  );
}

downloadTaskFile(taskId: string, fileNamee: string): void {
  this.taskService.downloadFile(taskId, fileNamee).subscribe(
    (blob) => {
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = fileNamee; // Postavlja ime fajla
      a.click();
      window.URL.revokeObjectURL(url); // Oslobađa memoriju nakon preuzimanja
    },
    (error) => {
      console.error('Error downloading file:', error);
    }
  );
}

  selectedFile: File | null = null;
  onFileSelected(event: any): void {
    const file = event.target.files[0];
    if (file) {
      this.selectedFile = file;
    }
  }


  uploadFile(): void {
    if (!this.selectedFile) {
      console.error('No file selected!');
      return;
    }

    if (!this.selectedTask?.id) {
      console.error('Task ID is missing.');
      return;
    }

    const formData = new FormData();
    formData.append('taskID', this.selectedTask.id);
    formData.append('file', this.selectedFile);

    this.taskService.uploadFile(formData).subscribe(
      (response) => {
        console.log('Fajl je uspešno upload-ovan!', response);
        this.message = 'Fajl je uspešno upload-ovan!';
        this.isSuccessMessage = true;
        this.cdRef.detectChanges();
        this.loadTaskFiles(this.selectedTask.id);
        // Osvežavanje stranice

        this.isTaskDetailsVisible = false;
        this.isTaskDetailsVisible = true;
      },
      (error) => {
        console.error('Greška prilikom upload-a fajla:', error);
        this.message = 'Greška prilikom upload-a fajla.';
        this.isSuccessMessage = false;

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

      this.projectService.removeMemberToProject(this.project.id, [userId]).subscribe(
        (response) => {
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
  async renderGraph() {
    const svg = d3
      .select(this.el.nativeElement)
      .select('#workflowGraph')
      .select('svg');

    if (svg.node()) {
      svg.remove(); // Brišemo postojeći SVG graf
    }

    // Kreiramo novi SVG element sa 100% dimenzijama
    const newSvg = d3
      .select(this.el.nativeElement)
      .select('#workflowGraph')
      .append('svg')
      .attr('width', '100%') // Postavljanje širine na 100%
      .attr('height', '100%'); // Postavljanje visine na 100%

    // Proveravamo da li su podaci za workflow ažurirani pre nego što kreiramo graf
    if (!this.workflows || this.workflows.length === 0) {
      return;
    }

    // Kreiramo sve čvorove (nodes)
    const nodes = Array.from(
      new Set(
        this.workflows.flatMap((workflow) => [
          workflow.task_id,
          ...workflow.dependency_task,
        ])
      )
    ).map((id) => ({
      id,
      x: Math.random() * 800,
      y: Math.random() * 600,
    }));

    // Kreiramo sve veze (links)
    const links = this.workflows.flatMap((workflow) =>
      workflow.dependency_task.map((dep: string) => ({
        source: workflow.task_id,
        target: dep,
      }))
    );

    // Kreiramo markere za strelice
    newSvg
      .append('defs')
      .append('marker')
      .attr('id', 'arrowhead')
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 15)
      .attr('refY', 0)
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', '#007bff');

    // Kreiramo simulaciju
    const simulation = d3
      .forceSimulation(nodes)
      .force('link', d3.forceLink(links).id((d: any) => d.id).distance(100))
      .force('charge', d3.forceManyBody().strength(-100))
      .force('center', d3.forceCenter(400, 300))
      .force('collide', d3.forceCollide().radius(20));

    // Kreiramo linije (links) sa strelicama
    const link = newSvg
      .append('g')
      .selectAll('line')
      .data(links)
      .enter()
      .append('line')
      .attr('stroke', '#ccc')
      .attr('stroke-width', 2)
      .attr('marker-end', 'url(#arrowhead)');

    // Kreiramo čvorove (nodes)
    const node = newSvg
      .append('g')
      .selectAll('circle')
      .data(nodes)
      .enter()
      .append('circle')
      .attr('r', 10)
      .attr('fill', '#007bff')
      .call(
        d3
          .drag<SVGCircleElement, { id: any; x: number; y: number }>()
          .on('start', (event, d) => {
            if (!event.active) simulation.alphaTarget(0.3).restart();
          })
          .on('drag', (event, d) => {
            d.x = event.x;
            d.y = event.y;
          })
          .on('end', (event, d) => {
            if (!event.active) simulation.alphaTarget(0);
          })
      );

    // Dodavanje tekstova sa asinhronim dohvatom imena
    const text = newSvg
      .append('g')
      .selectAll('text')
      .data(nodes)
      .enter()
      .append('text')
      .attr('font-size', 12)
      .attr('dx', 15)
      .attr('dy', 5)
      .text((d: any) => formatName(d.id)); // Koristimo funkciju za formatiranje naziva

    // Funkcija za formatiranje naziva (prvo slovo veliko)
    function formatName(name: string): string {
      return name.charAt(0).toUpperCase() + name.slice(1);
    }

    // Ažuriranje imena taskova
    for (const node1 of nodes) {
      try {
        const task = await this.taskService.getTaskById(node1.id).toPromise();
        const taskName = task?.name || `Task ${node1.id}`;
        text
          .filter((t: any) => t.id === node1.id)
          .text(formatName(taskName)); // Formatiramo naziv pre postavljanja
      } catch (error) {
        console.error(`Greška prilikom dohvatanja imena za ID ${node1.id}`, error);
      }
    }

    // Ažuriranje pozicija tokom simulacije
    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y);

      node.attr('cx', (d: any) => d.x).attr('cy', (d: any) => d.y);

      text.attr('x', (d: any) => d.x).attr('y', (d: any) => d.y);
    });

  }

}
