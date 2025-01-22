package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound               = errors.New("resource not found")
	ErrConflict               = errors.New("resource already exist")
	ErrUserAlreadyActivated   = errors.New("user already activated")
	ErrSessionCannotBeExtends = errors.New("session cannot be extended: RememberMe is not enabled")
	ErrDuplicateEmail         = UserFriendlyError{UserMessage: "email already taken", InternalErr: ErrConflict}
	ErrDuplicateUsername      = UserFriendlyError{UserMessage: "username already taken", InternalErr: ErrConflict}
	QueryTimeoutDuration      = time.Second * 5
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
		GetByEmail(context.Context, string) (*User, error)
		Delete(context.Context, int64) error
		Activate(context.Context, string) error
		CreateAndInvite(ctx context.Context, user *User, invitationExp time.Duration, token string) error
		createUserInvitation(ctx context.Context, tx *sql.Tx, token string, exp time.Time, userId int64) error
	}

	Sessions interface {
		CreateSession(ctx context.Context, userID int64, userAgent, ip string, expiry time.Duration, rememberMe bool) (*Session, error)
		ValidateSession(ctx context.Context, sessionID string, version int) (*Session, *User, bool, error)
		InvalidateSession(ctx context.Context, sessionID string) error
		UpdateLastUsed(ctx context.Context, sessionID string) error
		GetSessionsByUserID(ctx context.Context, userID string, isAdmin bool, paginateQuery PaginateQueryFilter) ([]Session, Metadata, error)
		GetSessionByID(ctx context.Context, sessionID string) (*Session, *User, error)
		ExtendSessionAndGenerateRefreshToken(ctx context.Context, session *Session, tokenMaker TokenMaker, rememberPeriod time.Duration) (string, error)
	}

	Comments interface {
		GetByPostID(context.Context, int64) ([]*Comment, error)
	}

	Followers interface {
		FollowUser(ctx context.Context, follower *Follower) error
		UnFollowUser(ctx context.Context, followedUserID int64, userID int64) error
	}

	Roles interface {
		GetByName(context.Context, string) (*Role, error)
	}
}

func NewPostgressStorage(db *sql.DB) Storage {
	return Storage{
		Users:     &UserStore{db},
		Posts:     &PostStore{db},
		Comments:  &CommentStore{db},
		Followers: &FollowerStore{db},
		Sessions:  &SessionStore{db},
		Roles:     &RoleStore{db},
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
