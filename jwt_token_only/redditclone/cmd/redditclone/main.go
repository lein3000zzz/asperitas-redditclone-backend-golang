package main

import (
	"fmt"
	"net/http"
	"redditclone/pkg/handlers"
	"redditclone/pkg/post"
	"redditclone/pkg/session"
	"redditclone/pkg/user"
	"redditclone/pkg/utils/middleware"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func configureRoutes(userHandler *handlers.UserHandler, postHandler *handlers.PostHandler, logger *zap.SugaredLogger) http.Handler {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/register", userHandler.Register).Methods(http.MethodPost)
	router.HandleFunc("/api/login", userHandler.Login).Methods(http.MethodPost)
	// router.HandleFunc("/logout", userHandler.Logout).Methods(http.MethodPost)

	router.HandleFunc("/api/posts", postHandler.CreatePost).Methods(http.MethodPost)
	router.HandleFunc("/api/posts", postHandler.ListPosts).Methods(http.MethodGet)
	router.HandleFunc("/api/posts/{category}", postHandler.ListPostsByCategory).Methods(http.MethodGet)
	router.HandleFunc("/api/post/{post_id}", postHandler.GetPost).Methods(http.MethodGet)
	router.HandleFunc("/api/post/{post_id}", postHandler.AddComment).Methods(http.MethodPost)
	router.HandleFunc("/api/post/{post_id}/{comment_id}", postHandler.DeleteComment).Methods(http.MethodDelete)
	router.HandleFunc("/api/post/{post_id}/upvote", postHandler.UpvotePost).Methods(http.MethodGet)
	router.HandleFunc("/api/post/{post_id}/downvote", postHandler.DownvotePost).Methods(http.MethodGet)
	router.HandleFunc("/api/post/{post_id}/unvote", postHandler.UnvotePost).Methods(http.MethodGet)
	router.HandleFunc("/api/post/{post_id}", postHandler.DeletePost).Methods(http.MethodDelete)
	router.HandleFunc("/api/user/{username}", postHandler.PostsByUser).Methods(http.MethodGet)

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))).Methods(http.MethodGet)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/html/index.html")
	}).Methods("GET")

	muxmwr := middleware.AccessLog(logger, router)
	muxmwr = middleware.Panic(muxmwr)

	return muxmwr
}

func main() {
	sm := session.NewSessionsManager()
	userRepo := user.NewMemoryRepo()
	postRepo := post.NewMemoryRepo()
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
	configuredRouter := configureRoutes(userHandler, postHandler, logger)
	fmt.Printf("Starting server at :%s", port)
	if err := http.ListenAndServe(":"+port, configuredRouter); err != nil {
		logger.Errorf("Server error: %v", err)
	}
}
