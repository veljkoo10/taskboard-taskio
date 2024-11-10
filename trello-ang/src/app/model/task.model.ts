export class Task {
    name: string;
    description: string;
    users: any[]
  
    constructor(name: string, description: string) {
      this.name = name;
      this.description = description;
      this.users = [];
    }
  }
  
  