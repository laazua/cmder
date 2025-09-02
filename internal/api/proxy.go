package api

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"text/template"

	"cmder/internal/config"
)

func Index(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join("web", "index.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "failed to load template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "failed to render template: "+err.Error(), http.StatusInternalServerError)
	}
}

// 入口：一个路由同时支持 HTTP & WebSocket
func Forward(w http.ResponseWriter, r *http.Request) {
	slog.Info("proxy forward", slog.String("uri", r.URL.RequestURI()))

	targetName := r.URL.Query().Get("name")
	var targetURI string
	for _, t := range config.GetProxy().Targets {
		if t.Name == targetName {
			targetURI = t.Address // e.g. http://127.0.0.1:6000
			break
		}
	}
	if targetURI == "" {
		http.Error(w, "目标主机未配置到", http.StatusNotFound)
		return
	}

	if isWebSocketRequest(r) {
		forwardWebSocket(w, r, targetURI)
	} else {
		forwardHTTP(w, r, targetURI)
	}
}
