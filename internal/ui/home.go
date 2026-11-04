package ui

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var home []byte

func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(home)
}
