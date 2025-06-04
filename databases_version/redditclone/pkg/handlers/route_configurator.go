package handlers

import (
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"redditclone/pkg/utils/middleware"
)

func ConfigureRoutes(userHandler *UserHandler, postHandler *PostHandler, logger *zap.SugaredLogger) http.Handler {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/register", userHandler.Register).Methods(http.MethodPost)
	router.HandleFunc("/api/login", userHandler.Login).Methods(http.MethodPost)
	router.HandleFunc("/logout", userHandler.Logout).Methods(http.MethodGet)

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
