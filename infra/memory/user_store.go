package memory

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/dchf12/chat/domain"
	"github.com/go-webauthn/webauthn/webauthn"
)

// UserStore はインメモリの UserRepository 実装。
type UserStore struct {
	mu    sync.RWMutex
	users map[string]domain.User
}

// NewUserStore は空の UserStore を生成する。
func NewUserStore() *UserStore {
	return &UserStore{
		users: make(map[string]domain.User),
	}
}

// Create はユーザーを保存する。名前の重複はエラーを返す。
func (s *UserStore) Create(_ context.Context, user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.Name == user.Name {
			return fmt.Errorf("user name %q already exists", user.Name)
		}
	}
	s.users[user.ID] = user
	return nil
}

// GetByID はIDでユーザーを取得する。
func (s *UserStore) GetByID(_ context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return domain.User{}, fmt.Errorf("user not found: %s", id)
	}
	return user, nil
}

// GetByWebAuthnID は WebAuthn ID でユーザーを検索する。
func (s *UserStore) GetByWebAuthnID(_ context.Context, webAuthnID []byte) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if bytes.Equal(u.WebAuthnIDB, webAuthnID) {
			return u, nil
		}
	}
	return domain.User{}, fmt.Errorf("user not found by webauthn id")
}

// GetByName は名前でユーザーを検索する。
func (s *UserStore) GetByName(_ context.Context, name string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Name == name {
			return u, nil
		}
	}
	return domain.User{}, fmt.Errorf("user not found: %s", name)
}

// AddCredential はユーザーにクレデンシャルを追加する（不変性パターン）。
func (s *UserStore) AddCredential(_ context.Context, userID string, cred webauthn.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return fmt.Errorf("user not found: %s", userID)
	}
	s.users[userID] = user.WithCredential(cred)
	return nil
}

// UpdateCredential はクレデンシャルの SignCount を更新する。
func (s *UserStore) UpdateCredential(_ context.Context, userID string, cred webauthn.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return fmt.Errorf("user not found: %s", userID)
	}

	for i, c := range user.Credentials {
		if bytes.Equal(c.ID, cred.ID) {
			user.Credentials[i].Authenticator.SignCount = cred.Authenticator.SignCount
			s.users[userID] = user
			return nil
		}
	}
	return fmt.Errorf("credential not found: %s", cred.ID)
}
