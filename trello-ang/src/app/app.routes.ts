import { Routes } from '@angular/router';
import { LoginComponent } from './components/login/login.component';
import { RegisterComponent } from './components/register/register.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { ProjectDetailsComponent } from './components/project-details/project-details.component';
import { UserProfileComponent } from './components/user-profile/user-profile.component';
import { MagicLinkComponent } from './components/magic-login/magic-login.component';
import { VerifyMagicLinkComponent } from './components/verify-magic-link/verify-magic-link.component';
import { AuthGuard } from './guards/auth.guard';
import {NotificationComponent} from "./components/notification/notification.component";
import { AnalyticsComponent } from './components/analytics/analytics.component';
import {HistoryComponent} from "./components/history/history.component";

export const appRoutes: Routes = [
  {
    path: 'dashboard',
    component: DashboardComponent,
    canActivate: [AuthGuard],
    data: { roles: ['Manager','Member'] }
  },
  {
    path: 'verify-magic-link',
    component: VerifyMagicLinkComponent,
  },
  {
    path: 'profile',
    component: UserProfileComponent,
    canActivate: [AuthGuard],
    data: { roles: ['Manager','Member'] }
  },
  {
    path: 'project-details',
    component: ProjectDetailsComponent,
    canActivate: [AuthGuard],
    data: { roles: ['Manager','Member'] }
  },
  {
    path: 'notification',
    component: NotificationComponent,
    canActivate: [AuthGuard],
    data: { roles: ['Manager','Member'] }
  },
  {
    path: 'history',
    component: HistoryComponent,
    canActivate: [AuthGuard],
    data: { roles: ['Manager'] }
  },
  { path: 'analytics', component: AnalyticsComponent},
  { path: 'magic-login', component: MagicLinkComponent },
  { path: '', redirectTo: '/login', pathMatch: 'full' },
  { path: 'login', component: LoginComponent },
  { path: 'register', component: RegisterComponent },
  { path: '**', redirectTo: '/login' },
];
