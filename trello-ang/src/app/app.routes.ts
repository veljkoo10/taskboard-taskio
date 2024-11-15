import { Routes } from '@angular/router';
import { LoginComponent } from './components/login/login.component';
import { RegisterComponent } from './components/register/register.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import {ProjectDetailsComponent} from "./components/project-details/project-details.component";
import {UserProfileComponent} from "./components/user-profile/user-profile.component";
import {MagicLinkComponent} from "./components/magic-login/magic-login.component";
import { VerifyMagicLinkComponent } from './components/verify-magic-link/verify-magic-link.component';


export const appRoutes: Routes = [
  { path: 'dashboard', component: DashboardComponent },
  { path: 'verify-magic-link', component: VerifyMagicLinkComponent },
  { path: 'profile', component: UserProfileComponent },
  { path: 'magic-login', component: MagicLinkComponent },
  { path: '', redirectTo: '/login', pathMatch: 'full' },
  { path: 'login', component: LoginComponent },
  { path: 'register', component: RegisterComponent },
  { path: '**', redirectTo: '/login' },
  {path:'',component: ProjectDetailsComponent},

];
