package session

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

const (
	SessionCookieName = "session_id"
	SessionCookieExp  = 72 * time.Hour
)

var (
	ErrNoAuth = errors.New("no session found")
)

type Session struct {
	ID     string
	UserID string
}

func NewSession(userID string) *Session {
	randID := make([]byte, 16)
	_, err := rand.Read(randID)
	if err != nil {
		return nil
	}

	return &Session{
		ID:     fmt.Sprintf("%x", randID),
		UserID: userID,
	}
}

type sessionKey string

var SessionKey sessionKey = "sessionKey"

func SessionFromContext(ctx context.Context) (*Session, error) {
	sess, ok := ctx.Value(SessionKey).(*Session)
	if !ok || sess == nil {
		return nil, ErrNoAuth
	}
	return sess, nil
}

func ContextWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, SessionKey, sess)
}
