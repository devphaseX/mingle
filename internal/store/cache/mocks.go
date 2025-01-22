package cache

import (
	"context"

	"github.com/devphaseX/mingle.git/internal/store"
)

func NewMockCache() Storage {
	return Storage{}
}

type MockUserStore struct {
}

func (s *MockUserStore) Get(ctx context.Context, userID int64) (*store.User, error) {
	return &store.User{}, nil
}

func (s *MockUserStore) Set(ctx context.Context, user *store.User) error {

	return nil
}
