package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound             = errors.New("resource not found")
	ErrConflict             = errors.New("resource already exist")
	ErrUserAlreadyActivated = errors.New("user already activated")
	ErrDuplicateEmail       = UserFriendlyError{UserMessage: "email already taken", InternalErr: ErrConflict}
	ErrDuplicateUsername    = UserFriendlyError{UserMessage: "username already taken", InternalErr: ErrConflict}
	QueryTimeoutDuration    = time.Second * 5
)

type Storage struct {
	Posts interface {
		GetById(context.Context, int64) (*Post, error)
		DeleteByUser(ctx context.Context, postId int64, userId int64) error
		UpdateByUser(context.Context, *Post) error
		Create(context.Context, *Post) error
		GetUserFeed(context.Context, int64, PaginateQueryFilter) ([]*PostWithMetadata, Metadata, error)
	}
	Users interface {
		Create(context.Context, *User, *sql.Tx) error
		GetById(context.Context, int64) (*User, error)
		Activate(context.Context, string) error
		CreateAndInvite(ctx context.Context, user *User, invitationExp time.Duration, token string) error
		createUserInvitation(ctx context.Context, tx *sql.Tx, token string, exp time.Time, userId int64) error
	}

	Comments interface {
		GetByPostID(context.Context, int64) ([]*Comment, error)
	}

	Followers interface {
		FollowUser(ctx context.Context, follower *Follower) error
		UnFollowUser(ctx context.Context, followedUserID int64, userID int64) error
	}
}

func NewPostgressStorage(db *sql.DB) Storage {
	return Storage{
		Users:     &UserStore{db},
		Posts:     &PostStore{db},
		Comments:  &CommentStore{db},
		Followers: &FollowerStore{db},
	}
}

type UserFriendlyError struct {
	UserMessage string
	InternalErr error
}

func (e UserFriendlyError) Error() string {
	return e.UserMessage
}

func (e UserFriendlyError) Unwrap() error {
	return e.InternalErr
}

func withTx(db *sql.DB, ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
