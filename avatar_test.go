package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthAvatar(t *testing.T) {
	var authAvatar AuthAvatar
	client := new(client)
	_, err := authAvatar.AvatarURL(client)
	if err != ErrNoAvatarURL {
		t.Error("値が存在しない場合、AuthAvatar.AvatarURLは" +
			"ErrNoAvatarURLを返すべきです")
	}
	testURL := "http://url-to-avatar/"
	client.userData = map[string]interface{}{"avatar_url": testURL}
	url, err := authAvatar.AvatarURL(client)
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
	m := md5.New()
	_, _ = io.WriteString(m, strings.ToLower("mail@example.com"))
	client.userData = map[string]interface{}{
		"userid": fmt.Sprintf("%x", m.Sum(nil)),
	}
	url, err := gravatarAvatar.AvatarURL(client)
	if err != nil {
		t.Error("GravatarAvatar.AvatarURLはエラーを返すべきではありません")
	}
	if url != "//www.gravatar.com/avatar/"+fmt.Sprintf("%x", client.userData["userid"]) {
		t.Errorf("GravatarAvatar.AvatarURLが%sという誤った値を返しました", url)
	}
}

func TestFileSystemAvatar(t *testing.T) {
	filename := filepath.Join("avatars", "abc.jpg")
	_ = os.WriteFile(filename, []byte{}, 0600)
	defer func() { os.Remove(filename) }()

	var fileSystemAvatar FileSystemAvatar
	client := new(client)
	client.userData = map[string]interface{}{"userid": "abc"}
	url, err := fileSystemAvatar.AvatarURL(client)
	if err != nil {
		t.Error("FileSystemAvatar.AvatarURLはエラーを返すべきではありません")
	}
	if url != "/avatars/abc.jpg" {
		t.Errorf("FileSystemAvatar.AvatarURLが%sという誤った値を返しました", url)
	}
}
