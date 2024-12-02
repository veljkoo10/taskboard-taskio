import { Component, OnInit, OnDestroy, Output, EventEmitter } from '@angular/core';
import { NotificationService } from '../../services/notification.service';
import { Notification, NotificationStatus } from '../../model/notification.model';
import * as CryptoJS from 'crypto-js';

@Component({
  selector: 'app-notification',
  templateUrl: './notification.component.html',
  styleUrls: ['./notification.component.css']
})
export class NotificationComponent implements OnInit, OnDestroy {

  notifications: Notification[] = [];
  noNotificationsMessage: string = "You have no notifications.";
  private SECRET_KEY = 'my-secret-key-12345';

  @Output() unreadCountChanged = new EventEmitter<number>(); // Emituj broj nepročitanih notifikacija

  constructor(private notificationService: NotificationService) {}

  get unreadCount(): number {
    return this.notifications.filter(notification => notification.status === 'unread').length;
  }

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

  ngOnDestroy(): void {
    this.markAllUnreadAsRead();
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

        // Emit unreadCount after notifications are loaded
        this.unreadCountChanged.emit(this.unreadCount);
      },
      (error) => {
        console.error('Error loading notifications', error);
      }
    );
  }
  markAllUnreadAsRead(): void {
    const encryptedUserId = localStorage.getItem('user_id');
    if (!encryptedUserId) {
      console.error('No user ID found in localStorage.');
      return;
    }

    try {
      const userId = this.decryptUserId(encryptedUserId);
      this.notificationService.markNotificationAsRead(userId).subscribe(
        () => {
          console.log(`All unread notifications for user ID ${userId} marked as read.`);
          this.notifications.forEach(notification => {
            if (notification.status === NotificationStatus.Unread) {
              notification.status = NotificationStatus.Read;
            }
          });

          this.unreadCountChanged.emit(this.unreadCount);
        },
        (error) => {
          console.error(`Error marking notifications as read for user ID ${userId}`, error);
        }
      );
    } catch (error) {
      console.error('Error decrypting user ID', error);
    }
  }
}
