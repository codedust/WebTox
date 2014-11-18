package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Path
	fmt.Println("[handleHTTP]", file)

	if file[0] == '/' {
		file = file[1:]
	}

	if len(file) == 0 {
		file = "index.html"
	}

	p := filepath.Join("../html", filepath.FromSlash(file))
	_, err := os.Stat(p)
	if err == nil {
		http.ServeFile(w, r, p)
		return
	} else {
		http.NotFound(w, r)
		return
	}
}
