export class Task {
  name: string;
  description: string;
  status: string;
  users: any[];
  project_id: string;

  constructor(name: string, description: string, status: string = 'pending', projectId: string = '') {
    this.name = name;
    this.description = description;
    this.status = status;
    this.users = [];
    this.project_id = projectId; // Initialize projectId
  }
}
