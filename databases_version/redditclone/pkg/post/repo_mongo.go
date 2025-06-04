package post

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"time"

	"redditclone/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// По сути, можно было бы и без своего айди на постах обойтись, оставив чисто тот _id, что генерируется в монго
// через примитив, но решил оставить так, как было изначально, не меняя начальную структуру, где идет "id".
// Плюс, в примере asperitas тоже "id" приходит в ответе.
const (
	categoryKey         = "category"
	idKey               = "id"
	commentsKey         = "comments"
	scoreKey            = "score"
	votesKey            = "votes"
	upvotePercentageKey = "upvotePercentage"
	authUsernameKey     = "author.username"
)

var (
	ErrPostNotFound    = errors.New("post not found")
	ErrCommentNotFound = errors.New("comment not found")
	ErrUnauthorized    = errors.New("unauthorized")
)

type PostMongoRepo struct {
	collection *mongo.Collection
	logger     *zap.SugaredLogger
}

func NewMongoRepo(collection *mongo.Collection, logger *zap.SugaredLogger) *PostMongoRepo {
	return &PostMongoRepo{
		collection: collection,
		logger:     logger,
	}
}

func (repo *PostMongoRepo) GetPosts() []*Post {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	postsFromDB, err := repo.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil
	}

	defer utils.HandleMongoCursorClose(postsFromDB, ctx)

	posts := make([]*Post, 0)
	for postsFromDB.Next(ctx) {
		var post Post
		if err := postsFromDB.Decode(&post); err != nil {
			repo.logger.Errorf("Error decoding post: %v", err)
			continue
		}
		repo.logger.Debugf("Successfully decoded post: %s", post.ID)
		posts = append(posts, &post)
	}
	repo.logger.Infof("Fetched %d posts from DB", len(posts))
	return posts
}

func (repo *PostMongoRepo) GetPostsByCategory(category string) []Post {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{categoryKey: category}
	postsFromDB, err := repo.collection.Find(ctx, filter)
	if err != nil {
		return nil
	}

	defer utils.HandleMongoCursorClose(postsFromDB, ctx)

	posts := make([]Post, 0)
	for postsFromDB.Next(ctx) {
		var post Post
		if err := postsFromDB.Decode(&post); err != nil {
			repo.logger.Errorf("Error decoding post: %v", err)
			continue
		}
		repo.logger.Debugf("Successfully decoded post: %s", post.ID)
		posts = append(posts, post)
	}
	repo.logger.Infof("Fetched %d posts from DB", len(posts))
	return posts
}

func (repo *PostMongoRepo) CreatePost(request NewPostRequest, username, userID string) *Post {
	postID := utils.GenerateID()
	createdTime := time.Now().UTC()
	newPost := &Post{
		ID:       postID,
		Author:   Author{Username: username, ID: userID},
		Category: request.Category,
		Type:     request.Type,
		Title:    request.Title,
		Score:    1,
		Views:    1,
		Votes:    []Vote{{User: userID, Vote: 1}},
		Comments: []Comment{},
		Created:  createdTime,
	}
	if request.Type == "link" {
		newPost.URL = request.URL
	} else {
		newPost.Text = request.Text
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := repo.collection.InsertOne(ctx, newPost); err != nil {
		repo.logger.Errorf("Error inserting new post: %v", err)
		return nil
	}
	repo.logger.Debugf("Successfully created post: %s", newPost.ID)
	return newPost
}

func (repo *PostMongoRepo) GetPost(id string) (Post, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var post Post
	err := repo.collection.FindOne(ctx, bson.M{idKey: id}).Decode(&post)

	if err != nil {
		repo.logger.Errorf("Error finding post: %v", err)
		return Post{}, ErrPostNotFound
	}
	repo.logger.Debugf("Successfully fetched post: %s", post.ID)

	return post, nil
}

func (repo *PostMongoRepo) AddComment(postID, username, userID, comment string) (*Post, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{idKey: postID}

	var post Post
	err := repo.collection.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		repo.logger.Errorf("Error finding post: %v", err)
		return nil, ErrPostNotFound
	}
	repo.logger.Debugf("Successfully fetched post: %s", post.ID)

	newComment := Comment{
		ID:      utils.GenerateID(),
		Body:    comment,
		Created: time.Now().UTC(),
		Author:  Author{Username: username, ID: userID},
	}

	update := bson.M{"$push": bson.M{commentsKey: newComment}}
	_, err = repo.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		repo.logger.Errorf("Error updating post with new comment: %v", err)
		return nil, fmt.Errorf("fail AddComment: %v", err)
	}
	repo.logger.Debugf("Successfully updated post with new comment: %s", post.ID)

	post.Comments = append(post.Comments, newComment)

	return &post, nil
}

func (repo *PostMongoRepo) DeleteComment(postID, commentID, userID string) (*Post, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{idKey: postID}

	var post Post
	err := repo.collection.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		repo.logger.Errorf("Error finding post: %v", err)
		return nil, ErrPostNotFound
	}
	repo.logger.Debugf("Successfully fetched post: %s", post.ID)

	commentIndex := -1
	for i, c := range post.Comments {
		if c.ID == commentID {
			commentIndex = i
			break
		}
	}

	if commentIndex == -1 || post.Comments[commentIndex].Author.ID != userID {
		if commentIndex == -1 {
			repo.logger.Errorf("Comment not found: %s", commentID)
			return nil, ErrCommentNotFound
		}
		repo.logger.Errorf("Unauthorized to delete comment: %s", commentID)
		return nil, ErrUnauthorized
	}

	update := bson.M{"$pull": bson.M{commentsKey: bson.M{idKey: commentID}}}
	_, err = repo.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		repo.logger.Errorf("Error deleting comment: %v", err)
		return nil, fmt.Errorf("fail DeleteComment: %v", err)
	}
	repo.logger.Debugf("Successfully deleted comment: %s", commentID)
	err = repo.collection.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		repo.logger.Errorf("Error finding post after comment deletion: %v", err)
		return nil, ErrPostNotFound
	}
	repo.logger.Debugf("Successfully fetched post after comment deletion: %s", post.ID)
	return &post, nil
}

func (repo *PostMongoRepo) VotePost(postID, userID string, vote int) (*Post, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{idKey: postID}

	var post Post
	err := repo.collection.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		repo.logger.Errorf("Error finding post: %v", err)
		return nil, ErrPostNotFound
	}
	repo.logger.Debugf("Successfully fetched post: %s", post.ID)

	existingIndex := -1
	for i, v := range post.Votes {
		if v.User == userID {
			existingIndex = i
			break
		}
	}

	switch vote {
	case 1:
		// upvote
		if existingIndex != -1 {
			// уже был голос
			if post.Votes[existingIndex].Vote != 1 {
				post.Score += 2
				post.Votes[existingIndex].Vote = 1
			}
		} else {
			// новый голос
			post.Score++
			post.Votes = append(post.Votes, Vote{User: userID, Vote: 1})
		}
	case -1:
		// downvote
		if existingIndex != -1 {
			// уже был голос
			if post.Votes[existingIndex].Vote != -1 {
				post.Score -= 2
				post.Votes[existingIndex].Vote = -1
			}
		} else {
			// новый голос
			post.Score--
			post.Votes = append(post.Votes, Vote{User: userID, Vote: -1})
		}
	case 0:
		// unvote
		if existingIndex != -1 {
			// уже был голос
			switch post.Votes[existingIndex].Vote {
			case 1:
				// unvote upvote
				post.Score--
			case -1:
				// unvote downvote
				post.Score++
			}
			post.Votes = append(post.Votes[:existingIndex], post.Votes[existingIndex+1:]...)
		}
	}

	repo.updateUpvotePercentage(&post)

	update := bson.M{"$set": bson.M{
		scoreKey:            post.Score,
		votesKey:            post.Votes,
		upvotePercentageKey: post.UpvotePercentage,
	}}
	_, err = repo.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		repo.logger.Errorf("Error updating post: %v", err)
		return nil, fmt.Errorf("fail VotePost: %v", err)
	}
	repo.logger.Debugf("Successfully updated post: %s", post.ID)

	err = repo.collection.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		repo.logger.Errorf("Error finding post: %v", err)
		return nil, ErrPostNotFound
	}
	return &post, nil
}

func (repo *PostMongoRepo) updateUpvotePercentage(post *Post) {
	totalVotes := len(post.Votes)
	upvotes := 0

	for _, v := range post.Votes {
		if v.Vote == 1 {
			upvotes++
		}
	}

	if totalVotes == 0 {
		post.UpvotePercentage = 100
	} else {
		post.UpvotePercentage = int((float64(upvotes) / float64(totalVotes)) * 100)
	}

	repo.logger.Debugf("Updated upvote percentage for post %s: %d", post.ID, post.UpvotePercentage)
}

func (repo *PostMongoRepo) DeletePost(postID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{idKey: postID}

	var post Post
	err := repo.collection.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		repo.logger.Errorf("Error finding post: %v", err)
		return false, ErrPostNotFound
	}
	repo.logger.Debugf("Successfully fetched post: %s", post.ID)

	if post.Author.ID != userID {
		repo.logger.Errorf("Unauthorized to delete post: %s", postID)
		return false, ErrUnauthorized
	}

	res, err := repo.collection.DeleteOne(ctx, filter)
	if err != nil {
		repo.logger.Errorf("Error deleting post: %v", err)
		return false, err
	}
	repo.logger.Debugf("Successfully deleted post: %s", postID)

	if res.DeletedCount == 0 {
		repo.logger.Errorf("No post found to delete: %s", postID)
		return false, ErrPostNotFound
	}
	return true, nil
}

func (repo *PostMongoRepo) PostsByUser(username string) []Post {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{authUsernameKey: username}

	postsFromDB, err := repo.collection.Find(ctx, filter)
	posts := make([]Post, 0)
	if err != nil {
		repo.logger.Errorf("Error finding posts by user: %v", err)
		return posts
	}
	repo.logger.Infof("Successfully fetched posts by user: %s", username)

	defer utils.HandleMongoCursorClose(postsFromDB, ctx)

	for postsFromDB.Next(ctx) {
		var post Post
		if err := postsFromDB.Decode(&post); err != nil {
			repo.logger.Errorf("Error decoding post: %v", err)
			continue
		}
		posts = append(posts, post)
	}
	repo.logger.Infof("Fetched %d posts by user: %s", len(posts), username)

	return posts
}
