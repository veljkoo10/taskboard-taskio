export interface Event {
  type: string;
  time: string;
  event: {
    managerId?: string;   // menadžer ID je opcionalan, ali može biti prisutan
    projectId: string;    // Projekat ID mora biti prisutan
    taskId?: string;      // zadatak ID je opcionalan
    memberId?: string;    // član ID je opcionalan
    status?: string;
    previousStatus?: string; // Prethodni status zadatka
    currentStatus?: string;  // Trenutni status zadatka
    filePath?: string;
  };
  projectId: string;      // Ovo je dodatni projekat ID koji može biti korišćen u drugim događajima
}
