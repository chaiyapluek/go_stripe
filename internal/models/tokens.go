package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"
)

const (
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext string    `json:"token"`
	UserID    int64     `json:"-"`
	Hash      []byte    `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// generate token that last for ttl and return its
func GenerateToken(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: int64(userID),
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]
	return token, nil
}

func (m *DBModel) InsertToken(t *Token, u User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
	delete from tokens
	where user_id=?`
	_, err := m.DB.ExecContext(ctx, stmt, u.ID)
	if err != nil {
		return err
	}

	stmt = `
	insert into tokens (user_id, name, email, token_hash, created_at, updated_at, expiry)
	value (?,?,?,?,?,?,?)`

	_, err = m.DB.ExecContext(ctx, stmt,
		u.ID,
		u.LastName,
		u.Email,
		t.Hash,
		time.Now(),
		time.Now(),
		t.Expiry,
	)

	if err != nil {
		return err
	}
	return nil
}

func (m *DBModel) GetUserForToken(token string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tokenHash := sha256.Sum256([]byte(token))

	var user User
	query := `
	select u.id, u.first_name, u.last_name, u.email
	from
		users u
		inner join tokens t on (u.id = t.user_id)
	where
		t.token_hash = ?
		and t.expiry > ?
	`

	err := m.DB.QueryRowContext(ctx, query, tokenHash[:], time.Now()).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
