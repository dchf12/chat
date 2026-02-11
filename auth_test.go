package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

func TestAuthCookieValue_RoundTrip(t *testing.T) {
	raw := makeAuthCookieValue(map[string]any{
		"userid": "u1",
		"name":   "alice",
	})

	userData, err := parseAuthCookieValue(raw)
	if err != nil {
		t.Fatalf("parseAuthCookieValue failed: %v", err)
	}

	if userData["userid"] != "u1" {
		t.Fatalf("unexpected userid: %v", userData["userid"])
	}
	if userData["name"] != "alice" {
		t.Fatalf("unexpected name: %v", userData["name"])
	}
}

func TestAuthCookieValue_TamperedPayload(t *testing.T) {
	raw := makeAuthCookieValue(map[string]any{
		"userid": "u1",
		"name":   "alice",
	})

	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid cookie value format: %q", raw)
	}
	tampered := "x" + parts[0][1:] + "." + parts[1]

	if _, err := parseAuthCookieValue(tampered); err == nil {
		t.Fatal("expected signature verification error for tampered cookie")
	}
}

func TestHandleLogin_SetsStateCookie(t *testing.T) {
	prev := googleConf
	googleConf = &oauth2.Config{
		ClientID:    "cid",
		RedirectURL: "http://localhost:8080/auth/callback/google",
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://example.com/auth",
		},
	}
	defer func() { googleConf = prev }()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/login/google", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("action", "provider")
	c.SetParamValues("login", "google")

	if err := handleLogin(c); err != nil {
		t.Fatalf("handleLogin failed: %v", err)
	}
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	found := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == oauthStateCookieName {
			found = true
			if cookie.Value == "" {
				t.Fatal("oauth state cookie is empty")
			}
		}
	}
	if !found {
		t.Fatal("oauth state cookie was not set")
	}
}

func TestHandleCallback_InvalidState(t *testing.T) {
	prev := googleConf
	googleConf = &oauth2.Config{
		ClientID: "cid",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}
	defer func() { googleConf = prev }()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/callback/google?state=bad&code=x", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: "good"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("action", "provider")
	c.SetParamValues("callback", "google")

	if err := handleCallback(c); err != nil {
		t.Fatalf("handleCallback failed: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
