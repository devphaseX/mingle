package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func NewMockStore() Storage {
	return Storage{
		Users: NewMockUserStore(),
	}
}

type MockUserStore struct {
	users       map[int64]*User
	invitations map[string]int64 // token -> userID
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users:       make(map[int64]*User),
		invitations: make(map[string]int64),
	}
}

func (m *MockUserStore) Create(ctx context.Context, user *User, tx *sql.Tx) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.ID == 0 {
		return errors.New("user ID cannot be zero")
	}
	if _, exists := m.users[user.ID]; exists {
		return errors.New("user already exists")
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserStore) GetById(ctx context.Context, id int64) (*User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockUserStore) Delete(ctx context.Context, id int64) error {
	if _, exists := m.users[id]; !exists {
		return errors.New("user not found")
	}
	delete(m.users, id)
	return nil
}

func (m *MockUserStore) Activate(ctx context.Context, email string) error {
	for _, user := range m.users {
		if user.Email == email {
			user.IsActive = true
			return nil
		}
	}
	return errors.New("user not found")
}

func (m *MockUserStore) CreateAndInvite(ctx context.Context, user *User, invitationExp time.Duration, token string) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if err := m.Create(ctx, user, nil); err != nil {
		return err
	}
	expirationTime := time.Now().Add(invitationExp)
	if err := m.createUserInvitation(ctx, nil, token, expirationTime, user.ID); err != nil {
		return err
	}
	return nil
}

func (m *MockUserStore) createUserInvitation(ctx context.Context, tx *sql.Tx, token string, exp time.Time, userId int64) error {
	if _, exists := m.invitations[token]; exists {
		return errors.New("invitation token already exists")
	}
	m.invitations[token] = userId
	return nil
}
