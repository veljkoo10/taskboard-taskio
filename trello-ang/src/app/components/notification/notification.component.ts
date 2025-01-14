import { Component, OnInit, OnDestroy, Output, EventEmitter } from '@angular/core';
import { NotificationService } from '../../services/notification.service';
import { Notification, NotificationStatus } from '../../model/notification.model';

@Component({
  selector: 'app-notification',
  templateUrl: './notification.component.html',
  styleUrls: ['./notification.component.css']
})
export class NotificationComponent implements OnInit, OnDestroy {

  notifications: Notification[] = [];
  noNotificationsMessage: string = "You have no notifications.";

  @Output() unreadCountChanged = new EventEmitter<number>(); // Emituj broj nepročitanih notifikacija

  constructor(private notificationService: NotificationService) {}

  get unreadCount(): number {
    return this.notifications.filter(notification => notification.status === 'unread').length;
  }

  ngOnInit(): void {
    const userId = localStorage.getItem('user_id');
    if (userId) {
      this.loadNotifications(userId);
    } else {
      console.error('No user ID found in localStorage.');
    }
  }

  ngOnDestroy(): void {
    this.markAllUnreadAsRead();
  }

  loadNotifications(userId: string): void {
    this.notificationService.getNotificationsByUserID(userId).subscribe(
      (data) => {
        if (!data || data.length === 0) {
          this.notifications = [];
        } else {
          this.notifications = data.map((notification: Notification) => ({
            ...notification,
            created_at: new Date(notification.created_at)
          }));
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
    const userId = localStorage.getItem('user_id');
    if (!userId) {
      console.error('No user ID found in localStorage.');
      return;
    }

    this.notificationService.markNotificationAsRead(userId).subscribe(
      () => {
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
  }
}
