package repoNotification

import (
	"fmt"
	"github.com/gocql/gocql"
	"log"
	"notification-service/models"
	"os"
	"strconv"
	"time"
)

type NotificationRepo struct {
	session *gocql.Session
	logger  *log.Logger
}

func New(logger *log.Logger) (*NotificationRepo, error) {

	dbHost := os.Getenv("CASSANDRA_HOST")
	if dbHost == "" {
		dbHost = "cassandra"
	}

	dbPort := os.Getenv("CASSANDRA_PORT")
	if dbPort == "" {
		dbPort = "9042"
	}

	port, err := strconv.Atoi(dbPort)
	if err != nil {
		logger.Println("Invalid Cassandra port:", dbPort)
		return nil, fmt.Errorf("Invalid Cassandra port: %s", dbPort)
	}

	cluster := gocql.NewCluster(dbHost)
	cluster.Port = port
	cluster.Keyspace = "system"
	session, err := cluster.CreateSession()
	if err != nil {
		logger.Println("Error connecting to Cassandra:", err)
		return nil, err
	}

	err = session.Query(fmt.Sprintf(`CREATE KEYSPACE IF NOT EXISTS notifications
	WITH replication = {'class' : 'SimpleStrategy', 'replication_factor' : 3}`)).Exec()
	if err != nil {
		logger.Println("Error creating keyspace:", err)
		return nil, err
	}
	session.Close()

	cluster.Keyspace = "notifications"
	cluster.Consistency = gocql.One
	session, err = cluster.CreateSession()
	if err != nil {
		logger.Println("Error connecting to notifications keyspace:", err)
		return nil, err
	}

	return &NotificationRepo{
		session: session,
		logger:  logger,
	}, nil
}

func (repo *NotificationRepo) DropKeyspace() error {
	dropKeyspaceQuery := fmt.Sprintf("DROP KEYSPACE IF EXISTS %s", "notifications")
	if err := repo.session.Query(dropKeyspaceQuery).Exec(); err != nil {
		repo.logger.Println("Error dropping keyspace:", err)
		return err
	}
	repo.logger.Println("Keyspace dropped successfully.")
	return nil
}

func (repo *NotificationRepo) GetNotificationsByUser(userID gocql.UUID) ([]models.Notification, error) {
	var notifications []models.Notification

	iter := repo.session.Query(`
		SELECT id, message, created_at, status FROM notifications WHERE user_id = ?`, userID).Iter()

	var notification models.Notification
	for iter.Scan(&notification.ID, &notification.Message, &notification.CreatedAt, &notification.Status) {
		notifications = append(notifications, notification)
	}

	if err := iter.Close(); err != nil {
		repo.logger.Println("Error fetching notifications:", err)
		return nil, err
	}

	return notifications, nil
}

func (repo *NotificationRepo) CloseSession() {
	repo.session.Close()
}

func (repo *NotificationRepo) CreateTables() {

	err := repo.session.Query(`CREATE TABLE IF NOT EXISTS notifications (
		user_id TEXT,
		created_at TIMESTAMP,
		id UUID,
		message TEXT,
		status TEXT,
		PRIMARY KEY (user_id, created_at, id)
	) WITH CLUSTERING ORDER BY (created_at DESC);`).Exec()
	if err != nil {
		repo.logger.Println("Error creating notifications table with clustering:", err)
		return
	}

}

func (repo NotificationRepo) Create(notification *models.Notification) error {

	notification.CreatedAt = time.Now()

	notification.ID, _ = gocql.RandomUUID()

	err := repo.session.Query(
		`INSERT INTO notifications (id, user_id, message, created_at, status)
	VALUES (?, ?, ?, ?, ?)`,
		notification.ID, notification.UserID, notification.Message, notification.CreatedAt, notification.Status).Exec()

	if err != nil {
		repo.logger.Println("Error inserting notification:", err)
		return err
	}
	return nil
}

func (repo *NotificationRepo) GetByID(id gocql.UUID) (*models.Notification, error) {
	var notification models.Notification
	err := repo.session.Query(`
		SELECT id, user_id, message, created_at, status
		FROM notifications WHERE id = ?`, id).Consistency(gocql.One).Scan(
		&notification.ID, &notification.UserID, &notification.Message, &notification.CreatedAt, &notification.Status)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, fmt.Errorf("notification with ID %v not found", id)
		}
		repo.logger.Println("Error fetching notification:", err)
		return nil, err
	}

	location, err := time.LoadLocation("Europe/Budapest")
	if err != nil {
		log.Println("Error loading time zone:", err)
		return nil, err
	}

	notification.CreatedAt = notification.CreatedAt.In(location)

	return &notification, nil
}

func (repo *NotificationRepo) GetByUserID(userID string) ([]*models.Notification, error) {
	var notifications []*models.Notification

	iter := repo.session.Query(`
        SELECT id, user_id, message, created_at, status
        FROM notifications 
        WHERE user_id = ? 
        ORDER BY created_at DESC`, userID).Iter()

	for {
		var notification models.Notification
		if !iter.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.CreatedAt, &notification.Status) {
			break
		}
		notifications = append(notifications, &notification)
	}

	if err := iter.Close(); err != nil {
		repo.logger.Println("Error closing iterator:", err)
		return nil, err
	}

	return notifications, nil
}

func (repo *NotificationRepo) UpdateStatus(createdAt time.Time, userID string, id gocql.UUID, status models.NotificationStatus) error {
	err := repo.session.Query(`
        UPDATE notifications 
        SET status = ? 
        WHERE user_id = ? AND created_at = ? AND id = ?`,
		status, userID, createdAt, id).Exec()

	if err != nil {
		repo.logger.Println("Error updating notification status:", err)
		return err
	}
	return nil
}

func (repo *NotificationRepo) GetAllNotifications() ([]models.Notification, error) {
	var notifications []models.Notification

	iter := repo.session.Query(`
		SELECT id, user_id, message, created_at, status FROM notifications`).Iter()

	var notification models.Notification
	for iter.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.CreatedAt, &notification.Status) {
		notifications = append(notifications, notification)
	}

	if err := iter.Close(); err != nil {
		repo.logger.Println("Error fetching all notifications:", err)
		return nil, err
	}

	return notifications, nil
}
func (r *NotificationRepo) MarkAllAsRead(userID string) error {
	var notificationIDs []struct {
		ID        gocql.UUID `json:"id"`
		CreatedAt time.Time  `json:"created_at"`
	}

	iter := r.session.Query("SELECT id, created_at FROM notifications WHERE user_id = ? AND status = ? ALLOW FILTERING", userID, "unread").Iter()
	for {
		var id gocql.UUID
		var createdAt time.Time
		if !iter.Scan(&id, &createdAt) {
			break
		}
		notificationIDs = append(notificationIDs, struct {
			ID        gocql.UUID `json:"id"`
			CreatedAt time.Time  `json:"created_at"`
		}{
			ID:        id,
			CreatedAt: createdAt,
		})
	}

	if err := iter.Close(); err != nil {
		return fmt.Errorf("failed to fetch unread notifications for user %s: %w", userID, err)
	}

	if len(notificationIDs) == 0 {
		return nil // No unread notifications, nothing to update
	}

	// Update status for each notification
	for _, notification := range notificationIDs {
		err := r.session.Query("UPDATE notifications SET status = ? WHERE user_id = ? AND created_at = ? AND id = ?",
			"read", userID, notification.CreatedAt, notification.ID).Exec()
		if err != nil {
			return fmt.Errorf("failed to update notification %s for user %s: %w", notification.ID, userID, err)
		}
	}

	return nil
}
func EnsureKeyspaceAndTable(session *gocql.Session, logger *log.Logger) error {
	var keyspaceCount int
	err := session.Query("SELECT count(*) FROM system_schema.keyspaces WHERE keyspace_name = 'notifications'").Scan(&keyspaceCount)
	if err != nil {
		return err
	}

	if keyspaceCount == 0 {
		logger.Println("Creating keyspace 'notifications'...")
		createKeyspaceQuery := `CREATE KEYSPACE IF NOT EXISTS notifications 
			WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`
		if err := session.Query(createKeyspaceQuery).Exec(); err != nil {
			return err
		}
	}

	var tableCount int
	err = session.Query("SELECT count(*) FROM system_schema.tables WHERE keyspace_name = 'notifications' AND table_name = 'notifications'").Scan(&tableCount)
	if err != nil {
		return err
	}

	if tableCount == 0 {
		logger.Println("Creating table 'notifications.notifications'...")
		createTableQuery := `CREATE TABLE IF NOT EXISTS notifications.notifications (
			user_id TEXT,
			created_at TIMESTAMP,
			id UUID,
			message TEXT,
			status TEXT,
			PRIMARY KEY (user_id, created_at, id)
		) WITH CLUSTERING ORDER BY (created_at DESC)`
		if err := session.Query(createTableQuery).Exec(); err != nil {
			return err
		}
	}

	return nil
}
