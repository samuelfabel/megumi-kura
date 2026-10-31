package social

import (
	"context"
	"errors"
	"testing"

	"github.com/samuelfabel/megumi-kura/api/internal/testdb"
)

func TestCreatePostValidation(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()
	if _, err := r.CreatePost(ctx, PostInput{Kind: "news", Title: "x"}, nil); err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if _, err := r.CreatePost(ctx, PostInput{Kind: "article", Title: "  "}, nil); err == nil {
		t.Fatal("expected error for blank title")
	}
	p, err := r.CreatePost(ctx, PostInput{Kind: " testimonial ", Title: " Obrigado ", Body: "texto"}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.Kind != "testimonial" || p.Title != "Obrigado" || p.Published {
		t.Fatalf("unexpected post: %+v", p)
	}
}

func TestListPostsPublishedOnly(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()
	draft, err := r.CreatePost(ctx, PostInput{Kind: "article", Title: "Rascunho"}, nil)
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	pub, err := r.CreatePost(ctx, PostInput{Kind: "article", Title: "Publicado", Published: true}, nil)
	if err != nil {
		t.Fatalf("create published: %v", err)
	}

	posts, err := r.ListPosts(ctx, true)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(posts) != 1 || posts[0].ID != pub.ID {
		t.Fatalf("published list = %+v, want only post %d", posts, pub.ID)
	}

	all, err := r.ListPosts(ctx, false)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("all posts = %d, want 2 (draft %d included)", len(all), draft.ID)
	}
}

func TestCommentModerationFlow(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()

	draft, err := r.CreatePost(ctx, PostInput{Kind: "article", Title: "Rascunho"}, nil)
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if _, err := r.AddComment(ctx, draft.ID, CommentInput{AuthorName: "Ana", Body: "Oi"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("comment on unpublished post err = %v, want ErrNotFound", err)
	}

	post, err := r.CreatePost(ctx, PostInput{Kind: "article", Title: "Post", Published: true}, nil)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if _, err := r.AddComment(ctx, post.ID, CommentInput{AuthorName: " ", Body: "Oi"}); err == nil {
		t.Fatal("expected error for blank author")
	}

	c, err := r.AddComment(ctx, post.ID, CommentInput{AuthorName: "Ana", Body: "Que bom!"})
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if c.Status != "pending" {
		t.Fatalf("new comment status = %q, want pending", c.Status)
	}

	public, err := r.GetPost(ctx, post.ID, true)
	if err != nil {
		t.Fatalf("get post: %v", err)
	}
	if len(public.Comments) != 0 {
		t.Fatalf("pending comment visible publicly: %+v", public.Comments)
	}

	pending, err := r.ListPendingComments(ctx)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != c.ID {
		t.Fatalf("pending = %+v", pending)
	}

	if _, err := r.ModerateComment(ctx, c.ID, "maybe", nil); err == nil {
		t.Fatal("expected error for invalid status")
	}
	if _, err := r.ModerateComment(ctx, 999999, "approved", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("moderate missing comment err = %v, want ErrNotFound", err)
	}
	approved, err := r.ModerateComment(ctx, c.ID, "approved", nil)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != "approved" {
		t.Fatalf("status = %q, want approved", approved.Status)
	}

	public, err = r.GetPost(ctx, post.ID, true)
	if err != nil {
		t.Fatalf("get post: %v", err)
	}
	if len(public.Comments) != 1 {
		t.Fatalf("approved comments = %d, want 1", len(public.Comments))
	}
}

func TestDeletePost(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()
	p, err := r.CreatePost(ctx, PostInput{Kind: "article", Title: "Apagar"}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := r.DeletePost(ctx, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := r.DeletePost(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete err = %v, want ErrNotFound", err)
	}
	if _, err := r.GetPost(ctx, p.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get deleted err = %v, want ErrNotFound", err)
	}
}
