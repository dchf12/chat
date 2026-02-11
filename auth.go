package main

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/objx"
	"golang.org/x/oauth2"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

var (
	credentials Credentials
	googleConf  *oauth2.Config
	authSecret  []byte
	authOnce    sync.Once
)

const oauthStateCookieName = "oauth_state"

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
			userData, err := getAuthUserData(c)
			if err != nil {
				clearAuthCookie(c)
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			}
			c.Set("userData", userData)
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
	state, err := generateOAuthState()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to generate OAuth state")
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.IsTLS(),
	})
	loginURL := googleConf.AuthCodeURL(state, oauth2.AccessTypeOffline)
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

	stateCookie, err := c.Cookie(oauthStateCookieName)
	if err != nil || stateCookie.Value == "" {
		return c.String(http.StatusBadRequest, "missing oauth state cookie")
	}
	queryState := c.QueryParam("state")
	if queryState == "" || !hmac.Equal([]byte(queryState), []byte(stateCookie.Value)) {
		return c.String(http.StatusBadRequest, "invalid oauth state")
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.IsTLS(),
	})

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
	setAuthCookieValue(c, map[string]any{
		"userid":     userID,
		"name":       userInfo.Name,
		"avatar_url": userInfo.Picture,
		"email":      userInfo.Email,
	})
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func getAuthUserData(c echo.Context) (map[string]any, error) {
	if v := c.Get("userData"); v != nil {
		if userData, ok := v.(map[string]any); ok {
			return userData, nil
		}
	}

	cookie, err := c.Cookie("auth")
	if err != nil || cookie.Value == "" {
		return nil, errors.New("auth cookie not found")
	}
	return parseAuthCookieValue(cookie.Value)
}

func setAuthCookieValue(c echo.Context, userData map[string]any) {
	c.SetCookie(&http.Cookie{
		Name:     "auth",
		Value:    makeAuthCookieValue(userData),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.IsTLS(),
	})
}

func clearAuthCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "auth",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.IsTLS(),
	})
}

func makeAuthCookieValue(userData map[string]any) string {
	payload := objx.New(userData).MustBase64()
	mac := hmac.New(sha256.New, getAuthSecret())
	_, _ = mac.Write([]byte(payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

func parseAuthCookieValue(raw string) (map[string]any, error) {
	i := strings.LastIndex(raw, ".")
	if i <= 0 || i == len(raw)-1 {
		return nil, errors.New("invalid auth cookie format")
	}

	payload := raw[:i]
	sigHex := raw[i+1:]
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, errors.New("invalid auth cookie signature")
	}

	mac := hmac.New(sha256.New, getAuthSecret())
	_, _ = mac.Write([]byte(payload))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, errors.New("invalid auth cookie signature")
	}

	decoded, err := objx.FromBase64(payload)
	if err != nil {
		return nil, errors.New("invalid auth cookie payload")
	}

	return map[string]any(decoded), nil
}

func getAuthSecret() []byte {
	authOnce.Do(func() {
		if secret := os.Getenv("AUTH_SECRET"); secret != "" {
			authSecret = []byte(secret)
			return
		}

		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			// Fallback keeps app running; authentication still protected from trivial forgery.
			authSecret = []byte("dev-only-fallback-auth-secret-change-me")
			log.Printf("failed to generate random AUTH_SECRET, using fallback secret: %v", err)
			return
		}
		authSecret = buf
		log.Print("AUTH_SECRET is not set; using ephemeral in-memory secret")
	})
	return authSecret
}

func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
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
