package memory

import (
	"context"
	"testing"

	"github.com/dchf12/chat/domain"
	"github.com/go-webauthn/webauthn/webauthn"
)

func testUser(id, name string) domain.User {
	return domain.User{
		ID:          id,
		WebAuthnIDB: []byte(id),
		Name:        name,
		DisplayName: name,
	}
}

func TestUserStore_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()
	user := testUser("u1", "alice")

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != "alice" {
		t.Errorf("want name alice, got %s", got.Name)
	}
}

func TestUserStore_Create_DuplicateName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	if err := store.Create(ctx, testUser("u1", "alice")); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, testUser("u2", "alice")); err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestUserStore_GetByWebAuthnID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	user := testUser("u1", "alice")
	if err := store.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByWebAuthnID(ctx, []byte("u1"))
	if err != nil {
		t.Fatalf("GetByWebAuthnID failed: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("want ID u1, got %s", got.ID)
	}
}

func TestUserStore_GetByWebAuthnID_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	_, err := store.GetByWebAuthnID(ctx, []byte("missing"))
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestUserStore_GetByName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	if err := store.Create(ctx, testUser("u1", "alice")); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetByName(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("want ID u1, got %s", got.ID)
	}
}

func TestUserStore_GetByName_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	_, err := store.GetByName(ctx, "nobody")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestUserStore_AddCredential(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	if err := store.Create(ctx, testUser("u1", "alice")); err != nil {
		t.Fatal(err)
	}

	cred := webauthn.Credential{
		ID:              []byte("cred1"),
		PublicKey:       []byte("pk1"),
		AttestationType: "none",
		Authenticator: webauthn.Authenticator{
			SignCount: 0,
		},
	}
	if err := store.AddCredential(ctx, "u1", cred); err != nil {
		t.Fatalf("AddCredential failed: %v", err)
	}

	got, _ := store.GetByID(ctx, "u1")
	if len(got.Credentials) != 1 {
		t.Fatalf("want 1 credential, got %d", len(got.Credentials))
	}
	if string(got.Credentials[0].ID) != "cred1" {
		t.Errorf("want cred ID cred1, got %s", got.Credentials[0].ID)
	}
}

func TestUserStore_AddCredential_UserNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	err := store.AddCredential(ctx, "missing", webauthn.Credential{})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestUserStore_UpdateCredential(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	if err := store.Create(ctx, testUser("u1", "alice")); err != nil {
		t.Fatal(err)
	}

	cred := webauthn.Credential{
		ID:              []byte("cred1"),
		PublicKey:       []byte("pk1"),
		AttestationType: "none",
		Authenticator: webauthn.Authenticator{
			SignCount: 0,
		},
	}
	if err := store.AddCredential(ctx, "u1", cred); err != nil {
		t.Fatal(err)
	}

	updated := webauthn.Credential{
		ID: []byte("cred1"),
		Authenticator: webauthn.Authenticator{
			SignCount: 5,
		},
	}
	if err := store.UpdateCredential(ctx, "u1", updated); err != nil {
		t.Fatalf("UpdateCredential failed: %v", err)
	}

	got, _ := store.GetByID(ctx, "u1")
	if got.Credentials[0].Authenticator.SignCount != 5 {
		t.Errorf("want SignCount 5, got %d", got.Credentials[0].Authenticator.SignCount)
	}
}

func TestUserStore_UpdateCredential_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewUserStore()

	if err := store.Create(ctx, testUser("u1", "alice")); err != nil {
		t.Fatal(err)
	}

	cred := webauthn.Credential{
		ID: []byte("missing"),
		Authenticator: webauthn.Authenticator{
			SignCount: 1,
		},
	}
	err := store.UpdateCredential(ctx, "u1", cred)
	if err == nil {
		t.Fatal("expected credential not found error")
	}
}

// interface compliance check
var _ domain.UserRepository = (*UserStore)(nil)
