package domain

import (
	"github.com/go-webauthn/webauthn/webauthn"
)

// User はパスキー認証のユーザーモデル。
// webauthn.User インターフェースを実装する。
type User struct {
	ID          string
	WebAuthnIDB []byte
	Name        string
	DisplayName string
	Email       string
	AvatarURL   string
	Credentials []webauthn.Credential
}

func (u User) WebAuthnID() []byte {
	return u.WebAuthnIDB
}

func (u User) WebAuthnName() string {
	return u.Name
}

func (u User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// WithCredential は新しいクレデンシャルを追加した新しい User を返す（不変性パターン）。
func (u User) WithCredential(cred webauthn.Credential) User {
	newCreds := make([]webauthn.Credential, len(u.Credentials), len(u.Credentials)+1)
	copy(newCreds, u.Credentials)
	newCreds = append(newCreds, cred)

	return User{
		ID:          u.ID,
		WebAuthnIDB: u.WebAuthnIDB,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		AvatarURL:   u.AvatarURL,
		Credentials: newCreds,
	}
}
