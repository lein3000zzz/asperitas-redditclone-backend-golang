package handlers

import (
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"net/http"
	"redditclone/pkg/session"
	"redditclone/pkg/user"
	"redditclone/pkg/utils"
)

type UserHandler struct {
	UserRepo user.UserRepo
	Logger   *zap.SugaredLogger
	Sessions *session.SessionsManager
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request user.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []map[string]interface{}{
				{
					"location": "body",
					"param":    "username",
					"value":    request.Username,
					"msg":      "already exists",
				},
			},
		})
		return
	}
	u, err := h.UserRepo.Register(request.Username, request.Password)
	if err != nil {
		if errors.Is(err, user.ErrAlreadyExists) {
			utils.WriteJSON(w, http.StatusConflict, map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"location": "body",
						"param":    "username",
						"value":    request.Username,
						"msg":      "already exists",
					},
				},
			})
		}
		if errors.Is(err, utils.ErrGenerateID) {
			utils.WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "error generating user id"})
		}
		return
	}

	err = utils.SendJwtToken(w, h.UserRepo.GenerateUserToken(*u))
	if err != nil {
		h.Logger.Errorf("failed to send JWT token: %v", err)
		return
	}
	h.Logger.Infof("Registered user %s", request.Username)

	sess, errCreate := h.Sessions.Create(w, u.ID)
	if errCreate != nil {
		h.Logger.Errorf("failed to create session: %v", errCreate)
		return
	}
	h.Logger.Infof("created session for %v", sess.UserID)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request user.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{"message": "invalid payload"})
		return
	}
	u, err := h.UserRepo.Authorize(request.Username, request.Password)
	if err != nil {
		if errors.Is(err, user.ErrNoUser) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"message": "user not found"})
		}
		if errors.Is(err, user.ErrBadPass) {
			utils.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"message": "invalid password"})
		}
		return
	}

	sess, errCreate := h.Sessions.Create(w, u.ID)
	if errCreate != nil {
		h.Logger.Errorf("failed to create session: %v", errCreate)
		return
	}
	h.Logger.Infof("created session for %v", sess.UserID)

	err = utils.SendJwtToken(w, h.UserRepo.GenerateUserToken(*u))
	if err != nil {
		h.Logger.Errorf("failed to send JWT token: %v", err)
		err := h.Sessions.DestroyCurrent(w, r)
		if err != nil {
			h.Logger.Errorf("failed to destroy session: %v", err)
		}
		return
	}
	h.Logger.Infof("Logged in user %s", request.Username)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	err := h.Sessions.DestroyCurrent(w, r)
	if err != nil {
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
