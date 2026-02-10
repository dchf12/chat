package domain

import (
	"context"

	"github.com/go-webauthn/webauthn/webauthn"
)

// UserRepository はユーザーの永続化を抽象化する。
type UserRepository interface {
	Create(ctx context.Context, user User) error
	GetByID(ctx context.Context, id string) (User, error)
	GetByWebAuthnID(ctx context.Context, webAuthnID []byte) (User, error)
	GetByName(ctx context.Context, name string) (User, error)
	AddCredential(ctx context.Context, userID string, cred webauthn.Credential) error
	UpdateCredential(ctx context.Context, userID string, cred webauthn.Credential) error
}

// SessionRepository は WebAuthn セレモニー中の SessionData を一時保存する。
type SessionRepository interface {
	Save(ctx context.Context, key string, session webauthn.SessionData) error
	// Get はセッションデータを取得し、同時に削除する（ワンタイム使用）。
	Get(ctx context.Context, key string) (webauthn.SessionData, error)
	Delete(ctx context.Context, key string) error
}
