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

const (
	paramCategory      = "category"
	paramPostID        = "post_id"
	paramUser          = "user"
	paramID            = "id"
	paramUsername      = "username"
	paramUpvoteScore   = 1
	paramUnvoteScore   = 0
	paramDownvoteScore = -1
)

type PostHandler struct {
	PostRepo post.PostRepo
	Logger   *zap.SugaredLogger
	Sessions *session.SessionsManager
}

func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts := h.PostRepo.GetPosts()
	utils.WriteJSON(w, http.StatusOK, posts)
}

func (h *PostHandler) ListPostsByCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars[paramCategory]
	posts := h.PostRepo.GetPostsByCategory(category)
	utils.WriteJSON(w, http.StatusOK, posts)
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.GetClaimsByKey(r, paramUser)

	if err != nil {
		if errors.Is(err, utils.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		}
		return
	}

	username, ok1 := userData[paramUsername].(string)
	userID, ok2 := userData[paramID].(string)

	if !ok1 || !ok2 || username == "" || userID == "" {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	var request post.NewPostRequest

	if errDecoder := json.NewDecoder(r.Body).Decode(&request); errDecoder != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{"message": "invalid payload"})
		h.Logger.Errorf("ERROR with json decoding: %v", errDecoder)
		return
	}
	newPost, err := h.PostRepo.CreatePost(request, username, userID)
	if err != nil {
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "error creating post"})
		return
	}
	utils.WriteJSON(w, http.StatusCreated, *newPost)

	h.Logger.Infof("created post by %s: %v", username, *newPost)
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars[paramPostID]
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
	userData, err := utils.GetClaimsByKey(r, paramUser)

	if err != nil {
		if errors.Is(err, utils.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		}
		return
	}

	username, ok1 := userData[paramUsername].(string)
	userID, ok2 := userData[paramID].(string)

	if !ok1 || !ok2 || username == "" || userID == "" {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	vars := mux.Vars(r)
	id := vars[paramPostID]

	var req struct {
		Comment string `json:"comment"`
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil || req.Comment == "" {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{"message": "invalid payload"})
		h.Logger.Errorf("ERROR with json decoding: %v", err)
		return
	}
	commentedPost, err := h.PostRepo.AddComment(id, username, userID, req.Comment)
	if err != nil {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		if errors.Is(err, utils.ErrGenerateID) {
			utils.WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "error creating comment"})
		}
		return
	}
	utils.WriteJSON(w, http.StatusCreated, *commentedPost)
	h.Logger.Infof("commented post by %s: %s", username, req.Comment)
}

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.GetClaimsByKey(r, paramUser)

	if err != nil {
		if errors.Is(err, utils.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		}
		return
	}

	username, ok1 := userData[paramUsername].(string)
	userID, ok2 := userData[paramID].(string)

	if !ok1 || !ok2 || username == "" || userID == "" {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	vars := mux.Vars(r)
	postID := vars[paramPostID]
	commentID := vars["comment_id"]

	editedPost, err := h.PostRepo.DeleteComment(postID, commentID, userID)
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
	h.Logger.Infof("Deleted comment by %s: comment: %s, post: %s", username, commentID, postID)
}

func (h *PostHandler) votePost(w http.ResponseWriter, r *http.Request, action int) {
	userData, err := utils.GetClaimsByKey(r, paramUser)

	if err != nil {
		if errors.Is(err, utils.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		}
		return
	}

	userID, ok := userData[paramID].(string)
	if !ok || userID == "" {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	vars := mux.Vars(r)
	postID := vars[paramPostID]

	votedPost, err := h.PostRepo.VotePost(postID, userID, action)
	if err != nil {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		return
	}
	utils.WriteJSON(w, http.StatusOK, votedPost)
	h.Logger.Infof("Voted post by %s, %s, %d", userID, postID, action)
}

func (h *PostHandler) UpvotePost(w http.ResponseWriter, r *http.Request) {
	h.votePost(w, r, paramUpvoteScore)
}

func (h *PostHandler) DownvotePost(w http.ResponseWriter, r *http.Request) {
	h.votePost(w, r, paramDownvoteScore)
}

func (h *PostHandler) UnvotePost(w http.ResponseWriter, r *http.Request) {
	h.votePost(w, r, paramUnvoteScore)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.GetClaimsByKey(r, paramUser)

	if err != nil {
		if errors.Is(err, utils.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		}
		return
	}

	userID, ok := userData[paramID].(string)
	if !ok || userID == "" {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	vars := mux.Vars(r)
	postID := vars[paramPostID]

	isPostRemoved, err := h.PostRepo.DeletePost(postID, userID)

	if err != nil {
		if errors.Is(err, post.ErrPostNotFound) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{"message": "post not found"})
		}
		if errors.Is(err, post.ErrUnauthorized) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"message": "unauthorized"})
		}
		return
	}

	if isPostRemoved {
		utils.WriteJSON(w, http.StatusNoContent, map[string]string{"message": "success"})
		h.Logger.Infof("Deleted post by %s: post: %s", userID, postID)
	} else {
		h.Logger.Errorf("ERROR! Havent deleted post by %s: post: %s", userID, postID)
	}
}

func (h *PostHandler) PostsByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars[paramUsername]

	posts := h.PostRepo.PostsByUser(username)

	utils.WriteJSON(w, http.StatusOK, posts)
}
