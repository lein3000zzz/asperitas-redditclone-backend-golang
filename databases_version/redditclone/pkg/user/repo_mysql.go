package user

import (
	"database/sql"
	"errors"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"redditclone/pkg/utils"
)

var (
	ErrNoUser        = errors.New("user not found")
	ErrBadPass       = errors.New("invalid password")
	ErrAlreadyExists = errors.New("user already exists")
)

type UserMySQLRepo struct {
	db *sql.DB
}

func NewMySQLRepo(db *sql.DB) *UserMySQLRepo {
	return &UserMySQLRepo{db: db}
}

func (repo *UserMySQLRepo) Authorize(username, password string) (*User, error) {
	var user User
	err := repo.db.
		QueryRow("SELECT id, username, password FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, ErrNoUser
	}
	// в проде так нельзя, да, но мы не в проде
	if user.Password != password {
		return nil, ErrBadPass
	}
	return &user, nil
}

func (repo *UserMySQLRepo) Register(username, password string) (*User, error) {
	// можно было бы сделать вот так, меньше кода, но вроде бы больше оверхед
	// Думаю, лучше так, как сделал в итоге
	// _, err := repo.Authorize(username, password)
	// if !errors.Is(err, ErrNoUser) {
	//	 return nil, ErrAlreadyExists
	// }
	exists, err := repo.checkUserExists(username)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrAlreadyExists
	}

	user := &User{
		Username: username,
		Password: password,
		ID:       utils.GenerateID(),
	}
	_, err = repo.db.Exec("INSERT INTO users (id, username, password) VALUES (?, ?, ?)",
		user.ID, user.Username, user.Password)

	if err != nil {
		return nil, err
	}
	return user, nil
}

func (repo *UserMySQLRepo) checkUserExists(username string) (bool, error) {
	var exists int
	err := repo.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (repo *UserMySQLRepo) GenerateUserToken(u User) *jwt.Token {
	// просто нужно для фронта
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]string{
			"username": u.Username,
			"id":       u.ID,
		},
		// "exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	return token
}
