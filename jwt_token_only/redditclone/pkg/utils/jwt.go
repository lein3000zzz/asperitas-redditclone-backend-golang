package utils

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

var jwtSecret = []byte("super_secret_key")

var (
	ErrNoKey        = errors.New("key not found")
	ErrUnauthorized = errors.New("unauthorized")
)

func SendJwtToken(w http.ResponseWriter, token *jwt.Token) error {
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return err
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"token": tokenString,
	})
	return nil
}

func checkTokenAndGetClaims(r *http.Request) (map[string]interface{}, error) {
	tokenString, err := getTokenFromHeader(r)
	if err != nil {
		return map[string]interface{}{}, ErrUnauthorized
	}
	_, userClaims, err := parseToken(tokenString)
	if err != nil {
		return map[string]interface{}{}, ErrUnauthorized
	}
	return userClaims, nil
}

func getTokenFromHeader(r *http.Request) (string, error) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		return "", fmt.Errorf("no token provided")
	}
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	return tokenString, nil
}

// из примера
func parseToken(tokenString string) (*jwt.Token, jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		method, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok || method.Alg() != "HS256" {
			return nil, fmt.Errorf("bad sign method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, nil, fmt.Errorf("bad token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil, fmt.Errorf("no payload")
	}
	return token, claims, nil
}

func GetClaimsByKey(r *http.Request, key string) (map[string]interface{}, error) {
	userClaims, err := checkTokenAndGetClaims(r)
	if err != nil {
		return nil, err
	}

	claims, ok := userClaims[key].(map[string]interface{})
	if !ok {
		return nil, ErrNoKey
	}
	return claims, nil
}
