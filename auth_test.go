package main

import (
	"strings"
	"testing"
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
