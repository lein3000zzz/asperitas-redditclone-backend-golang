package user

import "github.com/dgrijalva/jwt-go"

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

type UserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserRepo interface {
	Authorize(login, password string) (*User, error)
	Register(login, password string) (*User, error)
	GenerateUserToken(u User) *jwt.Token
}
