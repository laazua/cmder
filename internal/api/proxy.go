package api

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"text/template"

	"cmder/internal/config"
)

// Index 渲染模板文件
func Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// -- 解析操作系统上的模板文件
	// tmplPath := filepath.Join("web", "index.html")
	// tmpl, err := template.ParseFiles(tmplPath)
	// if err != nil {
	// 	http.Error(w, "加载模板文件失败: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// -- 解析嵌入内存的模板内容
	tmpl, err := template.New("index").Parse(embeddedIndexHTML)
	if err != nil {
		http.Error(w, "加载模板失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "渲染模板文件失败: "+err.Error(), http.StatusInternalServerError)
	}
}

// Forward 请求转发接口
func Forward(w http.ResponseWriter, r *http.Request) {
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
		slog.Info("代理转发websocket请求...", slog.String("Uri", r.URL.Path))
		forwardWebSocket(w, r, targetURI)
	} else {
		slog.Info("代理转发http请求...", slog.String("Uri", r.URL.Path))
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
