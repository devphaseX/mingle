package store

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrNotFound = errors.New("resource not foun")
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
