package session

import (
	"errors"
	"net/http"
	"redditclone/pkg/utils"
	"time"
)

const (
	SessionCookieName = "session_id"
	SessionCookieExp  = 30 * time.Minute
)

var ErrNoSession = errors.New("no valid session")

type Session struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	UserID   string `json:"user_id"`
}

func newSession(userID, username string) *Session {
	return &Session{
		ID:       utils.GenerateID(),
		UserID:   userID,
		Username: username,
	}
}

type SessionManager interface {
	Check(r *http.Request) (*Session, error)
	UpdateCookie(w http.ResponseWriter, r *http.Request) error
	Create(w http.ResponseWriter, userID, username string) (*Session, error)
	Destroy(w http.ResponseWriter, r *http.Request) error
}
