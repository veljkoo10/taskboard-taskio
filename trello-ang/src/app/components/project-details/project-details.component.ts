import {Component, Input, SimpleChanges, ElementRef, ViewChild} from '@angular/core';
import { Project } from '../../model/project.model';
import { ProjectService } from 'src/app/services/project.service';
import { UserService } from 'src/app/services/user.service';
import { ChangeDetectorRef } from '@angular/core';
import { TaskService } from 'src/app/services/task.service'; // Import TaskService
import {AuthService} from "../../services/auth.service";
import {catchError, EMPTY, forkJoin, Observable, switchMap, tap, throwError} from "rxjs";
import * as d3 from 'd3';
import {ConsoleLogger} from "@angular/compiler-cli";
import { CdkDragDrop, moveItemInArray, transferArrayItem } from '@angular/cdk/drag-drop';
import {Router} from "@angular/router";

@Component({
  selector: 'app-project-details',
  templateUrl: './project-details.component.html',
  styleUrls: ['./project-details.component.css'],
})
export class ProjectDetailsComponent {
  @Input() project: Project | null = null;
  isFileExistsModalVisible: boolean = false;
  errorMessage: string = '';
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
  @ViewChild('fileInput') fileInput!: ElementRef; // Referenca na input element
  oversizedFiles: string[] = [];
  isOversizedFileModalVisible = false;
  selectedFiles: File[] = [];
  isAddTaskUserVisible: boolean = false;
  selectedTaskUsers: any[] = [];
  draggedTask: any = null;
  sourceList: string = '';
  isLoadingDependencies: boolean = false;
  selectedSessionTasks: number[] = [];
  allTasks: any[] = [];
  isProjectDeleted: boolean = false;
  isDeleteProjectModalVisible: boolean = false;
  isDeleteSuccessModalVisible: boolean = false;
  taskUpdateOrder: Map<string, number> = new Map();
  updateCounter: number = 0; // Brojač za praćenje redosleda

  constructor(
    private taskService: TaskService,
    private projectService: ProjectService,
    private userService: UserService,
    private cdRef: ChangeDetectorRef,
    private authService: AuthService,
    private el: ElementRef,
    private router: Router
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
      this.loadFlows();
      this.renderGraph();


    }
  }

  hasButtons(): boolean {
    return this.isManager();
  }

  getAvailableSpots(): number {
    if (!this.project) {
      return 0;
    }
    return this.project.max_people - this.projectUsers.length-1;
  }

  showAddDependencyModal(): void {
    // Osiguraj da dependsOn bude niz
    if (!Array.isArray(this.selectedTask.dependsOn)) {
      this.selectedTask.dependsOn = []; // Inicijalizuj kao prazan niz
    }

    // Resetuj selekcije za trenutnu sesiju
    this.selectedSessionTasks = [...this.selectedTask.dependsOn];

    this.loadExistingTasks();
    this.isLoadingDependencies = true;

    // Pozovi API za učitavanje zavisnosti
    this.taskService.getTaskDependencies(this.selectedTask.id).subscribe(
      (dependencies) => {
        // Proveri validnost zavisnosti
        if (dependencies && Array.isArray(dependencies)) {
          this.selectedDependencies = dependencies; // Postavi zavisnosti ako su validne
          console.log('Selektovane zavisnosti:', this.selectedDependencies);
        } else {
          this.selectedDependencies = []; // Postavi prazne zavisnosti ako API vrati nevalidne podatke
          console.error('Zavisnosti nisu validne:', dependencies);
        }

        // Zatvori indikator učitavanja
        this.isLoadingDependencies = false;

        // Prikaži modal i sakrij detalje taska
        this.isAddDependencyModalVisible = true;
        this.isTaskDetailsVisible = false;
      },
      (error) => {
        // Obradi grešku
        console.error('Greška prilikom učitavanja zavisnosti:', error);

        // Postavi prazne zavisnosti i zatvori indikator učitavanja
        this.selectedDependencies = [];
        this.isLoadingDependencies = false;

        // Prikaži korisniku poruku o grešci
        this.dependencyFormError = 'Došlo je do greške prilikom učitavanja zavisnosti. Pokušajte ponovo.';

        // Sakrij modal
        this.isAddDependencyModalVisible = false;
      }
    );
  }

  loadExistingTasks(): void {
    if (!this.selectedTask || !this.selectedTask.id) {
      return;
    }

    // Resetuj listu zadataka na osnovu svih zadataka
    this.existingTasks = [...this.allTasks];

    // Filtriraj listu da ne uključuje selektovani zadatak
    this.existingTasks = this.existingTasks.filter(task => task.id !== this.selectedTask.id);

    console.log('Selektovani Task ID:', this.selectedTask.id);
    console.log('Filtrirani zadaci:', this.existingTasks);

    // Osveži prikaz ako je potrebno
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
    console.log('Pre selekcije - session:', this.selectedSessionTasks);
    console.log('Pre selekcije - dependencies:', this.selectedDependencies);

    const taskIndexInSession = this.selectedSessionTasks.indexOf(task.id);
    const taskIndexInDependencies = this.selectedDependencies.indexOf(task.id);

    // Task nije u zavisnostima (nije validiran da bude dodat)
    if (taskIndexInDependencies === -1) {
      if (taskIndexInSession === -1) {
        // Ako task nije selektovan, dodaj ga u sesiju
        this.selectedSessionTasks.push(task.id);
        console.log(`Task ${task.id} dodat u selectedSessionTasks`);
      } else {
        // Ako je task selektovan, ukloni ga iz sesije
        this.selectedSessionTasks.splice(taskIndexInSession, 1);
        console.log(`Task ${task.id} uklonjen iz selectedSessionTasks`);
      }
    } else {
      // Task je u zavisnostima, ali validacija sprečava da bude selektovan
      console.log(`Task ${task.id} je u zavisnostima, ali validacija sprečava dodavanje.`);

      // Omogućiti uklanjanje iz sesije i dalje
      if (taskIndexInSession !== -1) {
        this.selectedSessionTasks.splice(taskIndexInSession, 1);
        console.log(`Task ${task.id} uklonjen iz selectedSessionTasks`);
      }
    }

    console.log('Nakon selekcije - session:', this.selectedSessionTasks);
    console.log('Nakon selekcije - dependencies:', this.selectedDependencies);
  }

  // Potvrda i dodavanje zavisnosti
  confirmDependencies(): void {
    console.log('Zavisnosti pre potvrde:', this.selectedSessionTasks);

    // Proveri da li su zavisnosti selektovane
    if (this.selectedSessionTasks.length === 0) {
      this.dependencyFormError = 'Please select at least one task.';
      return; // Ako nema selektovanih zadataka, nemoj zatvoriti modal
    }

    // Sinhronizuj zavisnosti sa trenutnim sesijama
    this.selectedDependencies = [...this.selectedSessionTasks];

    // Pozivaj createWorkflow funkciju, koja može generisati grešku
    this.createWorkflow();

    // Dodaj timeout ili proveru da li je greška ažurirana
    setTimeout(() => {
      if (this.dependencyFormError) {
        return; // Nemoj zatvoriti modal ako postoji greška
      }

      // Ako nema greške, pozovi funkciju koja zatvara modal
      this.closeAddDependencyModal();
      this.cdRef.detectChanges();
      this.renderGraph();

      console.log('Potvrđene zavisnosti:', this.selectedDependencies);
    }, 500);
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
    if (!this.selectedTask.id || this.selectedDependencies.length === 0 || !this.projectId) {
      console.error('Invalid data for creating workflow');
      this.dependencyFormError = 'Invalid data for creating workflow.';
      return;
    }

    console.log(this.selectedDependencies);

    this.taskService.createWorkflow(this.selectedTask.id, this.selectedDependencies, this.projectId).subscribe(
      (response) => {
        console.log('Workflow created successfully: uspesno', response);
        this.dependencyFormError = ''; // Očistite grešku na uspešno kreiranje
        this.loadFlows();
        this.renderGraph();
      },
      (error) => {
        if (error.status === 400 && error.error?.message) {
          this.dependencyFormError = error.error.message; // Postavite grešku sa backend-a
        } else {
          this.dependencyFormError = 'It is not possible to create a workflow because it is entering a cycle';
        }
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
    this.loadFlows();
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
    this.renderGraph()
  }

  loadFlows() {
    // Proveri da li postoji objekat project i uzmi njegov ID
    if (this.project) {
      const projectIdStr = String(this.project.id);

      // Pozivanje servisa sa projectIdStr
      this.taskService.getWorkflowByProjectId(projectIdStr).subscribe((data) => {
        this.workflows = data;
        this.renderGraph();

        if (!this.workflows || this.workflows.length === 0) {
          return;
        }

        console.log('Loaded workflows:', this.workflows);
        console.log('Existing tasks:', this.existingTasks);

        for (let i = 0; i < this.workflows.length; i++) {
          const workflow = this.workflows[i];

          // Ispis ID-ja taska iz workflow-a
          console.log(`Workflow task_id: ${workflow.task_id}`);
        }

      }, (error) => {
        console.error('Error loading workflows:', error);
      });
    }
  }

  getUserInfoFromToken(): any {
    const token = localStorage.getItem('access_token');
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
    return localStorage.getItem('role') === 'Manager';
  }

  loadTasks() {
    if (this.project) {
      const projectIdStr = String(this.project.id);
      this.taskService.getTasks().subscribe(tasks => {
        this.pendingTasks = [];
        this.inProgressTasks = [];
        this.doneTasks = [];
        this.existingTasks = [];  // Resetovanje pre novog učitavanja

        // Čuvanje svih zadataka (samo za kasniji reset i filtriranje)
        this.allTasks = tasks.filter(task => String(task.project_id) === projectIdStr);

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
        const filteredUsers = data.filter(user => !this.isUserInProject(user.id));
        this.users = this.sortUsersAlphabetically(filteredUsers);
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
              this.users = [...this.users]; // Reset za Angular detekciju promena
              this.projectUsers = [...this.projectUsers]; // Reset za Angular detekciju promena
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

  trackByUserId(index: number, user: any): number {
    return user.id; // Unikatan identifikator za svakog korisnika
  }

  onStatusChange(): void {
    const status = this.selectedTask.status;
    this.updateTaskStatus(status)?.subscribe({
      next: () => {
        console.log('Status successfully updated to', status);
      },
      error: (err: any) => {
        console.error('Error updating status:', err);
      }
    });
  }

  updateTaskStatus(status: string): Observable<any> {
    const task = this.selectedTask || this.draggedTask;
    if (task) {
      const taskId = task.id;
      const userId = this.user.id;

      if (this.user.role === 'Manager') {
        this.dependencyMessage = 'Managers are not allowed to update task status.';
        return EMPTY; // Uvek vraćamo Observable
      }

      return this.taskService.isUserOnTask(taskId, userId).pipe(
        switchMap((isMember) => {
          if (isMember) {
            return this.taskService.updateTaskStatus(taskId, status).pipe(
              tap(() => {
                this.loadTasks();
                this.dependencyMessage = null;
              })
            );
          } else {
            this.dependencyMessage = 'You are not a member of this task and cannot update its status.';
            return EMPTY; // Uvek vraćamo Observable
          }
        }),
        catchError((error) => {
          this.dependencyMessage = 'You cannot update the status of a task unless the tasks it depends on have been updated.';
          return throwError(() => error); // Prosljeđuje Observable greške
        })
      );
    }

    return EMPTY; // Uvek vraćamo Observable
  }




  showAddTaskUserModal(task: any) {
    this.selectedTask = task; // Osiguranje da je task validan
    this.isAddTaskUserVisible = true; // Pokaži modal
    this.isTaskDetailsVisible = false; // Sakrij detalje zadatka

    // Učitaj korisnike zadatka svaki put kad se modal otvori
    this.taskService.getUsersForTask(task.id).subscribe(
      (users) => {
        this.taskUsers = this.sortUsersAlphabetically(users);

        // Ažuriraj dostupne korisnike (taskAvUsers)
        this.taskAvUsers = this.projectUsers.filter(
          (item) => !users.some((user: any) => user.id === item.id)
        );
      },
      (error) => {
        console.error('Error loading users for task:', error);
      }
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

      userIds.forEach(userId => {
        // Proveri da li je korisnik već dodeljen ovom tasku
        const isAlreadyAssigned = this.taskUsers.some(user => user.id === userId);

        if (!isAlreadyAssigned) {
          this.taskService.addUserToTask(taskId, userId).subscribe(
            response => {
              // Ažuriraj listu korisnika nakon uspešnog dodavanja
              this.taskService.getUsersForTask(taskId).subscribe(
                (updatedUsers) => {
                  this.taskUsers = this.sortUsersAlphabetically(updatedUsers);
                  console.log('Updated task users:', updatedUsers);

                  // Ažuriraj dostupne korisnike (taskAvUsers)
                  this.taskAvUsers = this.projectUsers.filter(
                    (item) => !updatedUsers.some((user: any) => user.id === item.id)
                  );
                },
                (error) => {
                  console.error('Error refreshing users after adding:', error);
                }
              );
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

          this.loadTasks();
        },
        (error) => {
          console.error('Error creating task:', error);

          if (error.status === 500) {
            this.taskFormError = 'Task with this name already exists.';
          } else {
            this.taskFormError = 'An error occurred while creating the task.';
          }
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
          document.querySelector('#fileInput')?.setAttribute('style', "display: block");
        } else {
          document.querySelector('#status-block')?.setAttribute('style', "display: none");
          document.querySelector('#fileInput')?.setAttribute('style', "display: none");

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


    this.loadTaskFiles(task.id);
  }

  loadTaskFiles(taskId: string): void {
    this.taskFiles = [];
    this.cdRef.detectChanges();
    this.taskService.getTaskFiles(taskId).subscribe(
      (files) => {
        if (files && Array.isArray(files) && files.length > 0) {
          this.taskFiles = files;
        } else {
          this.taskFiles = [];
        }
        this.cdRef.detectChanges();
      },
      (error) => {
        this.taskFiles = [];
      }
    );
  }

  formatFileName(fileName: string): string {
    if (!fileName) return '';
    return fileName.charAt(0).toUpperCase() + fileName.slice(1);
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

  onFilesSelected(event: any): void {
    const files = event.target.files;
    const MAX_FILE_SIZE_MB = 5;
    const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024;

    if (files && files.length > 0) {
      const oversizedFiles = [];
      for (let i = 0; i < files.length; i++) {
        if (files[i].size > MAX_FILE_SIZE_BYTES) {
          oversizedFiles.push(files[i].name);
        }
      }

      if (oversizedFiles.length > 0) {
        this.oversizedFiles = oversizedFiles;
        this.isOversizedFileModalVisible = true;
        this.selectedFiles = [];
        this.isTaskDetailsVisible = false;
        return;
      }

      this.selectedFiles = Array.from(files);
    }
  }

  closeOversizedFileModal(): void {
    this.isOversizedFileModalVisible = false;
    this.isTaskDetailsVisible = true;
  }



  uploadFiles(): void {
    if (!this.selectedFiles || this.selectedFiles.length === 0) {
      console.error('No files selected!');
      this.message = 'Niste odabrali fajlove za upload.';
      this.isSuccessMessage = false;
      return;
    }

    if (!this.selectedTask?.id) {
      console.error('Task ID is missing.');
      this.message = 'ID taska nije pronađen.';
      this.isSuccessMessage = false;
      return;
    }

    const taskId = this.selectedTask.id;
    const userId = this.user.id;

    // Proveri da li je korisnik član taska
    this.taskService.isUserOnTask(taskId, userId).subscribe(
      (isMember) => {
        if (isMember) {
          // Ako je korisnik član, pripremi i pošalji fajlove
          const formData = new FormData();
          formData.append('taskID', taskId);

          this.selectedFiles.forEach((file) => {
            formData.append('file', file);
          });

          // Pozivanje servisa za upload fajlova
          this.taskService.uploadFile(formData).subscribe(
            (response) => {
              console.log('Fajlovi su uspešno upload-ovani!', response);
              this.isSuccessMessage = true;

              // Resetovanje UI
              this.cdRef.detectChanges();
              this.loadTaskFiles(taskId);

              this.fileInput.nativeElement.value = '';
              this.selectedFiles = [];
            },
            (error) => {
              console.error('Error uploading files:', error);

              // Provera da li je greška 409 (Conflict)
              if (error.status === 409) {
                this.errorMessage = 'This file already exists.'; // Postavi poruku o grešci
                this.isFileExistsModalVisible = true; // Prikaži modal
              } else {
                // Prikazivanje generičke greške
                this.message = 'Greška prilikom upload-a fajlova. Pokušajte ponovo.';
                this.isSuccessMessage = false;
              }
            }
          );
        } else {
          // Poruka ako korisnik nije član taska
          console.error('User is not a member of the task.');
          this.message = 'Niste član ovog taska i nemate dozvolu za upload fajlova.';
          this.isSuccessMessage = false;
        }
      },
      (error) => {
        console.error('Error checking task membership:', error);
        this.message = 'Došlo je do greške prilikom provere članstva u tasku.';
        this.isSuccessMessage = false;
      }
    );
  }

  closeFileExistsModal(): void {
    this.errorMessage = '';
    this.fileInput.nativeElement.value = '';
    this.selectedFiles = [];
    this.isFileExistsModalVisible = false;

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
    console.log("OBRISOOOO")
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
  showUserNotOnTaskModal() {
    const modal = document.querySelector('.user-not-on-task-modal');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }

  closeUserNotOnTaskModal() {
    const modal = document.querySelector('.user-not-on-task-modal');
    if (modal) {
      modal.setAttribute('style', 'display: none; opacity: 0;');
    }
  }

  showErrorModal() {
    const modal = document.querySelector('.update-error-modal');
    if (modal) {
      modal.setAttribute('style', 'display: flex; opacity: 100%;');
    }
  }

  closeErrorModal() {
    const modal = document.querySelector('.update-error-modal');
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
        ]))
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
      })))

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
            // Ograničavamo poziciju čvora koji se prevlači
            const margin = 20;
            const maxX = (newSvg.node()?.getBoundingClientRect().width || 800) - margin;
            const maxY = (newSvg.node()?.getBoundingClientRect().height || 600) - margin;

            // Ažuriramo pozicije tako da čvorovi ostanu unutar grafičkog prostora
            d.x = Math.min(Math.max(event.x, margin), maxX);
            d.y = Math.min(Math.max(event.y, margin), maxY);

            // Ograničavanje svih čvorova tokom prevlačenja
            nodes.forEach((node) => {
              node.x = Math.min(Math.max(node.x, margin), maxX);
              node.y = Math.min(Math.max(node.y, margin), maxY);
            });
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

    // Ažuriranje imena taskova koristeći Promise.all
    try {
      await Promise.all(
        nodes.map(async (node1) => {
          try {
            const task = await this.taskService.getTaskById(node1.id).toPromise();
            const taskName = task?.name || `Task ${node1.id}`;
            text
              .filter((t: any) => t.id === node1.id)
              .text(formatName(taskName)); // Formatiramo naziv pre postavljanja
          } catch (error) {
            console.error(`Greška prilikom dohvatanja imena za ID ${node1.id}`, error);
          }
        })
      );
    } catch (error) {
      console.error('Greška u Promise.all:', error);
    }

    // Ažuriranje pozicija tokom simulacije
    simulation.on('tick', () => {
      // Ograničavanje svih čvorova tokom simulacije
      const margin = 20;
      const maxX = (newSvg.node()?.getBoundingClientRect().width || 800) - margin;
      const maxY = (newSvg.node()?.getBoundingClientRect().height || 600) - margin;

      nodes.forEach((d) => {
        d.x = Math.min(Math.max(d.x, margin), maxX);
        d.y = Math.min(Math.max(d.y, margin), maxY);
      });

      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y);

      node.attr('cx', (d: any) => d.x).attr('cy', (d: any) => d.y);

      text.attr('x', (d: any) => d.x).attr('y', (d: any) => d.y);
    });
  }

  onDragStart(event: DragEvent, task: any, source: string) {
    // Služi samo za postavljanje podataka za dragovanje
    this.draggedTask = task;
    this.sourceList = source;

    // Postavljanje podataka za dragovanje
    event.dataTransfer?.setData('text', JSON.stringify(task));

    // Prikazivanje ID-a i statusa zadatka
    console.log(`Task ID: ${task.id}, Status: ${source}`);
  }

  allowDrop(event: DragEvent) {
    event.preventDefault();
  }


  onDrop(event: DragEvent, targetList: string) {
    event.preventDefault();

    if (!this.draggedTask || this.sourceList === targetList) {
      return;
    }


    const userId = this.user.id;

    this.taskService.isUserOnTask(this.draggedTask.id, userId).subscribe(
      (isMember) => {
        this.moveTaskToTargetList(targetList);

        if (!isMember) {
          this.dependencyMessage =
            'You are not a member of this task and cannot keep it in the new list. Returning task in 2 seconds.';

          setTimeout(() => {
            this.removeTaskFromTargetList(targetList);
            this.restoreTaskToOriginalPosition();
            this.dependencyMessage = '';
            this.showUserNotOnTaskModal();
          }, 500);

          return;
        }

        const updateObservable = this.updateTaskStatus(targetList);
        if (updateObservable) {
          updateObservable.subscribe(
            () => {
              console.log(`Task ID: ${this.draggedTask.id}, successfully updated to: ${targetList}`);
              this.draggedTask = null;
              this.sourceList = '';
            },
            (error) => {
              console.error('Error updating task status:', error);
              this.dependencyMessage = 'An error occurred while updating task status. Reverting changes.';

              setTimeout(() => {
                this.removeTaskFromTargetList(targetList);
                this.restoreTaskToOriginalPosition();
                this.dependencyMessage = '';
                this.showErrorModal();
              }, 500);
            }
          );
        }
      },
      (error) => {
        console.error('Error checking task membership:', error);
        this.dependencyMessage = 'An error occurred while checking task membership.';
        this.showErrorModal();
      }
    );
  }



// Funkcija koja premesti zadatak u ciljnu listu odmah
  moveTaskToTargetList(targetList: string) {
    // Uklanjanje zadatka iz izvornog spiska
    this.removeTaskFromSource();

    // Dodavanje zadatka u ciljni spisak
    switch (targetList) {
      case 'pending':
        this.pendingTasks.push(this.draggedTask);
        break;
      case 'work in progress':
        this.inProgressTasks.push(this.draggedTask);
        break;
      case 'done':
        this.doneTasks.push(this.draggedTask);
        break;
    }
  }

// Funkcija koja uklanja zadatak iz ciljnog spiska ako korisnik nije član
  removeTaskFromTargetList(targetList: string) {
    switch (targetList) {
      case 'pending':
        this.pendingTasks = this.pendingTasks.filter(task => task !== this.draggedTask);
        break;
      case 'work in progress':
        this.inProgressTasks = this.inProgressTasks.filter(task => task !== this.draggedTask);
        break;
      case 'done':
        this.doneTasks = this.doneTasks.filter(task => task !== this.draggedTask);
        break;
    }
  }

// Vraća zadatak na originalnu poziciju ako korisnik nije član
  restoreTaskToOriginalPosition() {
    switch (this.sourceList) {
      case 'pending':
        this.pendingTasks.push(this.draggedTask);
        break;
      case 'work in progress':
        this.inProgressTasks.push(this.draggedTask);
        break;
      case 'done':
        this.doneTasks.push(this.draggedTask);
        break;
    }

    // Očistiti selektovani zadatak i izvor
    this.draggedTask = null;
    this.sourceList = '';
  }

  removeTaskFromSource() {
    switch (this.sourceList) {
      case 'pending':
        this.pendingTasks = this.pendingTasks.filter(task => task !== this.draggedTask);
        break;
      case 'work in progress':
        this.inProgressTasks = this.inProgressTasks.filter(task => task !== this.draggedTask);
        break;
      case 'done':
        this.doneTasks = this.doneTasks.filter(task => task !== this.draggedTask);
        break;
    }
  }


  // Dodajte ovu metodu za prikaz modala
  showDeleteProjectModal(): void {
    this.isDeleteProjectModalVisible = true;
  }

  // Dodajte ovu metodu za zatvaranje modala
  closeDeleteProjectModal(): void {
    this.isDeleteProjectModalVisible = false;
  }

  // Dodajte ovu metodu za potvrdu brisanja projekta
  confirmDeleteProject(): void {
    this.deleteProject();
    this.closeDeleteProjectModal();
  }

  // Ažurirajte postojeću metodu za brisanje projekta
  deleteProject(): void {
    if (!this.project?.id) {
      console.error('Project ID is missing.');
      return;
    }
    if (this.project) {
      // Pozovi servis za brisanje projekta
      this.projectService.deleteProject(this.project.id).subscribe(
        () => {
          // Postavi flag da je projekat obrisan
          this.isProjectDeleted = true;
          console.log('Project successfully deleted!');

          // Prikaži modal sa porukom o uspešnom brisanju


          this.showDeleteSuccessModal();
        },
        (error) => {
          console.error('Error deleting project:', error);
        }
      );
    }
  }

  // Metoda za prikaz modala sa porukom o uspešnom brisanju
  showDeleteSuccessModal(): void {
    this.isDeleteSuccessModalVisible = true;
  }

  // Metoda za zatvaranje modala sa porukom o uspešnom brisanju
  closeDeleteSuccessModal(): void {
    this.isDeleteSuccessModalVisible = false;
    // Ponovo učita trenutnu rutu da bi se osvežila komponenta
    this.router.navigateByUrl('/', { skipLocationChange: true }).then(() => {
      this.router.navigate(['/dashboard']);
    });
  }

}
