package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Post struct {
	ID        int64     `json:"id"`
	Context   string    `json:"content"`
	Title     string    `json:"title"`
	UserID    int64     `json:"user_id"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PostStore struct {
	db *sql.DB
}

func (s *PostStore) Create(ctx context.Context, post *Post) error {
	query := `INSERT INTO posts(title, content, user_id, tags)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, updated_at`

	args := []any{post.Title, post.Context, post.UserID, pq.Array(post.Tags)}
	row := s.db.QueryRowContext(ctx, query, args...)

	return row.Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
}
