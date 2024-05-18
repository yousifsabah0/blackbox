package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/yousifsabah0/blackbox/internal/validator"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	Anonymous         = &User{}
)

const (
	duplicateKeyError = `pq: duplicate key value violates unique constraint 'users_email_key'`
)

type User struct {
	ID int64 `json:"id"`

	Name  string `json:"name"`
	Email string `json:"email"`

	Password  Password `json:"-"`
	Activated bool     `json:"activated"`

	Version int32 `json:"-"`

	CreatedAt time.Time `json:"created_at"`
}

func (u *User) IsAnonymous() bool {
	return u == Anonymous
}

type UserModel struct {
	DB *sql.DB
}

func (u *UserModel) Insert(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					INSERT INTO users
					(name, email, password_hash, activated)
					VALUES 
					($1, $2, $3, $4)
					RETURNING id, version, created_at
	`
	args := []any{user.Name, user.Email, user.Password.hash, user.Activated}

	if err := u.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.Version, &user.CreatedAt); err != nil {
		if err.Error() == duplicateKeyError {
			return ErrDuplicateEmail
		}

		return err
	}

	return nil
}

func (u *UserModel) GetByEmail(email string) (*User, error) {
	var user User

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					SELECT 
					id, name, email, password_hash, activated, version, created_at
					FROM users
					WHERE
					email = $1
	`

	err := u.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (u *UserModel) Update(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					UPDATE users
					SET
					name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
					WHERE
					id = $5 AND version = $6
					RETURNING version
	`
	args := []any{user.Name, user.Email, user.Password.hash, user.Activated, user.ID, user.Version}

	err := u.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == duplicateKeyError:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (u *UserModel) GetForToken(scope, text string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var user User
	hash := sha256.Sum256([]byte(text))

	query := `
					SELECT
					users.id, users.name, users.email, users.password_hash, users.activated, users.created_at, users.version
					FROM users
					INNER JOIN tokens
					ON users.id = tokens.user_id
					WHERE
					tokens.hash = $1 AND
					tokens.scope = $2 AND
					tokens.expiry > $3
	`
	args := []any{hash[:], scope, time.Now()}
	if err := u.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.Name, &user.Email, &user.Password.hash, &user.Activated, &user.CreatedAt, &user.Version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}

		return nil, err
	}

	return &user, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordText(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.text != nil {
		ValidatePasswordText(v, *user.Password.text)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}
