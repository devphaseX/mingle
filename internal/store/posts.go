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
	Comments  []*Comment `json:"comments,omitempty"`
}

type PostWithMetadata struct {
	Post
	User struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
	}
	CommentCount int `json:"comments_count"`
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

func (s *PostStore) GetUserFeed(ctx context.Context, userID int64, paginateQuery PaginateQueryFilter) ([]*PostWithMetadata, Metadata, error) {
	query := fmt.Sprintf(`
			    SELECT count(p.id) OVER (), p.id, p.title, p.content,
				p.user_id, p.created_at,p.version,p.tags, count(c.id) as comments_count,
			 	users.first_name,users.last_name, users.username,
				users.id as current_user_id  FROM posts p
				INNER JOIN users ON p.user_id =  users.id
				LEFT JOIN followers f ON f.follower_id = users.id
				LEFT JOIN comments c ON p.id = c.post_id
                WHERE p.user_id = $1 or p.user_id = f.user_id
				GROUP BY p.id, users.id
				ORDER BY %s %s
				LIMIT $2 OFFSET $3
				;
	`, paginateQuery.SortColumn(), paginateQuery.SortDirection())

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)

	defer cancel()
	rows, err := s.db.QueryContext(
		ctx,
		query,
		userID,
		paginateQuery.Limit(),
		paginateQuery.Offset())

	if err != nil {
		return nil, Metadata{}, err
	}

	var posts = []*PostWithMetadata{}
	var totalRecords int
	for rows.Next() {
		var post PostWithMetadata

		var tagsJSON []byte

		rows.Scan(
			&totalRecords,
			&post.ID,
			&post.Title,
			&post.Context,
			&post.UserID,
			&post.CreatedAt,
			&post.Version,
			&tagsJSON,
			&post.CommentCount,
			&post.User.FirstName,
			&post.User.LastName,
			&post.User.Username,
			&post.User.ID,
		)

		if tagsJSON != nil {
			if err := json.Unmarshal(tagsJSON, &post.Tags); err != nil {
				return nil, Metadata{}, fmt.Errorf("failed to unmarshal tags: %v", err)
			}
		}

		posts = append(posts, &post)
	}

	metadata := calculateMetadata(totalRecords, paginateQuery.Page, paginateQuery.PageSize)
	return posts, metadata, nil
}
