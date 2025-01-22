package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID              int64      `json:"id"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	Username        string     `json:"username"`
	Email           string     `json:"email,omitempty"`
	IsActive        bool       `json:"is_active"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	Password        password   `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	RoleID          int64      `json:"role_id"`
	Role            Role       `json:"role"`
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
		INSERT INTO users (first_name, last_name, username, password_hash, email, role_id) VALUES ($1, $2, $3, $4, $5,
			SELECT id FROM roles where name = $6
		)
		RETURNING id, created_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	role := user.Role.Name
	if role == "" {
		role = "user"
	}

	args := []any{user.FirstName, user.LastName, user.Username, user.Password.hash, user.Email, role}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pq.Error
		// SQL State "23505" means unique_violation
		if errors.As(err, &pgErr) {
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
	query := `SELECT users.id, first_name, last_name, username,
			 email, created_at,is_active,email_verified_at, roles.* FROM users
			 JOIN roles ON users.role_id = roles.id
			 where users.id = $1
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
		&user.IsActive,
		&user.EmailVerifiedAt,
		&user.Role.ID,
		&user.Role.Name,
		&user.Role.Level,
		&user.Role.Description,
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

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, first_name, last_name, username, email,is_active,email_verified_at,password_hash, created_at FROM users
				 where email ilike $1
		`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)

	defer cancel()

	var user User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
		&user.Password.hash,
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

	_, err := tx.ExecContext(ctx, query, pq.Array([]byte(token)), userId, exp)

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

func (s *UserStore) Activate(ctx context.Context, token string) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		user, err := s.getUserFromInvitation(ctx, tx, token)

		fmt.Println(user)
		if err != nil {
			return err
		}

		if user.EmailVerifiedAt != nil {
			return ErrUserAlreadyActivated
		}

		user.IsActive = true
		now := time.Now()
		user.EmailVerifiedAt = &now

		if err := s.update(ctx, user, tx); err != nil {
			return err
		}

		return s.deleteUserInvitation(ctx, tx, token)
	})
}

func (s *UserStore) getUserFromInvitation(ctx context.Context, tx *sql.Tx, token string) (*User, error) {

	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.is_active, u.email_verified_at FROM users u
			  INNER JOIN user_invitations ui ON ui.user_id = u.id
			  where ui.token = $1 and ui.expiry > $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)

	defer cancel()

	var (
		user = &User{}
	)
	hash := sha256.Sum256([]byte(token))
	hashToken := hex.EncodeToString(hash[:])
	err := tx.QueryRowContext(ctx, query, pq.Array([]byte(hashToken)), time.Now()).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.IsActive,
		&user.EmailVerifiedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (app *UserStore) update(ctx context.Context, user *User, tx *sql.Tx) error {
	query := `UPDATE users SET first_name = $1,
	last_name = $2, email = $3, is_active = $4,
	email_verified_at = $5 WHERE id = $6`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)

	defer cancel()

	args := []any{user.FirstName, user.LastName, user.Email, user.IsActive, user.EmailVerifiedAt, user.ID}

	res, err := tx.ExecContext(ctx, query, args...)

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

func (s *UserStore) deleteUserInvitation(ctx context.Context, tx *sql.Tx, token string) error {
	query := `DELETE From user_invitations WHERE token = $1`

	hash := sha256.Sum256([]byte(token))
	hashToken := hex.EncodeToString(hash[:])

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)

	defer cancel()

	_, err := tx.ExecContext(ctx, query, pq.Array([]byte(hashToken)))

	return err
}

func (s *UserStore) Delete(ctx context.Context, userId int64) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		if err := s.delete(ctx, tx, userId); err != nil {
			return err
		}
		if err := s.deleteUserInvitations(ctx, tx, userId); err != nil {
			return err
		}

		return nil
	})

}

func (s *UserStore) delete(ctx context.Context, tx *sql.Tx, userId int64) error {
	deleteUserQuery := `
			DELETE FROM users WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, deleteUserQuery, userId)

	if err != nil {
		return err
	}
	return nil
}

func (s *UserStore) deleteUserInvitations(ctx context.Context, tx *sql.Tx, userId int64) error {
	query := `DELETE FROM user_invitations WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, query, userId)
	if err != nil {
		return err
	}

	return nil
}
