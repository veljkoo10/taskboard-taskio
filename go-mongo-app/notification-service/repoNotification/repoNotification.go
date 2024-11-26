package repoNotification

import (
	"fmt"
	"log"
	"notification-service/models"
	"os"
	"time"

	// NoSQL: module containing Cassandra api client
	"github.com/gocql/gocql"
)

// NotificationRepo struct encapsulating Cassandra api client for notifications
type NotificationRepo struct {
	session *gocql.Session
	logger  *log.Logger
}

func New(logger *log.Logger) (*NotificationRepo, error) {
	logger.Println("Initializing notification service...")

	// Čitanje IP adrese iz okruženja
	db := os.Getenv("CASS_DB")
	if db == "" {
		logger.Println("CASS_DB environment variable is not set")
		return nil, fmt.Errorf("CASS_DB environment variable is not set")
	}

	// Konfigurišemo klaster za povezivanje sa Cassandrom
	cluster := gocql.NewCluster(db)
	cluster.Keyspace = "system" // Početno se povezuje sa 'system' keyspace-om
	cluster.Consistency = gocql.One

	var session *gocql.Session
	var err error

	// Pokušaj povezivanja sa Cassandra bazom (retry logika)
	for i := 0; i < 5; i++ {
		logger.Printf("Attempting to connect to Cassandra, try %d...\n", i+1)
		session, err = cluster.CreateSession()
		if err == nil {
			logger.Println("Successfully connected to Cassandra!")
			break
		}

		// Ako se ne poveže, loguj grešku i pokušaj ponovo
		logger.Printf("Attempt %d: Failed to connect to Cassandra: %v\n", i+1, err)
		time.Sleep(10 * time.Second)
	}

	// Ako konekcija nije uspela nakon 5 pokušaja, vrati grešku
	if err != nil {
		logger.Println("Failed to connect to Cassandra after 5 attempts.")
		return nil, err
	}

	// Kreiraj 'notifications' keyspace ako ne postoji
	logger.Println("Creating keyspace 'notifications' if it does not exist...")
	err = session.Query(`
		CREATE KEYSPACE IF NOT EXISTS notifications 
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}
	`).Exec()
	if err != nil {
		logger.Println("Error creating keyspace:", err)
	}

	// Sada se poveži sa 'notifications' keyspace-om
	cluster.Keyspace = "notifications"
	session, err = cluster.CreateSession()
	if err != nil {
		logger.Println("Error connecting to notifications keyspace:", err)
		return nil, err
	}

	// Uspešno povezivanje sa 'notifications' keyspace-om
	logger.Println("Successfully connected to 'notifications' keyspace!")

	// Vraćamo repo objekat
	return &NotificationRepo{
		session: session,
		logger:  logger,
	}, nil
}

// Disconnect from database
func (nr *NotificationRepo) CloseSession() {
	nr.session.Close()
}
func (nr *NotificationRepo) GetAllNotifications() ([]models.Notification, error) {
	var notifications []models.Notification

	// Selektujemo sve notifikacije iz tabele
	query := "SELECT id, user_id, message, is_active, created_at FROM notifications"
	nr.logger.Println("Executing query:", query) // Dodajemo log za upit

	iter := nr.session.Query(query).Iter()

	var notification models.Notification
	for iter.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.IsActive, &notification.CreatedAt) {
		// Logujemo svaku uspešno učitanu notifikaciju
		nr.logger.Printf("Fetched notification: %+v", notification)
		notifications = append(notifications, notification)
	}

	// Proveravamo grešku tokom iteracije
	if err := iter.Close(); err != nil {
		nr.logger.Printf("Failed to close iterator: %v", err)
		return nil, err
	}

	nr.logger.Printf("Returning %d notifications", len(notifications)) // Log za broj notifikacija
	return notifications, nil
}

// Create notifications table
func (nr *NotificationRepo) CreateTables() {
	err := nr.session.Query(
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s 
					(id UUID, user_id UUID, message text, is_active boolean, created_at timestamp, 
					PRIMARY KEY (user_id, created_at))`, "notifications")).Exec()
	if err != nil {
		nr.logger.Println(err)
	}
}

// Insert new notification
func (nr *NotificationRepo) InsertNotification(notification *models.Notification) error {
	// Generisanje novog UUID-a za 'id'
	notificationID, _ := gocql.RandomUUID()

	// Direktno korišćenje notification.UserID jer je već gocql.UUID
	err := nr.session.Query(
		`INSERT INTO notifications (id, user_id, message, is_active, created_at) 
		VALUES (?, ?, ?, ?, ?)`,
		notificationID, notification.UserID, notification.Message, notification.IsActive, notification.CreatedAt).Exec()
	if err != nil {
		nr.logger.Println(err)
		return err
	}

	return nil
}

// Get notifications by user ID
func (nr *NotificationRepo) GetNotificationsByUser(userID gocql.UUID) ([]models.Notification, error) {
	scanner := nr.session.Query(`SELECT id, user_id, message, is_active, created_at FROM notifications WHERE user_id = ?`,
		userID).Iter().Scanner()

	var notifications []models.Notification
	for scanner.Next() {
		var notification models.Notification
		err := scanner.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.IsActive, &notification.CreatedAt)
		if err != nil {
			nr.logger.Println(err)
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	if err := scanner.Err(); err != nil {
		nr.logger.Println(err)
		return nil, err
	}
	return notifications, nil
}

// InsertUser unosi novog korisnika u bazu
func (nr *NotificationRepo) InsertUser(user *models.User) error {
	userID, _ := gocql.RandomUUID() // Generisanje UUID za korisnika

	err := nr.session.Query(`
		INSERT INTO users (id, first_name, last_name, email, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, userID, user.FirstName, user.LastName, user.Email, user.CreatedAt).Exec()

	if err != nil {
		nr.logger.Println("Failed to insert user:", err)
		return err
	}
	return nil
}
