package service

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"time"
	"user-service/db"
	"user-service/models"
	"user-service/notification"
)

var emailConfig = notification.EmailConfig{
	From:     "taskio2024@gmail.com",
	Password: "znnbgxgvshvythfq",
	SMTPHost: "smtp.gmail.com",
	SMTPPort: "587",
}

func GetUsers() ([]models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var users []models.User

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func GetUserByID(userID string) (models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return models.User{}, errors.New("invalid user ID format")
	}

	var user models.User
	err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return models.User{}, errors.New("user not found")
	}

	return user, nil
}

func validateUser(user models.User) error {
	if user.Username == "" || user.Email == "" || user.Password == "" || user.Name == "" || user.Surname == "" {
		return errors.New("all fields (username, email, password, name, surname) are required")
	}

	if !isValidEmail(user.Email) {
		return errors.New("invalid email format")
	}
	if !isPasswordValid(user.Password) {
		return errors.New("invalid password format: password must be at least 8 characters long and contain an uppercase letter, a lowercase letter, a number, and a special character")
	}

	existingUserByUsername, err := FindUserByUsername(user.Username)
	if err == nil && existingUserByUsername.Username != "" {
		return errors.New("username already exists")
	}

	existingUserByEmail, err := FindUserByEmail(user.Email)
	if err == nil && existingUserByEmail.Email != "" {
		return errors.New("email already exists")
	}

	return nil
}
func isPasswordValid(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#~$%^&*(),.?":{}|<>]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
func EmailExists(email string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
func UsernameExists(username string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
func RegisterUser(user models.User) (string, error) {
	if err := validateUser(user); err != nil {
		return "", err
	}

	collection := db.Client.Database("testdb").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	user.Password = string(hashedPassword)
	user.IsActive = false

	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}

	subject := "Thanks for registering"
	body := "Your registration is successful! Click the following link to activate your account: http://localhost:8080/confirm?email=" + user.Email
	err = notification.SendEmail(user.Email, subject, body, emailConfig)
	if err != nil {
		return "Registration successful, but failed to send confirmation email", nil
	}

	return "Registration successful. Please check your email to confirm registration.", nil
}

func FindUserByUsername(username string) (models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), map[string]interface{}{"username": username}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return models.User{}, nil
	}
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
func FindUserByEmail(email string) (models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return models.User{}, nil
	}
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
func ConfirmUser(email string) error {
	collection := db.Client.Database("testdb").Collection("users")

	filter := bson.M{"email": email}
	update := bson.M{"$set": bson.M{"isActive": true}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}
func IsUserActive(email string) (bool, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // User does not exist
		}
		return false, err
	}
	return user.IsActive, nil
}
func ResetPassword(email string) (string, error) {
	collection := db.Client.Database("testdb").Collection("users")
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return "", fmt.Errorf("The user with this email does not exist")
	}

	if !user.IsActive {
		return "", fmt.Errorf("The user is not active, you cannot reset the password")
	}

	err = SendPasswordResetEmail(user.Email)
	if err != nil {
		return "", err
	}

	return "The password reset email has been successfully sent. Check your email.", nil
}
func SendPasswordResetEmail(email string) error {

	subject := "Password reset"
	body := "Click the following link to reset your password: http://localhost:8080/reset-password?email=" + email
	err := notification.SendEmail(email, subject, body, emailConfig)
	return err
}
func LoginUser(user models.User) (models.User, error) {
	collection := db.Client.Database("testdb").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dbUser models.User
	err := collection.FindOne(ctx, bson.M{"username": user.Username}).Decode(&dbUser)
	if err == mongo.ErrNoDocuments {
		return models.User{}, errors.New("user not found")
	} else if err != nil {
		return models.User{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		return models.User{}, errors.New("invalid password")
	}

	return dbUser, nil
}

// ChangePassword menja korisničku lozinku nakon validacije stare lozinke.
func ChangePassword(userID, oldPassword, newPassword string) error {
	// Pronađi korisnika po ID-u
	user, err := GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Proveri da li je uneta stara lozinka ispravna
	if user.Password != oldPassword {
		return errors.New("incorrect old password")
	}

	// Hesiraj novu lozinku
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("could not hash the new password")
	}

	// Ažuriraj lozinku u bazi
	collection := db.Client.Database("testdb").Collection("users")
	update := bson.M{"$set": bson.M{"password": hashedPassword}}
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": user.ID}, update)
	if err != nil {
		return errors.New("failed to update password")
	}

	return nil
}
