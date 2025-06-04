package session

import (
	"context"
	"encoding/json"
	"net/http"
	_ "redditclone/pkg/utils"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisSessionManager struct {
	Client *redis.Client
}

func NewRedisSessionManager(redisURL string) *RedisSessionManager {
	client := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     "",
		DB:           0,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})
	return &RedisSessionManager{Client: client}
}

func (rsm *RedisSessionManager) Create(w http.ResponseWriter, userID, username string) (*Session, error) {
	sess := newSession(userID, username)
	data, err := json.Marshal(sess)
	if err != nil {
		return nil, err
	}

	err = rsm.Client.Set(context.Background(), sess.ID, data, SessionCookieExp).Err()
	if err != nil {
		return nil, err
	}

	cookie := &http.Cookie{
		Name:    SessionCookieName,
		Value:   sess.ID,
		Expires: time.Now().Add(SessionCookieExp),
		Path:    "/",
	}
	http.SetCookie(w, cookie)
	return sess, nil
}

func (rsm *RedisSessionManager) Check(r *http.Request) (*Session, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	ctx := context.Background()
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, ErrNoSession
	}

	data, err := rsm.Client.Get(ctx, cookie.Value).Bytes()
	if err != nil {
		return nil, ErrNoSession
	}

	var sess Session
	err = json.Unmarshal(data, &sess)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (rsm *RedisSessionManager) Destroy(w http.ResponseWriter, r *http.Request) error {
	ctx := context.Background()
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return err
	}

	err = rsm.Client.Del(ctx, cookie.Value).Err()
	if err != nil {
		return err
	}

	expired := &http.Cookie{
		Name:    SessionCookieName,
		Expires: time.Now().Add(-time.Hour),
		Path:    "/",
	}
	http.SetCookie(w, expired)
	return nil
}

func (rsm *RedisSessionManager) UpdateCookie(w http.ResponseWriter, r *http.Request) error {
	ctx := context.Background()
	cookie, err := r.Cookie(SessionCookieName)

	if err != nil {
		return err
	}

	err = rsm.Client.Expire(ctx, cookie.Value, SessionCookieExp).Err()
	if err != nil {
		return err
	}

	updatedCookie := &http.Cookie{
		Name:    SessionCookieName,
		Value:   cookie.Value,
		Path:    "/",
		Expires: time.Now().Add(SessionCookieExp),
	}
	http.SetCookie(w, updatedCookie)
	return nil
}
