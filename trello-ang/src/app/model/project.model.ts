export class Project {
  title: string;
  description: string;
  owner: string;
  expected_end_date: string; 
  min_people: number;         
  max_people: number;         
  users: any[];              

  constructor() {
    this.title = '';
    this.description = '';
    this.owner = '';
    this.expected_end_date = ''; 
    this.min_people = 0;
    this.max_people = 0;
    this.users = []; 
  }
}
