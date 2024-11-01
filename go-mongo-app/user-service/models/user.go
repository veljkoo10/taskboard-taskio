package models

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	IsActive bool   `json:"isActive"` // promenjeno iz isActive u IsActive da bi pratilo standard Go konvencije
}

// Konstruktori mogu biti dodati ovde
func NewUser(username, password, role, name, surname, email string) User {
	return User{
		Username: username,
		Password: password,
		Role:     role,
		Name:     name,
		Surname:  surname,
		Email:    email,
		IsActive: false, // podrazumevano je false
	}
}
