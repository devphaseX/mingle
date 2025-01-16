package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type Post struct {
	ID        int64      `json:"id"`
	Context   string     `json:"content"`
	Title     string     `json:"title"`
	UserID    int64      `json:"user_id"`
	Tags      []string   `json:"tags"`
	Version   int        `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Comments  []*Comment `json:"comments"`
}

type PostStore struct {
	db *sql.DB
}

func (s *PostStore) Create(ctx context.Context, post *Post) error {
	query := `INSERT INTO posts(title, content, user_id, tags)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, updated_at`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	args := []any{post.Title, post.Context, post.UserID, pq.Array(post.Tags)}
	row := s.db.QueryRowContext(ctx, query, args...)

	return row.Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
}

func (s *PostStore) GetById(ctx context.Context, id int64) (*Post, error) {
	query := `SELECT id, title, content, user_id, tags,version, created_at, updated_at FROM posts
			 WHERE id = $1
			 `

	var post Post
	var tagsJSON []byte

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID,
		&post.Title,
		&post.Context,
		&post.UserID,
		&tagsJSON,
		&post.Version,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		}
		return nil, err
	}

	if tagsJSON != nil {
		if err := json.Unmarshal(tagsJSON, &post.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %v", err)
		}

	}

	return &post, nil
}

func (s *PostStore) DeleteByUser(ctx context.Context, postId int64, userId int64) error {
	stmt := `DELETE FROM posts WHERE posts.id = $1 and posts.user_id = $2`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	res, err := s.db.ExecContext(ctx, stmt, postId, userId)

	if err != nil {
		return err
	}

	rowsCount, err := res.RowsAffected()

	if err != nil {
		return err
	}

	if rowsCount == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostStore) UpdateByUser(ctx context.Context, post *Post) error {
	query := `UPDATE posts SET title = $1, content = $2, version = version + 1
			WHERE id = $3 AND version = $4
			RETURNING version
		`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := s.db.QueryRowContext(ctx, query, post.Title, post.Context, post.ID, post.Version).Scan(&post.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNotFound
		default:
			return err
		}

	}
	return nil
}
