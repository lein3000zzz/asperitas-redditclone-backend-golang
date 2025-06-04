package post

import (
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.uber.org/zap"
	"testing"
	"time"
)

var (
	nilLogger = zap.NewNop().Sugar()
)

func TestGetPost_NotFound(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("post not found", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}})
		repo := NewMongoRepo(mt.Coll, nilLogger)

		_, err := repo.GetPost("no-such-id")
		if !errors.Is(err, ErrPostNotFound) {
			t.Fatalf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestGetPost_Found(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("post found", func(mt *mtest.T) {
		first := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post123"},
				{Key: "title", Value: "Hello World"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 5},
				{Key: "views", Value: 10},
				{Key: "author", Value: Author{Username: "alice", ID: "user42"}},
				{Key: "text", Value: "This is a test post."},
				{Key: "votes", Value: []Vote{}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Date(2025, 5, 5, 12, 0, 0, 0, time.UTC)},
				{Key: "upvotePercentage", Value: 100},
			},
		)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(first, end)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.GetPost("post123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.ID != "post123" || post.Title != "Hello World" {
			t.Errorf("unexpected post: %+v", post)
		}
	})
}

func TestGetPosts_Various(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("multiple posts", func(mt *mtest.T) {
		batch1 := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{{Key: "id", Value: "1"}, {Key: "title", Value: "A"}},
			bson.D{{Key: "id", Value: "2"}, {Key: "title", Value: "B"}},
		)
		batch2 := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(batch1, batch2)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		posts := repo.GetPosts()
		if len(posts) != 2 {
			t.Fatalf("expected 2 posts, got %d", len(posts))
		}
	})
	mt.Run("find error", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "fail"}})
		repo := NewMongoRepo(mt.Coll, nilLogger)
		posts := repo.GetPosts()
		if posts != nil {
			t.Fatalf("expected nil slice on error, got %+v", posts)
		}
	})
}

func TestGetPostsByCategory(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("by category success", func(mt *mtest.T) {
		batch := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{{Key: "id", Value: "cat1"}, {Key: "category", Value: "tech"}},
		)
		endBatch := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(batch, endBatch)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		posts := repo.GetPostsByCategory("tech")
		if len(posts) != 1 || posts[0].Category != "tech" {
			t.Errorf("unexpected posts: %+v", posts)
		}
	})
	mt.Run("find error", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "fail"}})
		repo := NewMongoRepo(mt.Coll, nilLogger)
		posts := repo.GetPostsByCategory("tech")
		if posts != nil {
			t.Fatalf("expected nil slice on error, got %+v", posts)
		}
	})
}

func TestCreatePost_SuccessAndError(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("success insert text", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		repo := NewMongoRepo(mt.Coll, nilLogger)
		req := NewPostRequest{Category: "c", Type: "text", Title: "T", Text: "body"}
		p := repo.CreatePost(req, "user", "uid")
		if p == nil || p.Title != "T" {
			t.Fatalf("expected valid post, got %+v", p)
		}
	})
	mt.Run("success insert link", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		repo := NewMongoRepo(mt.Coll, nilLogger)
		req := NewPostRequest{Category: "c", Type: "link", Title: "T", URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}
		p := repo.CreatePost(req, "user", "uid")
		if p == nil || p.Title != "T" {
			t.Fatalf("expected valid post, got %+v", p)
		}
	})
	mt.Run("insert error", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "fail"}})
		repo := NewMongoRepo(mt.Coll, nilLogger)
		req := NewPostRequest{Category: "c", Type: "text", Title: "T", Text: "body"}
		p := repo.CreatePost(req, "user", "uid")
		if p != nil {
			t.Fatalf("expected nil on insert error, got %+v", p)
		}
	})
}

func TestAddComment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("success add comment", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p1"},
				{Key: "comments", Value: []Comment{}},
				{Key: "title", Value: "Post"},
				{Key: "author", Value: Author{Username: "bob", ID: "uid"}},
			},
		)
		endInitial := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, endInitial)
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		updated := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p1"},
				{Key: "comments", Value: []Comment{
					{
						ID:   "new-comment",
						Body: "nice post",
						Author: Author{
							Username: "bob",
							ID:       "uid",
						},
					},
				}},
				{Key: "title", Value: "Post"},
				{Key: "author", Value: Author{Username: "bob", ID: "uid"}},
			},
		)
		endUpdated := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, endUpdated)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.AddComment("p1", "bob", "uid", "nice post")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(post.Comments) != 1 {
			t.Errorf("expected 1 comment, got %d", len(post.Comments))
		}
		if post.Comments[0].ID == "" {
			t.Errorf("expected non-empty comment ID")
		}
	})
	mt.Run("post not found", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}})
		repo := NewMongoRepo(mt.Coll, nilLogger)
		_, err := repo.AddComment("p-no", "bob", "uid", "Comment")
		if !errors.Is(err, ErrPostNotFound) {
			t.Fatalf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestDeleteComment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("success delete comment", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p2"},
				{Key: "comments", Value: []Comment{
					{
						ID:     "c1",
						Body:   "goooooooood",
						Author: Author{Username: "eve", ID: "uid2"},
					},
				}},
				{Key: "title", Value: "Post2"},
				{Key: "author", Value: Author{Username: "eve", ID: "uid2"}},
			},
		)
		endInitial := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, endInitial)
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		updated := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p2"},
				{Key: "comments", Value: []Comment{}},
				{Key: "title", Value: "Post2"},
				{Key: "author", Value: Author{Username: "eve", ID: "uid2"}},
			},
		)
		endUpdated := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, endUpdated)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.DeleteComment("p2", "c1", "uid2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(post.Comments) != 0 {
			t.Errorf("expected no comments, got %+v", post.Comments)
		}
	})
	mt.Run("comment not found", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p2"},
				{Key: "comments", Value: []Comment{}},
				{Key: "title", Value: "Post2"},
				{Key: "author", Value: Author{Username: "eve", ID: "uid2"}},
			},
		)
		endInitial := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, endInitial)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		_, err := repo.DeleteComment("p2", "nonexistent", "uid2")
		if !errors.Is(err, ErrCommentNotFound) {
			t.Fatalf("expected ErrCommentNotFound, got %v", err)
		}
	})
	mt.Run("unauthorized delete", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p3"},
				{Key: "comments", Value: []Comment{
					{
						ID:     "c2",
						Body:   "not yours",
						Author: Author{Username: "mallory", ID: "uid3"}},
				}},
				{Key: "title", Value: "Post3"},
				{Key: "author", Value: Author{Username: "mallory", ID: "uid3"}},
			},
		)
		endInitial := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, endInitial)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		_, err := repo.DeleteComment("p3", "c2", "other")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

func TestVotePostBasic(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("upvote", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post123"},
				{Key: "title", Value: "Test"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 0},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "alice", ID: "user42"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 0},
			},
		)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, end)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		updated := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post123"},
				{Key: "title", Value: "Test"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "alice", ID: "user42"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{{"user42", 1}}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 100},
			},
		)
		updatedEnd := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, updatedEnd)

		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.VotePost("post123", "user42", 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Score != 1 || len(post.Votes) != 1 || post.UpvotePercentage != 100 {
			t.Errorf("unexpected post state: %+v", post)
		}
	})

	mt.Run("downvote", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post124"},
				{Key: "title", Value: "Test Downvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 0},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "bob", ID: "user43"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 0},
			},
		)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, end)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		updated := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post124"},
				{Key: "title", Value: "Test Downvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: -1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "bob", ID: "user43"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{{"user43", -1}}},
				{Key: "comments", Value: bson.A{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 0},
			},
		)
		updatedEnd := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, updatedEnd)

		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.VotePost("post124", "user43", -1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Score != -1 || len(post.Votes) != 1 || post.UpvotePercentage != 0 {
			t.Errorf("unexpected post state: %+v", post)
		}
	})

	mt.Run("unvote", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post125"},
				{Key: "title", Value: "Test Unvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "carol", ID: "user44"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{{"user44", 1}}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 100},
			},
		)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, end)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		updated := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post125"},
				{Key: "title", Value: "Test Unvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 0},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "carol", ID: "user44"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 100},
			},
		)
		updatedEnd := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, updatedEnd)

		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.VotePost("post125", "user44", 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Score != 0 || len(post.Votes) != 0 || post.UpvotePercentage != 100 {
			t.Errorf("unexpected post state: %+v", post)
		}
	})
}

func TestVotePostChange(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("change downvote to upvote", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post126"},
				{Key: "title", Value: "Test Change Vote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: -1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "alice", ID: "user42"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{
					{"user42", -1},
				}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 0},
			},
		)
		endInitial := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, endInitial)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		updated := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post126"},
				{Key: "title", Value: "Test Change Vote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "alice", ID: "user42"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{
					{"user42", 1},
				}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 100},
			},
		)
		updatedEnd := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, updatedEnd)

		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.VotePost("post126", "user42", 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Score != 1 || len(post.Votes) != 1 || post.Votes[0].Vote != 1 || post.UpvotePercentage != 100 {
			t.Errorf("unexpected post state: %+v", post)
		}
	})

	mt.Run("change upvote to downvote", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post127"},
				{Key: "title", Value: "Test Change Upvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "alice", ID: "user42"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{{"user42", 1}}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 100},
			},
		)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, end)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		updated := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post127"},
				{Key: "title", Value: "Test Change Upvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: -1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "alice", ID: "user42"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{{"user42", -1}}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 0},
			},
		)
		updatedEnd := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, updatedEnd)

		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.VotePost("post127", "user42", -1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Score != -1 || len(post.Votes) != 1 || post.Votes[0].Vote != -1 || post.UpvotePercentage != 0 {
			t.Errorf("unexpected post state: %+v", post)
		}
	})

	mt.Run("unvote downvote", func(mt *mtest.T) {
		initial := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post128"},
				{Key: "title", Value: "Test Unvote Downvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: -1},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "dave", ID: "user45"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{{"user45", -1}}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 0},
			},
		)
		endInitial := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(initial, endInitial)

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		updated := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "post128"},
				{Key: "title", Value: "Test Unvote Downvote"},
				{Key: "category", Value: "general"},
				{Key: "type", Value: "text"},
				{Key: "score", Value: 0},
				{Key: "views", Value: 1},
				{Key: "author", Value: Author{Username: "dave", ID: "user45"}},
				{Key: "text", Value: "Body"},
				{Key: "votes", Value: []Vote{}},
				{Key: "comments", Value: []Comment{}},
				{Key: "created", Value: time.Now().UTC()},
				{Key: "upvotePercentage", Value: 100},
			},
		)
		endUpdated := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", mt.DB.Name(), mt.Coll.Name()),
			mtest.NextBatch,
		)
		mt.AddMockResponses(updated, endUpdated)

		repo := NewMongoRepo(mt.Coll, nilLogger)
		post, err := repo.VotePost("post128", "user45", 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Score != 0 || len(post.Votes) != 0 || post.UpvotePercentage != 100 {
			t.Errorf("unexpected post state: %+v", post)
		}
	})
}

func TestDeletePost(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("successful deletion", func(mt *mtest.T) {
		findResp := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p4"},
				{Key: "author", Value: Author{Username: "dave", ID: "uid4"}},
			},
		)
		findEnd := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(findResp, findEnd)
		delResp := mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1})
		mt.AddMockResponses(delResp)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		ok, err := repo.DeletePost("p4", "uid4")
		if err != nil || ok != true {
			t.Fatalf("expected deletion success, got ok=%v err=%v", ok, err)
		}
	})
	mt.Run("post not found", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}})
		repo := NewMongoRepo(mt.Coll, nilLogger)
		_, err := repo.DeletePost("p-no", "uid")
		if !errors.Is(err, ErrPostNotFound) {
			t.Fatalf("expected ErrPostNotFound, got %v", err)
		}
	})
	mt.Run("unauthorized deletion", func(mt *mtest.T) {
		findResp := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{
				{Key: "id", Value: "p5"},
				{Key: "author", Value: Author{Username: "dave", ID: "uid4"}},
			},
		)
		findEnd := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(findResp, findEnd)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		ok, err := repo.DeletePost("p5", "other")
		if !errors.Is(err, ErrUnauthorized) || ok {
			t.Fatalf("expected unauthorized, got ok=%v err=%v", ok, err)
		}
	})
}

func TestPostsByUser(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("user posts found", func(mt *mtest.T) {
		batch := mtest.CreateCursorResponse(
			1,
			"db.coll",
			mtest.FirstBatch,
			bson.D{{Key: "id", Value: "p6"}, {Key: "author", Value: Author{Username: "sam", ID: "uid6"}}},
		)
		endBatch := mtest.CreateCursorResponse(
			0,
			"db.coll",
			mtest.NextBatch,
		)
		mt.AddMockResponses(batch, endBatch)
		repo := NewMongoRepo(mt.Coll, nilLogger)
		posts := repo.PostsByUser("sam")
		if len(posts) != 1 || posts[0].Author.Username != "sam" {
			t.Errorf("unexpected posts: %+v", posts)
		}
	})
	mt.Run("find error", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "fail"}})
		repo := NewMongoRepo(mt.Coll, nilLogger)
		posts := repo.PostsByUser("sam")
		if len(posts) != 0 {
			t.Errorf("expected empty slice on error, got %+v", posts)
		}
	})
}
