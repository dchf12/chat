package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestAuthAvatar(t *testing.T) {
	var authAvatar AuthAvatar
	client := new(client)
	url, err := authAvatar.AvatarURL(client)
	if err != ErrNoAvatarURL {
		t.Error("値が存在しない場合、AuthAvatar.AvatarURLは" +
			"ErrNoAvatarURLを返すべきです")
	}
	testURL := "http://url-to-avatar/"
	client.userData = map[string]interface{}{"avatar_url": testURL}
	url, err = authAvatar.AvatarURL(client)
	if err != nil {
		t.Error("値が存在する場合、AuthAvatar.AvatarURLは" +
			"エラーを返すべきではありません")
	} else {
		if url != testURL {
			t.Error("AuthAvatar.AvatarURLは正しいURLを返すべきです")
		}
	}
}

func TestGravatarAvatar(t *testing.T) {
	var gravatarAvatar GravatarAvatar
	client := new(client)
	client.userData = map[string]interface{}{
		"email": "email@example.com",
	}
	url, err := gravatarAvatar.AvatarURL(client)
	if err != nil {
		t.Error("GravatarAvatar.AvatarURLはエラーを返すべきではありません")
	}
	m := md5.New()
	io.WriteString(m, strings.ToLower(client.userData["email"].(string)))
	if url != "//www.gravatar.com/avatar/"+fmt.Sprintf("%x", m.Sum(nil)) {
		t.Errorf("GravatarAvatar.AvatarURLが%sという誤った値を返しました", url)
	}
}
