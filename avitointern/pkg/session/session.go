package session

import (
	"avitointern/pkg/user"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
)

type Session struct {
	ID     string
	UserID string
	User   *user.User
}

func NewSession(user *user.User) *Session {
	randID := make([]byte, 16)
	rand.Read(randID)

	return &Session{
		ID:     fmt.Sprintf("%x", randID),
		UserID: user.ID,
		User:   user,
	}
}

var (
	ErrNoAuth = errors.New("No session found")
)

type sessKey string

var SessionKey sessKey = "sessionKey"

func SessionFromContext(ctx context.Context) (*Session, error) {
	sess, ok := ctx.Value(SessionKey).(*Session)
	if !ok || sess == nil {
		return nil, ErrNoAuth
	}
	return sess, nil
}

func ContextWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, SessionKey, sess)
}
