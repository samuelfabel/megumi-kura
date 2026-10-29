package social

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type Post struct {
	ID           int64     `json:"id"`
	Kind         string    `json:"kind"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	ImageMediaID *int64    `json:"image_media_id,omitempty"`
	Published    bool      `json:"published"`
	CreatedBy    *int64    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Comments     []Comment `json:"comments,omitempty"`
}

type Comment struct {
	ID         int64     `json:"id"`
	PostID     int64     `json:"post_id"`
	AuthorName string    `json:"author_name"`
	Body       string    `json:"body"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type PostInput struct {
	Kind         string `json:"kind"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	ImageMediaID *int64 `json:"image_media_id"`
	Published    bool   `json:"published"`
}

type CommentInput struct {
	AuthorName string `json:"author_name"`
	Body       string `json:"body"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) ListPosts(ctx context.Context, publishedOnly bool) ([]Post, error) {
	q := `
		SELECT id, kind, title, body, image_media_id, published, created_by, created_at, updated_at
		FROM social_posts`
	if publishedOnly {
		q += ` WHERE published = TRUE`
	}
	q += ` ORDER BY created_at DESC`
	rows, err := r.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) GetPost(ctx context.Context, id int64, approvedCommentsOnly bool) (Post, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, kind, title, body, image_media_id, published, created_by, created_at, updated_at
		FROM social_posts WHERE id = $1`, id)
	p, err := scanPostRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, err
	}
	comments, err := r.listComments(ctx, id, approvedCommentsOnly)
	if err != nil {
		return Post{}, err
	}
	p.Comments = comments
	return p, nil
}

func (r *Repository) CreatePost(ctx context.Context, in PostInput, createdBy *int64) (Post, error) {
	in.Kind = strings.TrimSpace(in.Kind)
	in.Title = strings.TrimSpace(in.Title)
	if in.Kind != "article" && in.Kind != "testimonial" {
		return Post{}, errors.New("kind must be article or testimonial")
	}
	if in.Title == "" {
		return Post{}, errors.New("title is required")
	}
	var id int64
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO social_posts(kind, title, body, image_media_id, published, created_by)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		in.Kind, in.Title, in.Body, in.ImageMediaID, in.Published, createdBy,
	).Scan(&id)
	if err != nil {
		return Post{}, err
	}
	return r.GetPost(ctx, id, false)
}

func (r *Repository) UpdatePost(ctx context.Context, id int64, in PostInput) (Post, error) {
	in.Kind = strings.TrimSpace(in.Kind)
	in.Title = strings.TrimSpace(in.Title)
	if in.Kind != "article" && in.Kind != "testimonial" {
		return Post{}, errors.New("kind must be article or testimonial")
	}
	if in.Title == "" {
		return Post{}, errors.New("title is required")
	}
	res, err := r.DB.ExecContext(ctx, `
		UPDATE social_posts SET kind=$2, title=$3, body=$4, image_media_id=$5, published=$6, updated_at=NOW()
		WHERE id=$1`, id, in.Kind, in.Title, in.Body, in.ImageMediaID, in.Published)
	if err != nil {
		return Post{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Post{}, ErrNotFound
	}
	return r.GetPost(ctx, id, false)
}

func (r *Repository) DeletePost(ctx context.Context, id int64) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM social_posts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) AddComment(ctx context.Context, postID int64, in CommentInput) (Comment, error) {
	in.AuthorName = strings.TrimSpace(in.AuthorName)
	in.Body = strings.TrimSpace(in.Body)
	if in.AuthorName == "" || in.Body == "" {
		return Comment{}, errors.New("author_name and body are required")
	}
	var exists bool
	_ = r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM social_posts WHERE id=$1 AND published=TRUE)`, postID).Scan(&exists)
	if !exists {
		return Comment{}, ErrNotFound
	}
	var c Comment
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO social_comments(post_id, author_name, body, status)
		VALUES ($1,$2,$3,'pending')
		RETURNING id, post_id, author_name, body, status, created_at`,
		postID, in.AuthorName, in.Body,
	).Scan(&c.ID, &c.PostID, &c.AuthorName, &c.Body, &c.Status, &c.CreatedAt)
	return c, err
}

func (r *Repository) ModerateComment(ctx context.Context, commentID int64, status string, moderatorID *int64) (Comment, error) {
	if status != "approved" && status != "rejected" {
		return Comment{}, errors.New("status must be approved or rejected")
	}
	res, err := r.DB.ExecContext(ctx, `
		UPDATE social_comments SET status=$2, moderated_at=NOW(), moderated_by=$3 WHERE id=$1`,
		commentID, status, moderatorID)
	if err != nil {
		return Comment{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Comment{}, ErrNotFound
	}
	var c Comment
	err = r.DB.QueryRowContext(ctx, `
		SELECT id, post_id, author_name, body, status, created_at FROM social_comments WHERE id=$1`, commentID,
	).Scan(&c.ID, &c.PostID, &c.AuthorName, &c.Body, &c.Status, &c.CreatedAt)
	return c, err
}

func (r *Repository) ListPendingComments(ctx context.Context) ([]Comment, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, post_id, author_name, body, status, created_at
		FROM social_comments WHERE status = 'pending' ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.Body, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) listComments(ctx context.Context, postID int64, approvedOnly bool) ([]Comment, error) {
	q := `
		SELECT id, post_id, author_name, body, status, created_at
		FROM social_comments WHERE post_id = $1`
	if approvedOnly {
		q += ` AND status = 'approved'`
	}
	q += ` ORDER BY created_at ASC`
	rows, err := r.DB.QueryContext(ctx, q, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.Body, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanPost(rows *sql.Rows) (Post, error) {
	return scanPostRow(rows)
}

func scanPostRow(row scannable) (Post, error) {
	var p Post
	var img, by sql.NullInt64
	err := row.Scan(&p.ID, &p.Kind, &p.Title, &p.Body, &img, &p.Published, &by, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Post{}, err
	}
	if img.Valid {
		p.ImageMediaID = &img.Int64
	}
	if by.Valid {
		p.CreatedBy = &by.Int64
	}
	return p, nil
}
