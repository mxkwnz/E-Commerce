package swagger

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed openapi.json
var openAPISpec []byte

//go:embed index.html
var indexHTML []byte

//go:embed ui/*
var uiFiles embed.FS

func Register(mux *http.ServeMux) {
	ui, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		panic(err)
	}

	mux.Handle("GET /swagger/assets/", http.StripPrefix("/swagger/assets/", http.FileServer(http.FS(ui))))

	mux.HandleFunc("GET /swagger/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(openAPISpec)
	})
	mux.HandleFunc("GET /swagger/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
}
