import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { AppRoutingModule } from '../../app-routing.module';
import { AppComponent } from './app.component';
import { LoginComponent } from '../login/login.component';
import { HttpClientModule } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { RegisterComponent } from '../register/register.component';
import { DashboardComponent } from '../dashboard/dashboard.component';
import { ProjectDetailsComponent } from '../project-details/project-details.component';
import { UserProfileComponent } from '../user-profile/user-profile.component';
import {MagicLinkComponent} from "../magic-login/magic-login.component";
import {VerifyMagicLinkComponent} from "../verify-magic-link/verify-magic-link.component";


@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    DashboardComponent,
    ProjectDetailsComponent,
    UserProfileComponent,
    MagicLinkComponent,
    VerifyMagicLinkComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule,
    HttpClientModule,
    FormsModule,
    RegisterComponent,
  ],
  bootstrap: [AppComponent]
})
export class AppModule {}
