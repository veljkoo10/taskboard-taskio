package models

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
}
