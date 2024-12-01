import { Component, OnInit } from '@angular/core';
import { NotificationService } from '../../services/notification.service';
import { Notification } from '../../model/notification.model';
import * as CryptoJS from 'crypto-js';

@Component({
  selector: 'app-notification',
  templateUrl: './notification.component.html',
  styleUrls: ['./notification.component.css']
})
export class NotificationComponent implements OnInit {

  notifications: Notification[] = [];
  noNotificationsMessage: string = "You have no notifications.";  // Poruka kada nema notifikacija
  private SECRET_KEY = 'my-secret-key-12345'; // Ključ za dešifrovanje

  constructor(private notificationService: NotificationService) {}

  ngOnInit(): void {
    console.log("Loading notifications...");
    const encryptedUserId = localStorage.getItem('user_id');
    if (encryptedUserId) {
      try {
        const userId = this.decryptUserId(encryptedUserId);
        this.loadNotifications(userId);
      } catch (error) {
        console.error('Error decrypting user ID', error);
      }
    } else {
      console.log('Error: User ID not found');
    }
  }

  decryptUserId(encryptedUserId: string): string {
    const bytes = CryptoJS.AES.decrypt(encryptedUserId, this.SECRET_KEY);
    return bytes.toString(CryptoJS.enc.Utf8);
  }

  loadNotifications(userId: string): void {
    this.notificationService.getNotificationsByUserID(userId).subscribe(
      (data) => {
        if (!data || data.length === 0) {
          this.notifications = [];
          console.log('No notifications found.');
        } else {
          this.notifications = data.map((notification: Notification) => ({
            ...notification,
            created_at: new Date(notification.created_at)
          }));
          console.log('Notifications loaded:', this.notifications);
        }
      },
      (error) => {
        console.error('Error loading notifications', error);
      }
    );
  }
}
