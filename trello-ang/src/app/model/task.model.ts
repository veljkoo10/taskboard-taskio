export class Task {
  name: string;
  description: string;
  status: string;
  users: any[];
  project_id: string;
  dependsOn: string[];  // Niz ID-eva zadataka na koje trenutni zadatak zavisi

  constructor(name: string, description: string, status: string = 'pending', projectId: string = '', dependsOn: string[] = []) {
    this.name = name;
    this.description = description;
    this.status = status;
    this.users = [];
    this.project_id = projectId;
    this.dependsOn = dependsOn;  // Inicijalizacija polja za zavisnosti
  }
}
