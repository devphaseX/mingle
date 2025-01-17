package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type Follower struct {
	UserID     int64     `json:"user_id"`
	FollowerID int64     `json:"follower_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type FollowerStore struct {
	db *sql.DB
}

func (s *FollowerStore) FollowUser(ctx context.Context, follower *Follower) error {
	query := `INSERT INTO followers(user_id, follower_id) VALUES ($1, $2) RETURNING created_at`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, follower.UserID, follower.FollowerID).
		Scan(&follower.CreatedAt)

	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			// SQL State "23505" means unique_violation
			if pgErr.Code == "23505" {
				return errors.Join(
					fmt.Errorf("user %d is already followed by user %d", follower.UserID, follower.FollowerID),
					ErrUserAlreadyFollowed,
				)
			}
		}
		return fmt.Errorf("failed to follow user: %w", err)
	}
	return nil
}

func (s *FollowerStore) UnFollowUser(ctx context.Context, followedUserID int64, userId int64) error {
	stmt := `DELETE FROM followers WHERE user_id = $1 and follower_id = $2`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	res, err := s.db.ExecContext(ctx, stmt, followedUserID, userId)

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
