package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func uploaderHandler(w http.ResponseWriter, req *http.Request) {
	userID := req.FormValue("userid")
	file, header, err := req.FormFile("avatarFile")
	if err != nil {
		_, _ = io.WriteString(w, err.Error())
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		_, _ = io.WriteString(w, err.Error())
		return
	}
	filename := filepath.Join("avatars", userID+filepath.Ext(header.Filename))
	if err := os.WriteFile(filename, data, 0600); err != nil {
		_, _ = io.WriteString(w, err.Error())
		return
	}
	_, _ = io.WriteString(w, "Successful")
}
