package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

const maxAvatarUploadBytes int64 = 5 << 20 // 5 MiB

func uploaderHandler(c echo.Context) error {
	req := c.Request()
	req.Body = http.MaxBytesReader(c.Response(), req.Body, maxAvatarUploadBytes)

	file, err := c.FormFile("avatarFile")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return c.String(http.StatusBadRequest, err.Error())
		}
		if strings.Contains(err.Error(), "request body too large") {
			return c.String(http.StatusRequestEntityTooLarge, "file is too large")
		}
		return c.String(http.StatusBadRequest, err.Error())
	}
	if file.Size > maxAvatarUploadBytes {
		return c.String(http.StatusRequestEntityTooLarge, "file is too large")
	}

	src, err := file.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	defer func() { _ = src.Close() }()

	data, err := io.ReadAll(src)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	userData, err := getAuthUserData(c)
	if err != nil {
		return c.String(http.StatusUnauthorized, "unauthorized")
	}

	rawUserID, ok := userData["userid"].(string)
	if !ok || rawUserID == "" {
		return c.String(http.StatusUnauthorized, "invalid user")
	}

	userID := filepath.Base(rawUserID)
	if userID == "" || userID == "." || userID == ".." {
		return c.String(http.StatusBadRequest, "invalid userid")
	}
	filename := filepath.Join("avatars", userID+filepath.Ext(file.Filename))
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, "Successful")
}
