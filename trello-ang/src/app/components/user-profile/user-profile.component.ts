import { Component, OnInit } from '@angular/core';
import { AuthService } from '../../services/auth.service';  // Importuj AuthService

@Component({
  selector: 'app-user-profile',
  templateUrl: './user-profile.component.html',
  styleUrls: ['./user-profile.component.css']
})
export class UserProfileComponent implements OnInit {
  userProfile: any = {};  // Ovdje čuvamo podatke o korisniku
  userIcon: string = '';   // Ako ima profilnu sliku

  constructor(private authService: AuthService) {}

  ngOnInit(): void {
    // Pozovi authService da učita podatke o korisniku
    this.authService.getProfileData().subscribe(
      (data) => {
        console.log(data);  // Proveri podatke u konzoli
        this.userProfile = data;  // Spremi podatke o korisniku
        this.userIcon = data.profilePicture;  // Ako postoji profilna slika
      },
      (error) => {
        console.error('Failed to load user profile data', error);
      }
    );
  }
}
