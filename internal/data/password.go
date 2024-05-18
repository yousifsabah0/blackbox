package data

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	cost = 12
)

type Password struct {
	text *string
	hash []byte
}

func (p *Password) Hash(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), cost)
	if err != nil {
		return err
	}

	p.text = &text
	p.hash = hash

	return nil
}

func (p *Password) Matches(text string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(p.hash, []byte(text)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, err
		}

		return false, err
	}

	return true, nil
}
