package handler

import (
	_ "embed"
	"net/http"
)

//go:embed README.html
var readmeFile []byte

func HanlderHelp(w http.ResponseWriter, r *http.Request) {
	w.Write(readmeFile)
}
