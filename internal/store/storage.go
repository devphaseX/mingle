package store

import (
	"context"
	"database/sql"
)

type Storage struct {
	Posts interface {
		Create(context.Context, *Post) error
	}
	Users interface {
		Create(context.Context, *User) error
	}
}

func NewPostgressStorage(db *sql.DB) Storage {
	return Storage{
		Users: &UserStore{db},
		Posts: &PostStore{db},
	}
}
