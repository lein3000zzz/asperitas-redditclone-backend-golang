package session

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

type SessionsManager struct {
	data map[string]*Session
	sync.RWMutex
}

func NewSessionsManager() *SessionsManager {
	return &SessionsManager{
		data: make(map[string]*Session, 10),
	}
}

func (sm *SessionsManager) Check(r *http.Request) (*Session, error) {
	sessionCookie, err := r.Cookie("session_id")

	if errors.Is(err, http.ErrNoCookie) {
		return nil, ErrNoAuth
	}

	if err != nil || sessionCookie == nil {
		return nil, err
	}

	sm.RLock()
	defer sm.RUnlock()
	sess, ok := sm.data[sessionCookie.Value]

	if !ok {
		return nil, ErrNoAuth
	}

	return sess, nil
}

func (sm *SessionsManager) Create(w http.ResponseWriter, userID string) (*Session, error) {
	sess := NewSession(userID)

	sm.Lock()
	defer sm.Unlock()
	sm.data[sess.ID] = sess

	cookie := &http.Cookie{
		Name:    SessionCookieName,
		Value:   sess.ID,
		Expires: time.Now().Add(SessionCookieExp),
		Path:    "/",
	}
	http.SetCookie(w, cookie)
	return sess, nil
}

// оставил это из примера, но у нас в логике нет logout, мы вообще этот запрос не обрабатываем никак
func (sm *SessionsManager) DestroyCurrent(w http.ResponseWriter, r *http.Request) error {
	sess, err := SessionFromContext(r.Context())
	if err != nil {
		return err
	}

	sm.Lock()
	delete(sm.data, sess.ID)
	sm.Unlock()

	cookie := http.Cookie{
		Name:    "session_id",
		Expires: time.Now().AddDate(0, 0, -1),
		Path:    "/",
	}
	http.SetCookie(w, &cookie)
	return nil
}
