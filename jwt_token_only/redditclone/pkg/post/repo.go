package post

import (
	"errors"
	"redditclone/pkg/utils"
	"sync"
	"time"
)

var (
	ErrPostNotFound    = errors.New("post not found")
	ErrCommentNotFound = errors.New("comment not found")
	ErrUnauthorized    = errors.New("unauthorized")
)

type PostMemoryRepo struct {
	sync.RWMutex
	Posts map[string]*Post
}

func NewMemoryRepo() *PostMemoryRepo {
	return &PostMemoryRepo{
		Posts: make(map[string]*Post),
	}
}

func (repo *PostMemoryRepo) GetPosts() []*Post {
	repo.RLock()
	defer repo.RUnlock()
	posts := make([]*Post, 0, len(repo.Posts))
	for _, post := range repo.Posts {
		posts = append(posts, post)
	}
	return posts
}

func (repo *PostMemoryRepo) GetPostsByCategory(category string) []Post {
	repo.RLock()
	defer repo.RUnlock()
	posts := make([]Post, 0)
	for _, post := range repo.Posts {
		if post.Category == category {
			posts = append(posts, *post)
		}
	}
	return posts
}

func (repo *PostMemoryRepo) CreatePost(request NewPostRequest, username, userID string) (*Post, error) {
	repo.Lock()
	defer repo.Unlock()
	postID, err := utils.GenerateID()
	if err != nil {
		return nil, err
	}
	createdTime := time.Now().UTC()
	newPost := &Post{
		ID:               postID,
		Author:           Author{Username: username, ID: userID},
		Category:         request.Category,
		Type:             request.Type,
		Title:            request.Title,
		Score:            1,
		Views:            1,
		Votes:            []Vote{{User: userID, Vote: 1}},
		Comments:         []Comment{},
		Created:          createdTime,
		UpvotePercentage: 100,
	}

	if request.Type == "link" {
		newPost.URL = request.URL
	} else {
		newPost.Text = request.Text
	}

	repo.Posts[postID] = newPost
	return newPost, nil
}

func (repo *PostMemoryRepo) GetPost(id string) (Post, error) {
	repo.RLock()
	defer repo.RUnlock()
	post, ok := repo.Posts[id]
	if !ok {
		return Post{}, ErrPostNotFound // делать указатель на пост в возвращаемом значении неудобно + inconsistent относительно того, что в других функциях
	}

	copyPost := *post

	return copyPost, nil
}

func (repo *PostMemoryRepo) AddComment(postID, username, userID, comment string) (*Post, error) {
	repo.Lock()
	defer repo.Unlock()
	commentedPost, ok := repo.Posts[postID]
	if !ok {
		return nil, ErrPostNotFound
	}
	commentID, err := utils.GenerateID()
	if err != nil {
		return nil, err
	}
	newComment := Comment{
		ID:      commentID,
		Body:    comment,
		Created: time.Now().UTC(),
		Author: Author{
			Username: username,
			ID:       userID,
		},
	}
	commentedPost.Comments = append(commentedPost.Comments, newComment)
	// repo.Posts[postID] = commentedPost
	return commentedPost, nil
}

func (repo *PostMemoryRepo) DeleteComment(postID, commentID, userID string) (*Post, error) {
	repo.Lock()
	defer repo.Unlock()
	removedCommentPost, ok := repo.Posts[postID]
	if !ok {
		return nil, ErrPostNotFound
	}
	if removedCommentPost.Author.ID != userID {
		return nil, ErrUnauthorized
	}
	newComments := []Comment{}
	found := false
	for _, c := range removedCommentPost.Comments {
		if c.ID == commentID {
			found = true
			continue
		}
		newComments = append(newComments, c)
	}
	if !found {
		return nil, ErrCommentNotFound
	}
	removedCommentPost.Comments = newComments
	// repo.Posts[postID] = removedCommentPost
	return removedCommentPost, nil
}

func (repo *PostMemoryRepo) VotePost(postID, userID string, action int) (*Post, error) {
	repo.Lock()
	defer repo.Unlock()
	votedPost, ok := repo.Posts[postID]
	if !ok {
		return nil, ErrPostNotFound
	}

	var existingVote *Vote
	var existingIndex = -1
	for i, v := range votedPost.Votes {
		if v.User == userID {
			existingVote = &votedPost.Votes[i]
			existingIndex = i
			break
		}
	}

	switch action {
	case 1:
		{
			if existingVote != nil {
				if existingVote.Vote == 1 {
					// Ничего не меняем
				} else {
					// Изменение с даунвоута на апвоут.
					votedPost.Score += 2
					votedPost.Votes[existingIndex].Vote = 1
				}
			} else {
				votedPost.Score++
				votedPost.Votes = append(votedPost.Votes, Vote{User: userID, Vote: 1})
			}
		}
	case -1:
		{
			if existingVote != nil {
				if existingVote.Vote == -1 {
					// Ничего не меняем
				} else {
					// Изменение с апвоута на даунвоут.
					votedPost.Score -= 2
					votedPost.Votes[existingIndex].Vote = -1
				}
			} else {
				votedPost.Score--
				votedPost.Votes = append(votedPost.Votes, Vote{User: userID, Vote: -1})
			}
		}
	case 0:
		{
			if existingVote != nil {
				switch existingVote.Vote {
				case 1:
					votedPost.Score--
				case -1:
					votedPost.Score++
				}
				votedPost.Votes = append(votedPost.Votes[:existingIndex], votedPost.Votes[existingIndex+1:]...)
			}
		}
	}
	// Можно подсчитать сразу в процессе поиска голоса от юзера, а потом, в зависимости от действия,
	// менять переменные totalVotes и upvotes, но я решил отдельно вынести
	totalVotes := len(votedPost.Votes)
	upvotes := 0
	for _, v := range votedPost.Votes {
		if v.Vote == 1 {
			upvotes++
		}
	}

	if totalVotes == 0 {
		votedPost.UpvotePercentage = 100
	} else {
		votedPost.UpvotePercentage = int((float64(upvotes) / float64(totalVotes)) * 100)
	}

	return votedPost, nil
}

func (repo *PostMemoryRepo) DeletePost(postID, userID string) (bool, error) {
	repo.Lock()
	defer repo.Unlock()
	if _, ok := repo.Posts[postID]; !ok {
		return false, ErrPostNotFound
	}

	if repo.Posts[postID].Author.ID != userID {
		return false, ErrUnauthorized
	}

	delete(repo.Posts, postID)
	return true, nil
}

func (repo *PostMemoryRepo) PostsByUser(username string) []Post {
	repo.RLock()
	defer repo.RUnlock()
	posts := make([]Post, 0)
	for _, post := range repo.Posts {
		if post.Author.Username == username {
			posts = append(posts, *post)
		}
	}
	return posts
}
