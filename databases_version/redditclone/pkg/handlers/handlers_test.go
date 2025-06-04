package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"redditclone/pkg/utils"
	"redditclone/pkg/utils/mocks"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"go.uber.org/zap/zaptest"
	"redditclone/pkg/post"
	"redditclone/pkg/session"
	"redditclone/pkg/user"
)

func TestPostHandler_ListPosts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockPostRepo(ctrl)
	sample := []*post.Post{{ID: "1"}, {ID: "2"}}
	mockRepo.EXPECT().GetPosts().Return(sample)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	handler.ListPosts(w, req)
	res := w.Result()
	defer utils.CloseBody(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
	var got []post.Post
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(got) != 2 || got[0].ID != "1" || got[1].ID != "2" {
		t.Errorf("unexpected body: %+v", got)
	}
}

func TestPostHandler_ListPostsByCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	sample := []post.Post{{ID: "1", Category: "fun"}}
	mockRepo.EXPECT().GetPostsByCategory("fun").Return(sample)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/posts/category/fun", nil),
		map[string]string{"category": "fun"})
	w := httptest.NewRecorder()

	handler.ListPostsByCategory(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
	var posts []post.Post
	err := json.NewDecoder(w.Body).Decode(&posts)
	if err != nil {
		return
	}
	if len(posts) == 0 || posts[0].Category != "fun" {
		t.Errorf("unexpected body: %+v", posts)
	}
}

func TestPostHandler_CreatePost_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSess := mocks.NewMockSessionManager(ctrl)
	mockSess.EXPECT().Check(gomock.Any()).Return((*session.Session)(nil), errors.New("no session"))

	handler := &PostHandler{
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	handler.CreatePost(w, req)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Result().StatusCode)
	}
}

func TestPostHandler_CreatePost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)

	sess := &session.Session{Username: "u", UserID: "uid"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)

	reqBody := post.NewPostRequest{
		Category: "fun",
		Type:     "text",
		Title:    "Title",
		Text:     "body",
	}
	marshalledBody, err := json.Marshal(reqBody)
	if err != nil {
		return
	}
	newP := &post.Post{ID: "new1", Title: "T"}
	mockRepo.EXPECT().
		CreatePost(post.NewPostRequest{Category: "fun", Type: "text", Title: "Title", Text: "body"}, "u", "uid").
		Return(newP)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(marshalledBody))
	w := httptest.NewRecorder()

	handler.CreatePost(w, req)
	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Result().StatusCode)
	}
	var got post.Post
	err = json.NewDecoder(w.Body).Decode(&got)
	if err != nil {
		return
	}
	if got.ID != "new1" || got.Title != "T" {
		t.Errorf("unexpected post: %+v", got)
	}
}

func TestPostHandler_GetPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockPostRepo(ctrl)

	mockRepo.EXPECT().
		GetPost("42").
		Return(post.Post{}, post.ErrPostNotFound)
	handler := &PostHandler{
		PostRepo: mockRepo,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}
	req1 := mux.SetURLVars(httptest.NewRequest("GET", "/posts/42", nil), map[string]string{"post_id": "42"})
	w1 := httptest.NewRecorder()
	handler.GetPost(w1, req1)
	if w1.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w1.Result().StatusCode)
	}

	sample := post.Post{ID: "42", Title: "Test"}
	mockRepo.EXPECT().
		GetPost("42").
		Return(sample, nil)
	req2 := mux.SetURLVars(httptest.NewRequest("GET", "/posts/42", nil), map[string]string{"post_id": "42"})
	w2 := httptest.NewRecorder()
	handler.GetPost(w2, req2)
	if w2.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Result().StatusCode)
	}
	var got post.Post
	err := json.NewDecoder(w2.Body).Decode(&got)
	if err != nil {
		return
	}
	if got.ID != "42" || got.Title != "Test" {
		t.Errorf("unexpected body: %+v", got)
	}
}

func TestPostHandler_AddComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)

	commentReq := "Good Post"
	expectedPost := &post.Post{ID: "1"}
	mockRepo.EXPECT().AddComment("1", "user", "uid", commentReq).Return(expectedPost, nil)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	reqBody := map[string]string{
		"comment": commentReq,
	}

	marshalledBody, err := json.Marshal(reqBody)
	if err != nil {
		return
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/posts/1/comments", bytes.NewBuffer(marshalledBody)), map[string]string{"post_id": "1"})
	w := httptest.NewRecorder()

	handler.AddComment(w, req)
	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Result().StatusCode)
	}
}

func TestPostHandler_DeleteComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)

	expectedPost := &post.Post{ID: "1"}
	mockRepo.EXPECT().DeleteComment("1", "c1", "uid").Return(expectedPost, nil)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/posts/1/comments/c1", nil), map[string]string{"post_id": "1", "comment_id": "c1"})
	w := httptest.NewRecorder()

	handler.DeleteComment(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestPostHandler_UpvotePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)
	expectedPost := &post.Post{ID: "1", Score: 1}
	mockRepo.EXPECT().VotePost("1", "uid", 1).Return(expectedPost, nil)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/posts/1/upvote", nil), map[string]string{"post_id": "1"})
	w := httptest.NewRecorder()

	handler.UpvotePost(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestPostHandler_DownvotePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)
	expectedPost := &post.Post{ID: "1", Score: -1}
	mockRepo.EXPECT().VotePost("1", "uid", -1).Return(expectedPost, nil)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/posts/1/downvote", nil), map[string]string{"post_id": "1"})
	w := httptest.NewRecorder()

	handler.DownvotePost(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestPostHandler_UnvotePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)
	expectedPost := &post.Post{ID: "1", Score: 0}
	mockRepo.EXPECT().VotePost("1", "uid", 0).Return(expectedPost, nil)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/posts/1/unvote", nil), map[string]string{"post_id": "1"})
	w := httptest.NewRecorder()

	handler.UnvotePost(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestUserHandler_Register(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mocks.NewMockUserRepo(ctrl)
		mockSess := mocks.NewMockSessionManager(ctrl)
		logger := zaptest.NewLogger(t).Sugar()

		u := &user.User{ID: "id1", Username: "u"}
		mockRepo.EXPECT().Register("u", "p").Return(u, nil)
		mockSess.EXPECT().Create(gomock.Any(), "id1", "u").
			Return(&session.Session{UserID: "id1", Username: "u"}, nil)
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user": map[string]string{
				"user_id":  u.ID,
				"username": u.Username,
			},
		})
		mockRepo.EXPECT().GenerateUserToken(*u).Return(jwtToken)

		handler := &UserHandler{
			UserRepo: mockRepo,
			Sessions: mockSess,
			Logger:   logger,
		}

		body := map[string]string{
			"username": "u",
			"password": "p",
		}
		jsonBody, err := json.Marshal(body)

		if err != nil {
			return
		}

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonBody))

		handler.Register(w, req)
		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
	})
}

func TestUserHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	logger := zaptest.NewLogger(t).Sugar()

	userObj := &user.User{ID: "id", Username: "user", Password: "pass"}
	mockRepo.EXPECT().Authorize("user", "pass").Return(userObj, nil)
	mockSess.EXPECT().Create(gomock.Any(), "id", "user").Return(&session.Session{UserID: "id", Username: "user"}, nil)
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]string{"user_id": userObj.ID, "username": userObj.Username},
	})
	mockRepo.EXPECT().GenerateUserToken(*userObj).Return(jwtToken)

	handler := &UserHandler{
		UserRepo: mockRepo,
		Sessions: mockSess,
		Logger:   logger,
	}

	reqBody := map[string]string{
		"username": "user",
		"password": "pass",
	}
	marshalledBody, err := json.Marshal(reqBody)
	if err != nil {
		return
	}
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBuffer(marshalledBody))
	w := httptest.NewRecorder()
	handler.Login(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestUserHandler_Login_BadJSON(t *testing.T) {
	handler := &UserHandler{
		Logger: zaptest.NewLogger(t).Sugar(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString("invalid"))
	w := httptest.NewRecorder()
	handler.Login(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestUserHandler_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSess := mocks.NewMockSessionManager(ctrl)
	mockSess.EXPECT().Destroy(gomock.Any(), gomock.Any()).Return(nil)

	handler := &UserHandler{
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	w := httptest.NewRecorder()
	handler.Logout(w, req)
	if w.Result().StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Result().StatusCode)
	}
}

func TestPostHandler_DeletePost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid1"}

	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)

	mockRepo.EXPECT().DeletePost("1", "uid1").Return(true, nil)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/posts/1", nil), map[string]string{"post_id": "1"})
	w := httptest.NewRecorder()

	handler.DeletePost(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
}

func TestPostHandler_DeletePost_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid1"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)

	mockRepo.EXPECT().DeletePost("1", "uid1").Return(false, post.ErrPostNotFound)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/posts/1", nil), map[string]string{"post_id": "1"})
	w := httptest.NewRecorder()

	handler.DeletePost(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

func TestPostHandler_DeletePost_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	sess := &session.Session{Username: "user", UserID: "uid1"}
	mockSess.EXPECT().Check(gomock.Any()).Return(sess, nil)
	mockSess.EXPECT().UpdateCookie(gomock.Any(), gomock.Any()).Return(nil)

	mockRepo.EXPECT().DeletePost("1", "uid1").Return(false, post.ErrUnauthorized)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Sessions: mockSess,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/posts/1", nil), map[string]string{"post_id": "1"})
	w := httptest.NewRecorder()

	handler.DeletePost(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}

func TestPostHandler_PostsByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPostRepo(ctrl)
	expectedPosts := []post.Post{
		{ID: "1", Title: "Post1", Author: post.Author{Username: "testuser", ID: "uid1"}},
		{ID: "2", Title: "Post2", Author: post.Author{Username: "testuser", ID: "uid1"}},
	}
	mockRepo.EXPECT().PostsByUser("testuser").Return(expectedPosts)

	handler := &PostHandler{
		PostRepo: mockRepo,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}

	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/posts/user/testuser", nil), map[string]string{"username": "testuser"})
	w := httptest.NewRecorder()

	handler.PostsByUser(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
	var posts []post.Post
	if err := json.NewDecoder(res.Body).Decode(&posts); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(posts) != 2 || posts[0].Author.Username != "testuser" {
		t.Errorf("unexpected posts: %+v", posts)
	}
}

func TestUserHandler_Register_AlreadyExists(t *testing.T) {
	err := os.Setenv("JWT_SECRET", "my-test-secret")
	if err != nil {
		return
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockSess := mocks.NewMockSessionManager(ctrl)
	logger := zaptest.NewLogger(t).Sugar()

	mockRepo.EXPECT().Register("existingUser", "pass").Return((*user.User)(nil), user.ErrAlreadyExists)

	handler := &UserHandler{
		UserRepo: mockRepo,
		Sessions: mockSess,
		Logger:   logger,
	}

	body := map[string]string{
		"username": "existingUser",
		"password": "pass",
	}

	jsonBody, err := json.Marshal(body)

	if err != nil {
		return
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonBody))

	handler.Register(w, req)
	res := w.Result()
	defer utils.CloseBody(res.Body)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 already exists, got %d", res.StatusCode)
	}
}

func TestUserHandler_Register_InvalidJSON(t *testing.T) {
	handler := &UserHandler{
		Logger: zaptest.NewLogger(t).Sugar(),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(`{"username": "bad\bad", "password":""`))
	w := httptest.NewRecorder()

	handler.Register(w, req)
	res := w.Result()
	defer utils.CloseBody(res.Body)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 bad request, got %d", res.StatusCode)
	}
}
