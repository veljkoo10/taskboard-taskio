// src/app/models/notification.model.ts

export enum NotificationStatus {
  Unread = 'unread',
  Read = 'read'
}

export interface Notification {
  id: string;
  user_id: string;
  message: string;
  created_at: string;
  is_active: boolean;
  status: NotificationStatus;
}

export type Notifications = Notification[];
