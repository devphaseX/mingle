package store

import (
	"context"
	"database/sql"
	"time"
)

type Comment struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	PostID    int64     `json:"post_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user"`
}

type CommentStore struct {
	db *sql.DB
}

func (c *CommentStore) GetByPostID(ctx context.Context, postId int64) ([]*Comment, error) {
	query := `SELECT c.id, c.post_id, c.user_id,
			c.content, c.created_at,users.first_name, users.last_name,
		    users.username, users.id FROM comments c
			JOIN users on users.id = c.user_id
			WHERE c.post_id = $1
			ORDER by c.created_at DESC
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	rows, err := c.db.QueryContext(ctx, query, postId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	comments := []*Comment{}

	for rows.Next() {
		var comment Comment

		user := &comment.User
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
			&user.FirstName,
			&user.LastName,
			&user.Username,
			&user.ID,
		)

		if err != nil {
			return nil, err
		}

		comments = append(comments, &comment)
	}

	return comments, nil
}
