package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("auth"); err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	} else if err != nil {
		panic(err)
	} else {
		h.next.ServeHTTP(w, r)
	}
}

func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}

// パスの形式 /auth/{action}/{provider}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	segs := strings.Split(r.URL.Path, "/")
	action := segs[2]
	provider := segs[3]

	creds := loadCredentials()
	googleConf := &oauth2.Config{
		ClientID:     creds.Web.ClientID,
		ClientSecret: creds.Web.ClientSecret,
		RedirectURL:  creds.Web.RedirectURL[0],
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  creds.Web.AuthURL,
			TokenURL: creds.Web.TokenURL,
		},
	}

	switch action {
	case "login":
		if provider != "google" {
			http.Error(w, fmt.Sprintf("Unsupported provider: %s", provider), http.StatusBadRequest)
			return
		}
		loginURL := googleConf.AuthCodeURL("state", oauth2.AccessTypeOffline)
		http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Auth action %s not supported", action)
	}
}

type Credentials struct {
	Web struct {
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		RedirectURL  []string `json:"redirect_uris"`
		AuthURL      string   `json:"auth_uri"`
		TokenURL     string   `json:"token_uri"`
	} `json:"web"`
}

func loadCredentials() Credentials {
	credsFile, err := ioutil.ReadFile("secret.json")
	if err != nil {
		log.Fatalf("Error reading credentials file: %v", err)
	}
	var creds Credentials
	if err := json.Unmarshal(credsFile, &creds); err != nil {
		log.Fatalf("Error unmarshalling credentials: %v", err)
	}
	return creds
}
