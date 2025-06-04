package user

import (
	"errors"
	"redditclone/pkg/utils"
	"sync"

	"github.com/dgrijalva/jwt-go"
)

var (
	ErrNoUser        = errors.New("user not found")
	ErrBadPass       = errors.New("invald password")
	ErrAlreadyExists = errors.New("user already exists")
)

type UserMemoryRepo struct {
	sync.RWMutex
	Users map[string]*User
}

func NewMemoryRepo() *UserMemoryRepo {
	return &UserMemoryRepo{
		Users: make(map[string]*User),
	}
}

func (repo *UserMemoryRepo) Authorize(username, password string) (*User, error) {
	repo.RLock()
	defer repo.RUnlock()
	u, ok := repo.Users[username]
	if !ok {
		return nil, ErrNoUser
	}
	if u.Password != password {
		return nil, ErrBadPass
	}
	return u, nil
}

func (repo *UserMemoryRepo) Register(username, password string) (*User, error) {
	repo.Lock()
	defer repo.Unlock()
	if _, exists := repo.Users[username]; exists {
		return nil, ErrAlreadyExists
	}
	newUserID, err := utils.GenerateID()
	if err != nil {
		return nil, err
	}
	u := &User{
		Username: username,
		Password: password,
		ID:       newUserID,
	}
	repo.Users[username] = u
	return u, nil
}

func (repo *UserMemoryRepo) GenerateUserToken(u User) *jwt.Token {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]string{
			"username": u.Username,
			"id":       u.ID,
		},
		// "exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	return token
}
