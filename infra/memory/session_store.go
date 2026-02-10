package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

const sessionTTL = 60 * time.Second

type sessionEntry struct {
	data      webauthn.SessionData
	expiresAt time.Time
}

// SessionStore はインメモリの SessionRepository 実装。
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]sessionEntry
}

// NewSessionStore は空の SessionStore を生成する。
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]sessionEntry),
	}
}

// Save はセッションデータを保存する。TTL は 60 秒。
func (s *SessionStore) Save(_ context.Context, key string, session webauthn.SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[key] = sessionEntry{
		data:      session,
		expiresAt: time.Now().Add(sessionTTL),
	}
	return nil
}

// Get はセッションデータを取得し、同時に削除する（ワンタイム使用）。
func (s *SessionStore) Get(_ context.Context, key string) (webauthn.SessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.sessions[key]
	if !ok {
		return webauthn.SessionData{}, fmt.Errorf("session not found: %s", key)
	}

	delete(s.sessions, key)

	if time.Now().After(entry.expiresAt) {
		return webauthn.SessionData{}, fmt.Errorf("session expired: %s", key)
	}
	return entry.data, nil
}

// Delete はセッションデータを削除する。
func (s *SessionStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, key)
	return nil
}
