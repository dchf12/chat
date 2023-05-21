package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/stretchr/objx"
	"golang.org/x/oauth2"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Cookieがない、または空の場合はログイン画面にリダイレクト
	if cookie, err := r.Cookie("auth"); err == http.ErrNoCookie || cookie.Value == "" {
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
	case "callback":
		if provider != "google" {
			http.Error(w, fmt.Sprintf("Unsupported provider: %s", provider), http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		ctx := context.Background()
		token, err := googleConf.Exchange(ctx, code)
		if err != nil {
			http.Error(w, fmt.Sprintf("Code exchange failed: %s", err.Error()), http.StatusBadRequest)
			return
		}
		client := googleConf.Client(ctx, token)
		oauth2Service, err := oauth2api.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create a new oauth2 service: %s", err.Error()), http.StatusBadRequest)
			return
		}
		userInfo, err := oauth2Service.Userinfo.Get().Do()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get user info: %s", err.Error()), http.StatusBadRequest)
			return
		}
		m := md5.New()
		io.WriteString(m, strings.ToLower(userInfo.Email))
		userID := fmt.Sprintf("%x", m.Sum(nil))
		authCookieValue := objx.New(map[string]interface{}{
			"userid":     userID,
			"name":       userInfo.Name,
			"avatar_url": userInfo.Picture,
			"email":      userInfo.Email,
		}).MustBase64()
		// set cookie
		http.SetCookie(w, &http.Cookie{
			Name:  "auth",
			Value: authCookieValue,
			Path:  "/",
		})
		http.Redirect(w, r, "/chat", http.StatusTemporaryRedirect)
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
