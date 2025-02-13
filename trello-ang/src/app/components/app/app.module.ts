import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { AppRoutingModule } from '../../app-routing.module';
import { AppComponent } from './app.component';
import { LoginComponent } from '../login/login.component';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { RegisterComponent } from '../register/register.component';
import { DashboardComponent } from '../dashboard/dashboard.component';
import { ProjectDetailsComponent } from '../project-details/project-details.component';
import { UserProfileComponent } from '../user-profile/user-profile.component';
import { MagicLinkComponent } from "../magic-login/magic-login.component";
import { VerifyMagicLinkComponent } from "../verify-magic-link/verify-magic-link.component";
import { AuthGuard } from "../../guards/auth.guard";
import { AuthInterceptor } from '../../interceptor/auth.interceptor';
import { appRoutes } from "../../app.routes";
import { RouterModule } from "@angular/router";
import {RecaptchaModule} from "ng-recaptcha";
import {CapitalizePipe} from "../../pipe/capitalize.pipe";
import {RecaptchaValueAccessor} from "../../recaptcha/recaptcha-value-accessor.directive";
import {NotificationComponent} from "../notification/notification.component";
import {DragDropModule} from "@angular/cdk/drag-drop";
import { AnalyticsComponent } from '../analytics/analytics.component';
import {HistoryComponent} from "../history/history.component";

@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    DashboardComponent,
    ProjectDetailsComponent,
    UserProfileComponent,
    MagicLinkComponent,
    VerifyMagicLinkComponent,
    CapitalizePipe,
    RecaptchaValueAccessor,
    NotificationComponent,
    HistoryComponent,
    AnalyticsComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule,
    HttpClientModule,
    FormsModule,
    RegisterComponent,
    RouterModule.forRoot(appRoutes),
    RecaptchaModule,
    DragDropModule
  ],
  providers: [
    AuthGuard,
    {
      provide: HTTP_INTERCEPTORS,
      useClass: AuthInterceptor,
      multi: true
    }
  ],
  bootstrap: [AppComponent]
})
export class AppModule {}
