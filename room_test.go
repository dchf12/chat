package main

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"
)

func TestIsAllowedWebSocketOrigin_SameHostHTTP(t *testing.T) {
	req := httptest.NewRequest("GET", "http://localhost:8080/room", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://localhost:8080")

	if !isAllowedWebSocketOrigin(req) {
		t.Fatal("expected origin to be allowed")
	}
}

func TestIsAllowedWebSocketOrigin_CrossHostDenied(t *testing.T) {
	req := httptest.NewRequest("GET", "http://localhost:8080/room", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://evil.example")

	if isAllowedWebSocketOrigin(req) {
		t.Fatal("expected cross-host origin to be denied")
	}
}

func TestIsAllowedWebSocketOrigin_HTTPSRequiredOnTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "https://localhost:8443/room", nil)
	req.Host = "localhost:8443"
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("Origin", "http://localhost:8443")

	if isAllowedWebSocketOrigin(req) {
		t.Fatal("expected non-https origin to be denied for TLS requests")
	}
}
