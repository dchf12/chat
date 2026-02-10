package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dchf12/chat/domain"
	"github.com/go-webauthn/webauthn/webauthn"
)

func TestSessionStore_SaveAndGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewSessionStore()

	session := webauthn.SessionData{
		Challenge: "test-challenge",
		UserID:    []byte("u1"),
	}

	if err := store.Save(ctx, "s1", session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := store.Get(ctx, "s1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Challenge != "test-challenge" {
		t.Errorf("want challenge test-challenge, got %s", got.Challenge)
	}
}

func TestSessionStore_Get_OneTimeUse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewSessionStore()

	session := webauthn.SessionData{Challenge: "once"}
	if err := store.Save(ctx, "s1", session); err != nil {
		t.Fatal(err)
	}

	// first Get succeeds
	if _, err := store.Get(ctx, "s1"); err != nil {
		t.Fatalf("first Get failed: %v", err)
	}

	// second Get should fail (one-time use)
	if _, err := store.Get(ctx, "s1"); err == nil {
		t.Fatal("expected error on second Get (one-time use)")
	}
}

func TestSessionStore_Get_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewSessionStore()

	_, err := store.Get(ctx, "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestSessionStore_Get_Expired(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewSessionStore()

	session := webauthn.SessionData{Challenge: "expired"}
	if err := store.Save(ctx, "s1", session); err != nil {
		t.Fatal(err)
	}

	// manually expire the entry
	store.mu.Lock()
	entry := store.sessions["s1"]
	entry.expiresAt = time.Now().Add(-1 * time.Second)
	store.sessions["s1"] = entry
	store.mu.Unlock()

	_, err := store.Get(ctx, "s1")
	if err == nil {
		t.Fatal("expected expired session error")
	}
}

func TestSessionStore_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewSessionStore()

	session := webauthn.SessionData{Challenge: "del"}
	if err := store.Save(ctx, "s1", session); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(ctx, "s1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get(ctx, "s1")
	if err == nil {
		t.Fatal("expected not found after Delete")
	}
}

// interface compliance check
var _ domain.SessionRepository = (*SessionStore)(nil)
