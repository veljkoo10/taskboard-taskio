package models

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	IsActive bool   `json:"isActive"`
}

func NewUser(username, password, role, name, surname, email string) User {
	return User{
		Username: username,
		Password: password,
		Role:     role,
		Name:     name,
		Surname:  surname,
		Email:    email,
		IsActive: false,
	}
}
