package main

import (
	"context"
	"database/sql"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"redditclone/pkg/handlers"
	"redditclone/pkg/post"
	"redditclone/pkg/session"
	"redditclone/pkg/user"

	"go.uber.org/zap"
)

func initUserDB() *sql.DB {
	dsn := "root:love@tcp(localhost:3306)/golang?"
	dsn += "&charset=utf8"
	dsn += "&interpolateParams=true"

	db, err := sql.Open("mysql", dsn)

	panicOnErr(err)

	db.SetMaxOpenConns(10)

	err = db.Ping()

	panicOnErr(err)

	return db
}

func initPostsDB() *mongo.Collection {
	ctx := context.Background()
	sess, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost"))
	panicOnErr(err)
	collection := sess.Database("golang").Collection("posts")
	return collection
}

// тут была реализация с redis.Conn, как было в примерах, но потом я погуглил и
// еще раз взглянул на задание - увидел другую библиотеку и сделал через нее.
// Вопрос: почему в примерах "github.com/gomodule/redigo/redis"? Он все же лучше или хуже, чем "github.com/redis/go-redis/v9"?
// Просто хочется понять юзкейсы и для того, и для того + почему в примерах выбор пал в сторону первого
// func initSessRedis() redis.Conn {
//	redisAddr := "redis://localhost:6379/0"
//	redisConn, err := redis.DialURL(redisAddr)
//	panicOnErr(err)
//	return redisConn
// }

func main() {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("Error initializing zap logger:", err)
		return
	}

	defer func(zapLogger *zap.Logger) {
		err := zapLogger.Sync()
		if err != nil {
			fmt.Println("Error syncing zap logger:", err)
		}
	}(zapLogger)

	logger := zapLogger.Sugar()

	userDB := initUserDB()
	collectionPosts := initPostsDB()
	// redisConn := initSessRedis()
	redisAddr := "localhost:6379"
	sm := session.NewRedisSessionManager(redisAddr)

	userRepo := user.NewMySQLRepo(userDB)
	postRepo := post.NewMongoRepo(collectionPosts, logger)

	userHandler := &handlers.UserHandler{
		UserRepo: userRepo,
		Logger:   logger,
		Sessions: sm,
	}

	postHandler := &handlers.PostHandler{
		PostRepo: postRepo,
		Logger:   logger,
		Sessions: sm,
	}

	port := "8080"
	configuredRouter := handlers.ConfigureRoutes(userHandler, postHandler, logger)
	fmt.Printf("Starting server at :%s", port)
	if err := http.ListenAndServe(":"+port, configuredRouter); err != nil {
		logger.Errorf("Server error: %v", err)
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
