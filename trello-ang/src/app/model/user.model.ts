export class User {
  id: string;
  username: string;
  password: string;
  role: string;
  name: string;
  surname: string;
  email: string;
  isActive: boolean;

  constructor(id: string,username: string, password: string, role: string, name: string, surname: string, email: string) {
    this.id = id;
    this.username = username;
    this.password = password;
    this.role = role;
    this.name = name;
    this.surname = surname;
    this.email = email;
    this.isActive = false;
  }
}

