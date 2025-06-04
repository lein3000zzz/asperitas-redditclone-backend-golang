package utils

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"net/http"
)

func GenerateID() string {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "" // не знаю, какую тут логику стоит имплементировать. Можно вообще сделать несколько (ограниченное количество) попыток
	}
	return hex.EncodeToString(bytes)
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	out, err := json.Marshal(data)
	if err != nil {
		fmt.Println("dasdasdasdada")
		return
	}
	_, err = w.Write(out)
	if err != nil {
		return
	}
}

func HandleMongoCursorClose(cursor *mongo.Cursor, ctx context.Context) {
	err := cursor.Close(ctx)
	if err != nil {
		return
	}
}

func CloseDB(db *sql.DB) {
	err := db.Close()
	if err != nil {
		return
	}
}

func CloseBody(body io.ReadCloser) {
	err := body.Close()
	if err != nil {
		return
	}
}
