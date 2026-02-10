package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/objx"
	"golang.org/x/oauth2"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

var (
	credentials Credentials
	googleConf  *oauth2.Config
)

func init() {
	creds, err := loadCredentials()
	if err != nil {
		log.Printf("OAuth credentials not loaded (passkey-only mode): %v", err)
		return
	}
	credentials = creds
	googleConf = &oauth2.Config{
		ClientID:     credentials.Web.ClientID,
		ClientSecret: credentials.Web.ClientSecret,
		RedirectURL:  credentials.Web.RedirectURL[0],
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  credentials.Web.AuthURL,
			TokenURL: credentials.Web.TokenURL,
		},
	}
}

func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("auth")
			if err != nil || cookie.Value == "" {
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			}
			return next(c)
		}
	}
}

func loginHandler(c echo.Context) error {
	switch action := c.Param("action"); action {
	case "login":
		return handleLogin(c)
	case "callback":
		return handleCallback(c)
	default:
		return c.String(http.StatusNotFound, fmt.Sprintf("Auth action %s not supported", action))
	}
}

func handleLogin(c echo.Context) error {
	provider := c.Param("provider")
	if provider != "google" {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Unsupported provider: %s", provider))
	}
	if googleConf == nil {
		return c.String(http.StatusServiceUnavailable, "OAuth is not configured")
	}
	loginURL := googleConf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	return c.Redirect(http.StatusTemporaryRedirect, loginURL)
}

func handleCallback(c echo.Context) error {
	provider := c.Param("provider")
	if provider != "google" {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Unsupported provider: %s", provider))
	}
	if googleConf == nil {
		return c.String(http.StatusServiceUnavailable, "OAuth is not configured")
	}
	code := c.QueryParam("code")
	ctx := context.Background()
	token, err := googleConf.Exchange(ctx, code)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Code exchange failed: %s", err.Error()))
	}
	client := googleConf.Client(ctx, token)
	oauth2Service, err := oauth2api.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Failed to create a new oauth2 service: %s", err.Error()))
	}
	userInfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Failed to get user info: %s", err.Error()))
	}
	m := md5.New()
	_, _ = io.WriteString(m, strings.ToLower(userInfo.Email))
	userID := fmt.Sprintf("%x", m.Sum(nil))
	authCookieValue := objx.New(map[string]any{
		"userid":     userID,
		"name":       userInfo.Name,
		"avatar_url": userInfo.Picture,
		"email":      userInfo.Email,
	}).MustBase64()
	c.SetCookie(&http.Cookie{
		Name:  "auth",
		Value: authCookieValue,
		Path:  "/",
	})
	return c.Redirect(http.StatusTemporaryRedirect, "/")
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

func loadCredentials() (Credentials, error) {
	credsFile, err := os.ReadFile("secret.json")
	if err != nil {
		return Credentials{}, fmt.Errorf("error reading credentials file: %v", err)
	}
	var creds Credentials
	if err := json.Unmarshal(credsFile, &creds); err != nil {
		return Credentials{}, fmt.Errorf("error unmarshalling credentials: %v", err)
	}
	return creds, nil
}
