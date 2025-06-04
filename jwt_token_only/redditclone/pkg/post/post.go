package post

import "time"

type Comment struct {
	ID     string `json:"id"`
	Author struct {
		Username string `json:"username"`
		ID       string `json:"id"`
	} `json:"author"`
	Body    string    `json:"body"`
	Created time.Time `json:"created"`
}

type Vote struct {
	User string `json:"user"`
	Vote int    `json:"vote"`
}

type Author struct {
	Username string `json:"username"`
	ID       string `json:"id"`
}

type Post struct {
	Score            int       `json:"score"`
	Views            int       `json:"views"`
	Type             string    `json:"type"`
	Title            string    `json:"title"`
	URL              string    `json:"url"`
	Author           Author    `json:"author"`
	Category         string    `json:"category"`
	Text             string    `json:"text"`
	Votes            []Vote    `json:"votes"`
	Comments         []Comment `json:"comments"`
	Created          time.Time `json:"created"`
	UpvotePercentage int       `json:"upvotePercentage"`
	ID               string    `json:"id"`
}

type NewPostRequest struct {
	Category string `json:"category"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	URL      string `json:"url"`
}

type PostRepo interface {
	GetPost(id string) (Post, error)
	GetPosts() []*Post
	GetPostsByCategory(category string) []Post
	CreatePost(request NewPostRequest, username, userID string) (*Post, error)
	AddComment(postID, username, userID, comment string) (*Post, error)
	DeleteComment(postID, commentID, userID string) (*Post, error)
	DeletePost(postID, userID string) (bool, error)
	PostsByUser(username string) []Post
	VotePost(postID, userID string, vote int) (*Post, error)
}
