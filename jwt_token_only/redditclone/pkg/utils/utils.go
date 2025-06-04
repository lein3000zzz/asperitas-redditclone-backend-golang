package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrGenerateID = errors.New("can't generate id")
)

func GenerateID() (string, error) {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", ErrGenerateID
	}
	return hex.EncodeToString(bytes), nil
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	out, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, err = w.Write(out)
	if err != nil {
		return
	}
}
