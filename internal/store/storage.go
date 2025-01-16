package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("resource not foun")
	QueryTimeoutDuration = time.Second * 5
)

type Storage struct {
	Posts interface {
		GetById(context.Context, int64) (*Post, error)
		DeleteByUser(ctx context.Context, postId int64, userId int64) error
		UpdateByUser(context.Context, *Post) error
		Create(context.Context, *Post) error
	}
	Users interface {
		Create(context.Context, *User) error
		GetById(context.Context, int64) (*User, error)
	}

	Comments interface {
		GetByPostID(context.Context, int64) ([]*Comment, error)
	}
}

func NewPostgressStorage(db *sql.DB) Storage {
	return Storage{
		Users:    &UserStore{db},
		Posts:    &PostStore{db},
		Comments: &CommentStore{db},
	}
}
