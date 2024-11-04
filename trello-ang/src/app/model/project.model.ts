export class Project {
  title: string;
  description: string;
  owner: string;
  expectedEndDate: Date | string;
  minNumber: number;
  maxNumber: number;

  constructor() {
    this.title = '';
    this.description = '';
    this.owner = '';
    this.expectedEndDate = '';
    this.minNumber = 0;
    this.maxNumber = 0;
  }
}
