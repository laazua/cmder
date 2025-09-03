package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"text/template"

	"cmder/internal/config"
)

// Index 渲染模板文件
func Index(w http.ResponseWriter, r *http.Request) {
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

// Forward 请求转发接口
func Forward(w http.ResponseWriter, r *http.Request) {

	slog.Info("代理转发请求...", slog.String("uri", r.URL.RequestURI()))

	targetName := r.URL.Query().Get("name")
	var targetURI string
	for _, t := range config.GetProxy().Targets {
		if t.Name == targetName {
			targetURI = t.Address
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

// targets 获取target列表

func Targets(w http.ResponseWriter, r *http.Request) {
	var targets []string
	for _, target := range config.GetProxy().Targets {
		targets = append(targets, target.Name)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"targets": targets,
	})
}
