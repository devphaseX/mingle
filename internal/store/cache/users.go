package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devphaseX/mingle.git/internal/store"
	"github.com/redis/go-redis/v9"
)

var (
	UserExpTime = time.Minute
)

type UserStore struct {
	rdb *redis.Client
}

func createUserCacheKey(userID int64) string {
	return fmt.Sprintf("user-%v", userID)
}

func (s *UserStore) Get(ctx context.Context, userID int64) (*store.User, error) {
	cacheKey := createUserCacheKey(userID)
	data, err := s.rdb.Get(ctx, cacheKey).Result()

	if err == redis.Nil {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	user := &store.User{}
	if err := json.Unmarshal([]byte(data), user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserStore) Set(ctx context.Context, user *store.User) error {
	cacheKey := createUserCacheKey(user.ID)

	json, err := json.Marshal(user)

	if err != nil {
		return err
	}

	return s.rdb.SetEx(ctx, cacheKey, json, UserExpTime).Err()
}
