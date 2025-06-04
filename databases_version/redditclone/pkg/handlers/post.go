package handlers

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"redditclone/pkg/post"
	"redditclone/pkg/session"
	"redditclone/pkg/utils"
)

type PostHandler struct {
	PostRepo post.PostRepo
	Logger   *zap.SugaredLogger
	Sessions session.SessionManager
}

func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts := h.PostRepo.GetPosts()
	utils.WriteJSON(w, http.StatusOK, posts)
}

func (h *PostHandler) ListPostsByCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars["category"]
	posts := h.PostRepo.GetPostsByCategory(category)
	utils.WriteJSON(w, http.StatusOK, posts)
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	currentSession, err := h.Sessions.Check(r)

	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	err = h.Sessions.UpdateCookie(w, r)

	if err != nil {
		h.Logger.Errorf("failed to update cookie: %v", err)
		// Ну не обновили и не обновили. Не выкидывать же юзера и не прерывать же его действие
	}

	var request post.NewPostRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{"message": "invalid payload"})
		return
	}
	newPost := h.PostRepo.CreatePost(request, currentSession.Username, currentSession.UserID)

	utils.WriteJSON(w, http.StatusCreated, *newPost)

	h.Logger.Infof("created post by %s: %v", currentSession.Username, *newPost)
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["post_id"]
	postByID, err := h.PostRepo.GetPost(id)
	if err != nil {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		return
	}
	utils.WriteJSON(w, http.StatusOK, postByID)
}

func (h *PostHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	currentSession, err := h.Sessions.Check(r)

	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	err = h.Sessions.UpdateCookie(w, r)

	if err != nil {
		h.Logger.Errorf("failed to update cookie: %v", err)
		// Ну не обновили и не обновили. Не выкидывать же юзера и не прерывать же его действие
	}

	vars := mux.Vars(r)
	id := vars["post_id"]

	var req struct {
		Comment string `json:"comment"`
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil || req.Comment == "" {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{"message": "invalid payload"})
		return
	}
	commentedPost, err := h.PostRepo.AddComment(id, currentSession.Username, currentSession.UserID, req.Comment)
	if err != nil {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		return
	}
	utils.WriteJSON(w, http.StatusCreated, *commentedPost)
	h.Logger.Infof("commented post by %s: %s", currentSession.Username, req.Comment)
}

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	currentSession, err := h.Sessions.Check(r)

	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	err = h.Sessions.UpdateCookie(w, r)

	if err != nil {
		h.Logger.Errorf("failed to update cookie: %v", err)
		// Ну не обновили и не обновили. Не выкидывать же юзера и не прерывать же его действие
	}

	vars := mux.Vars(r)
	postID := vars["post_id"]
	commentID := vars["comment_id"]

	editedPost, err := h.PostRepo.DeleteComment(postID, commentID, currentSession.UserID)
	if err != nil {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		if errors.Is(err, post.ErrCommentNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "comment not found"})
		}
		if errors.Is(err, post.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"message": "unauthorized"})
		}
		return
	}

	utils.WriteJSON(w, http.StatusOK, editedPost)
	h.Logger.Infof("Deleted comment by %s: comment: %s, post: %s", currentSession.Username, commentID, postID)
}

func (h *PostHandler) votePost(w http.ResponseWriter, r *http.Request, action int) {
	currentSession, err := h.Sessions.Check(r)

	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	err = h.Sessions.UpdateCookie(w, r)

	if err != nil {
		h.Logger.Errorf("failed to update cookie: %v", err)
		// Ну не обновили и не обновили. Не выкидывать же юзера и не прерывать же его действие
	}

	vars := mux.Vars(r)
	postID := vars["post_id"]

	votedPost, err := h.PostRepo.VotePost(postID, currentSession.UserID, action)
	if err != nil {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		return
	}
	utils.WriteJSON(w, http.StatusOK, votedPost)
	h.Logger.Infof("Voted post by %s, %s, %d", currentSession.UserID, postID, action)
}

func (h *PostHandler) UpvotePost(w http.ResponseWriter, r *http.Request) {
	h.votePost(w, r, 1)
}

func (h *PostHandler) DownvotePost(w http.ResponseWriter, r *http.Request) {
	h.votePost(w, r, -1)
}

func (h *PostHandler) UnvotePost(w http.ResponseWriter, r *http.Request) {
	h.votePost(w, r, 0)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	currentSession, err := h.Sessions.Check(r)

	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	err = h.Sessions.UpdateCookie(w, r)

	if err != nil {
		h.Logger.Errorf("failed to update cookie: %v", err)
		// Ну не обновили и не обновили. Не выкидывать же юзера и не прерывать же его действие
	}

	vars := mux.Vars(r)
	postID := vars["post_id"]

	isPostRemoved, err := h.PostRepo.DeletePost(postID, currentSession.UserID)

	if err != nil || !isPostRemoved {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		if errors.Is(err, post.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"message": "unauthorized"})
		}
		h.Logger.Errorf("ERROR! Havent deleted post by %s: post: %s", currentSession.UserID, postID)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "success"})
	h.Logger.Infof("Deleted post by %s: post: %s", currentSession.UserID, postID)
}

func (h *PostHandler) PostsByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	posts := h.PostRepo.PostsByUser(username)

	utils.WriteJSON(w, http.StatusOK, posts)
}
