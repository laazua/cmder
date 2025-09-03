package api

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"text/template"

	"cmder/internal/config"
)

func Index(w http.ResponseWriter, r *http.Request) {
	if !ipWhiteList(r) {
		slog.Info("不允许访问")
		http.Error(w, "想干嘛", http.StatusForbidden)
		return
	}
	tmplPath := filepath.Join("web", "index.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "加载模板文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "渲染模板文件失败: "+err.Error(), http.StatusInternalServerError)
	}
}

// 入口：一个路由同时支持 HTTP & WebSocket
func Forward(w http.ResponseWriter, r *http.Request) {
	slog.Info("代理转发请求...", slog.String("uri", r.URL.RequestURI()))

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
	r.Header.Set("X-Security-Key", config.GetProxy().XSecurityKey)
	if isWebSocketRequest(r) {
		forwardWebSocket(w, r, targetURI)
	} else {
		forwardHTTP(w, r, targetURI)
	}
}
