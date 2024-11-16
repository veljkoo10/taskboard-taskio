import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-magic-link',
  templateUrl: './magic-login.component.html',
  styleUrls: ['./magic-login.component.css']
})
export class MagicLinkComponent {

  email: string = '';
  message: string = '';

  constructor(
    private authService: AuthService,
    private router: Router,
    private http: HttpClient
  ) {}


}
