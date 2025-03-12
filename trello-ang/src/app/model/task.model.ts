export class Task {
  name: string;
  description: string;
  status: string;
  users: any[];
  project_id: string;
  dependsOn: string[];
  taskFiles: { fileName: string, content: string }[];
  position: number; // Dodato position polje

  constructor(
    name: string,
    description: string,
    status: string = 'pending',
    projectId: string = '',
    dependsOn: string[] = [],
    taskFiles: { fileName: string, content: string }[] = [],
    position: number = 0 // Dodato position polje u konstruktor
  ) {
    this.name = name;
    this.description = description;
    this.status = status;
    this.users = [];
    this.project_id = projectId;
    this.dependsOn = dependsOn;
    this.taskFiles = taskFiles;
    this.position = position; // Inicijalizacija position polja
  }
}
