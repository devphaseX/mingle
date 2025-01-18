package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID              int64     `json:"id"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Username        string    `json:"username"`
	Email           string    `json:"email,omitempty"`
	IsActive        bool      `json:"is_active"`
	EmailVerifiedAt time.Time `json:"email_verified_at,omitempty"`
	Password        password  `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plantextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plantextPassword), 12)

	if err != nil {
		return err
	}

	p.plaintext = &plantextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))

	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

type UserStore struct {
	db *sql.DB
}

func (s *UserStore) Create(ctx context.Context, user *User, tx *sql.Tx) error {
	query := `
		INSERT INTO users (username, password, email) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	args := []any{user.Username, user.Password.hash, user.Email}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pq.Error
		// SQL State "23505" means unique_violation
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.Constraint == "users_email_key" {
				return ErrDuplicateEmail
			} else if pgErr.Constraint == "users_username_key" {
				return ErrDuplicateUsername
			}
		}
		return err
	}
	return nil
}

func (s *UserStore) GetById(ctx context.Context, userId int64) (*User, error) {

	query := `SELECT id, first_name, last_name, username, email, created_at FROM users
			 where id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)

	defer cancel()

	var user User
	err := s.db.QueryRowContext(ctx, query, userId).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (s *UserStore) createUserInvitation(ctx context.Context, tx *sql.Tx, token string, exp time.Time, userId int64) error {
	query := `INSERT INTO user_invitations(token, user_id, expiry)
			 VALUES ($1, $2, $3)
	`
	_, err := s.db.ExecContext(ctx, query, pq.Array([]byte(token)), userId, exp)

	if err != nil {
		return err
	}
	return nil
}

func (s *UserStore) CreateAndInvite(ctx context.Context, user *User, invitationExp time.Duration, token string) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		if err := s.Create(ctx, user, tx); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
		defer cancel()
		err := s.createUserInvitation(ctx, tx, token, time.Now().Add(invitationExp), user.ID)

		if err != nil {
			return err
		}

		return nil
	})

}
