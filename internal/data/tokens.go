package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/yousifsabah0/blackbox/internal/validator"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Text   string    `json:"token"`
	Hash   []byte    `json:"-"`
	UserID int64     `json:"-"`
	Expiry time.Time `json:"expiry"`
	Scope  string    `json:"-"`
}

type TokenModel struct {
	DB *sql.DB
}

func ValidTokenText(v *validator.Validator, text string) {
	v.Check(text != "", "token", "token must be provided")
	v.Check(len(text) == 26, "token", "must be 26 bytes long")
}

func (t TokenModel) Insert(token *Token) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					INSERT INTO tokens
					(hash, user_id, expiry, scope)
					VALUES
					($1, $2, $3, $4)
	`
	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

	_, err := t.DB.ExecContext(ctx, query, args...)
	return err
}

func (t TokenModel) New(userID int64, expiry time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, expiry, scope)
	if err != nil {
		return nil, err
	}

	if err := t.Insert(token); err != nil {
		return nil, err
	}

	return token, nil
}

func (t TokenModel) DeleteAllForUser(scope string, userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `DELETE FROM tokens WHERE scope = $1 AND user_id = $2`
	_, err := t.DB.ExecContext(ctx, query, scope, userID)
	return err
}

func generateToken(userID int64, expiry time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(expiry),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Text = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Text))
	token.Hash = hash[:]

	return token, nil
}
