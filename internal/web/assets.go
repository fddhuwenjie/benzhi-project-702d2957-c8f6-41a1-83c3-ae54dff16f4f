package web

import (
	"embed"
	"net/http"
)

//go:embed static/index.html static/app.css static/app.js
var assets embed.FS

func serveAsset(w http.ResponseWriter, name, contentType string) {
	b, err := assets.ReadFile("static/" + name)
	if err != nil {
		http.Error(w, "资源不存在", 404)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(200)
	_, _ = w.Write(b)
}
func (s *Server) HandleWorkbench(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	serveAsset(w, "index.html", "text/html; charset=utf-8")
}
func (s *Server) HandleCSS(w http.ResponseWriter, r *http.Request) {
	serveAsset(w, "app.css", "text/css; charset=utf-8")
}
func (s *Server) HandleJS(w http.ResponseWriter, r *http.Request) {
	serveAsset(w, "app.js", "text/javascript; charset=utf-8")
}
