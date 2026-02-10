package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dchf12/chat/domain"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/objx"
)

// PasskeyHandler は WebAuthn パスキー認証のハンドラー。
type PasskeyHandler struct {
	webAuthn    *webauthn.WebAuthn
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
}

// NewPasskeyHandler は PasskeyHandler を生成する。
func NewPasskeyHandler(wa *webauthn.WebAuthn, ur domain.UserRepository, sr domain.SessionRepository) *PasskeyHandler {
	return &PasskeyHandler{
		webAuthn:    wa,
		userRepo:    ur,
		sessionRepo: sr,
	}
}

type registerRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// BeginRegistration はパスキー登録を開始する。
func (h *PasskeyHandler) BeginRegistration(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username is required"})
	}
	if len(req.Username) > 50 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username must be 50 characters or less"})
	}

	ctx := c.Request().Context()
	if _, err := h.userRepo.GetByName(ctx, req.Username); err == nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "username already exists"})
	}

	webAuthnID := make([]byte, 64)
	if _, err := rand.Read(webAuthnID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate WebAuthn ID"})
	}

	user := domain.User{
		ID:          generateUUID(),
		WebAuthnIDB: webAuthnID,
		Name:        req.Username,
		DisplayName: req.DisplayName,
	}

	options, session, err := h.webAuthn.BeginRegistration(
		user,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to begin registration: %v", err)})
	}

	if err := h.sessionRepo.Save(ctx, session.Challenge, *session); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save session"})
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save user"})
	}

	c.SetCookie(&http.Cookie{
		Name:     "webauthn_session",
		Value:    session.Challenge,
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	return c.JSON(http.StatusOK, options)
}

// FinishRegistration はパスキー登録を完了する。
func (h *PasskeyHandler) FinishRegistration(c echo.Context) error {
	cookie, err := c.Cookie("webauthn_session")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session cookie not found"})
	}

	ctx := c.Request().Context()
	session, err := h.sessionRepo.Get(ctx, cookie.Value)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session not found"})
	}

	user, err := h.userRepo.GetByWebAuthnID(ctx, session.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user not found"})
	}

	credential, err := h.webAuthn.FinishRegistration(user, session, c.Request())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("failed to finish registration: %v", err)})
	}

	if err := h.userRepo.AddCredential(ctx, user.ID, *credential); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save credential"})
	}

	deleteCookie(c, "webauthn_session")
	setAuthCookie(c, user)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// BeginLogin はパスキーログインを開始する。
func (h *PasskeyHandler) BeginLogin(c echo.Context) error {
	options, session, err := h.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to begin login: %v", err)})
	}

	ctx := c.Request().Context()
	if err := h.sessionRepo.Save(ctx, session.Challenge, *session); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save session"})
	}

	c.SetCookie(&http.Cookie{
		Name:     "webauthn_session",
		Value:    session.Challenge,
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	return c.JSON(http.StatusOK, options)
}

// FinishLogin はパスキーログインを完了する。
func (h *PasskeyHandler) FinishLogin(c echo.Context) error {
	cookie, err := c.Cookie("webauthn_session")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session cookie not found"})
	}

	ctx := c.Request().Context()
	session, err := h.sessionRepo.Get(ctx, cookie.Value)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session not found"})
	}

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		return h.userRepo.GetByWebAuthnID(ctx, userHandle)
	}

	user, credential, err := h.webAuthn.FinishPasskeyLogin(handler, session, c.Request())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("failed to finish login: %v", err)})
	}

	domainUser, ok := user.(domain.User)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "unexpected user type"})
	}

	if err := h.userRepo.UpdateCredential(ctx, domainUser.ID, *credential); err != nil {
		c.Logger().Warnf("failed to update credential sign count: %v", err)
	}

	deleteCookie(c, "webauthn_session")
	setAuthCookie(c, domainUser)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// setAuthCookie は認証Cookieを設定する（OAuth の handleCallback と同じ形式）。
func setAuthCookie(c echo.Context, user domain.User) {
	m := md5.New()
	_, _ = io.WriteString(m, strings.ToLower(user.Email))
	userID := fmt.Sprintf("%x", m.Sum(nil))

	if user.Email == "" {
		userID = hex.EncodeToString(user.WebAuthnIDB[:16])
	}

	avatarURL := user.AvatarURL
	if avatarURL == "" {
		avatarURL = fmt.Sprintf("https://www.gravatar.com/avatar/%s?d=mp", userID)
	}

	authCookieValue := objx.New(map[string]any{
		"userid":     userID,
		"name":       user.DisplayName,
		"avatar_url": avatarURL,
		"email":      user.Email,
	}).MustBase64()

	c.SetCookie(&http.Cookie{
		Name:  "auth",
		Value: authCookieValue,
		Path:  "/",
	})
}

func deleteCookie(c echo.Context, name string) {
	c.SetCookie(&http.Cookie{
		Name:   name,
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
}

// generateUUID は UUID v4 を生成する。
func generateUUID() string {
	uuid := make([]byte, 16)
	_, _ = rand.Read(uuid)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
