package main

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/dchf12/chat/domain"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v4"
)

// --- Mock Repositories ---

type mockUserRepo struct {
	users map[string]domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]domain.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, echo.NewHTTPError(http.StatusNotFound, "user not found")
}

func (m *mockUserRepo) GetByWebAuthnID(_ context.Context, webAuthnID []byte) (domain.User, error) {
	for _, u := range m.users {
		if string(u.WebAuthnIDB) == string(webAuthnID) {
			return u, nil
		}
	}
	return domain.User{}, echo.NewHTTPError(http.StatusNotFound, "user not found")
}

func (m *mockUserRepo) GetByName(_ context.Context, name string) (domain.User, error) {
	for _, u := range m.users {
		if u.Name == name {
			return u, nil
		}
	}
	return domain.User{}, echo.NewHTTPError(http.StatusNotFound, "user not found")
}

func (m *mockUserRepo) AddCredential(_ context.Context, userID string, cred webauthn.Credential) error {
	if u, ok := m.users[userID]; ok {
		u.Credentials = append(u.Credentials, cred)
		m.users[userID] = u
	}
	return nil
}

func (m *mockUserRepo) UpdateCredential(_ context.Context, userID string, cred webauthn.Credential) error {
	if u, ok := m.users[userID]; ok {
		for i, c := range u.Credentials {
			if string(c.ID) == string(cred.ID) {
				u.Credentials[i] = cred
				m.users[userID] = u
				return nil
			}
		}
	}
	return nil
}

type mockSessionRepo struct {
	sessions map[string]webauthn.SessionData
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[string]webauthn.SessionData)}
}

func (m *mockSessionRepo) Save(_ context.Context, key string, session webauthn.SessionData) error {
	m.sessions[key] = session
	return nil
}

func (m *mockSessionRepo) Get(_ context.Context, key string) (webauthn.SessionData, error) {
	s, ok := m.sessions[key]
	if !ok {
		return webauthn.SessionData{}, echo.NewHTTPError(http.StatusNotFound, "session not found")
	}
	delete(m.sessions, key)
	return s, nil
}

func (m *mockSessionRepo) Delete(_ context.Context, key string) error {
	delete(m.sessions, key)
	return nil
}

// --- Tests ---

func TestSetAuthCookie_WithEmail(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := domain.User{
		ID:          "test-id",
		WebAuthnIDB: make([]byte, 64),
		Name:        "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
		AvatarURL:   "https://example.com/avatar.png",
	}

	setAuthCookie(c, user)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected auth cookie to be set")
	}

	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth" {
			authCookie = cookie
			break
		}
	}
	if authCookie == nil {
		t.Fatal("auth cookie not found")
	}

	decoded, err := parseAuthCookieValue(authCookie.Value)
	if err != nil {
		t.Fatalf("failed to parse auth cookie: %v", err)
	}
	if decoded["name"] != "Test User" {
		t.Errorf("expected name 'Test User', got %v", decoded["name"])
	}
	if decoded["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %v", decoded["email"])
	}
	if decoded["avatar_url"] != "https://example.com/avatar.png" {
		t.Errorf("expected avatar_url 'https://example.com/avatar.png', got %v", decoded["avatar_url"])
	}
	// md5 of "test@example.com"
	if decoded["userid"] == "" {
		t.Error("expected userid to be set")
	}
}

func TestSetAuthCookie_WithoutEmail(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	webAuthnID := make([]byte, 64)
	for i := range webAuthnID {
		webAuthnID[i] = byte(i)
	}

	user := domain.User{
		ID:          "test-id",
		WebAuthnIDB: webAuthnID,
		Name:        "testuser",
		DisplayName: "Test User",
	}

	setAuthCookie(c, user)

	cookies := rec.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth" {
			authCookie = cookie
			break
		}
	}
	if authCookie == nil {
		t.Fatal("auth cookie not found")
	}

	decoded, err := parseAuthCookieValue(authCookie.Value)
	if err != nil {
		t.Fatalf("failed to parse auth cookie: %v", err)
	}

	expectedUserID := hex.EncodeToString(webAuthnID[:16])
	if decoded["userid"] != expectedUserID {
		t.Errorf("expected userid %q, got %v", expectedUserID, decoded["userid"])
	}

	expectedAvatar := "https://www.gravatar.com/avatar/" + expectedUserID + "?d=mp"
	if decoded["avatar_url"] != expectedAvatar {
		t.Errorf("expected avatar_url %q, got %v", expectedAvatar, decoded["avatar_url"])
	}
}

func TestBeginRegistration_EmptyUsername(t *testing.T) {
	e := echo.New()
	body := `{"username":"","display_name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/passkey/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	wa, _ := webauthn.New(&webauthn.Config{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
	})

	h := NewPasskeyHandler(wa, newMockUserRepo(), newMockSessionRepo())
	err := h.BeginRegistration(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestBeginRegistration_DuplicateUsername(t *testing.T) {
	e := echo.New()
	body := `{"username":"existinguser","display_name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/passkey/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	wa, _ := webauthn.New(&webauthn.Config{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
	})

	userRepo := newMockUserRepo()
	userRepo.users["existing-id"] = domain.User{
		ID:   "existing-id",
		Name: "existinguser",
	}

	h := NewPasskeyHandler(wa, userRepo, newMockSessionRepo())
	err := h.BeginRegistration(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(uuid) {
		t.Errorf("generated UUID %q does not match UUID v4 pattern", uuid)
	}

	uuid2 := generateUUID()
	if uuid == uuid2 {
		t.Error("two generated UUIDs should be different")
	}
}
