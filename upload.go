package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

func uploaderHandler(c echo.Context) error {
	file, err := c.FormFile("avatarFile")
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
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

	userID := filepath.Base(c.FormValue("userid"))
	if userID == "" || userID == "." || userID == ".." {
		return c.String(http.StatusBadRequest, "invalid userid")
	}
	filename := filepath.Join("avatars", userID+filepath.Ext(file.Filename))
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, "Successful")
}
