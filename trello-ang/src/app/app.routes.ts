import { Routes } from '@angular/router';
import { LoginComponent } from './components/login/login.component';
import { RegisterComponent } from './components/register/register.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import {ProjectDetailsComponent} from "./components/project-details/project-details.component";
import {UserProfileComponent} from "./components/user-profile/user-profile.component";


export const appRoutes: Routes = [
  { path: 'dashboard', component: DashboardComponent },
  { path: 'users/:id/profile', component: UserProfileComponent },  // Ruta za prikazivanje korisničkog profila
  { path: '', redirectTo: '/login', pathMatch: 'full' },
  { path: 'login', component: LoginComponent },
  { path: 'register', component: RegisterComponent },
  { path: '**', redirectTo: '/login' },
  {path:'',component: ProjectDetailsComponent},
  { path: '', redirectTo: '/dashboard', pathMatch: 'full' },

];
