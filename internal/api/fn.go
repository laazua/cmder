package api

import (
	"bufio"
	"cmder/internal/config"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// ---------------- 工具函数 ----------------

// streamOutput 缓存io行读取标准输出和标准错误
func streamOutput(reader *bufio.Reader, t *task) {
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		t.broadcast(line)
	}
}

// extractIP 提取请求中的客户端 IP（X-Forwarded-For > X-Real-IP > RemoteAddr）
func extractIP(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		for _, p := range parts {
			ip := strings.TrimSpace(p)
			if ip != "" {
				if host, _, err := net.SplitHostPort(ip); err == nil {
					return host
				}
				return ip
			}
		}
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		if host, _, err := net.SplitHostPort(xr); err == nil {
			return host
		}
		return strings.TrimSpace(xr)
	}
	if ra := r.RemoteAddr; ra != "" {
		if host, _, err := net.SplitHostPort(ra); err == nil {
			return host
		}
		return ra
	}
	return ""
}

// iPInWhiteList 检查 IP 是否在白名单中，支持 IP 和 CIDR
func iPInWhiteList(provider config.WhiteListProvider, r *http.Request) bool {
	ipStr := extractIP(r)
	if ipStr == "" {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	slog.Info("来访地址", slog.String("IP", ipStr))

	for _, entry := range provider.GetWhiteList() {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// CIDR 支持
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err != nil {
				slog.Warn("解析 CIDR 失败", slog.String("entry", entry), slog.Any("err", err))
				continue
			}
			if cidr.Contains(ip) {
				slog.Info("命中白名单(CIDR)", slog.String("entry", entry))
				return true
			}
			continue
		}
		// 直接 IP 匹配
		if entry == ipStr {
			slog.Info("命中白名单(IP)", slog.String("entry", entry))
			return true
		}
		if parsed := net.ParseIP(entry); parsed != nil && parsed.Equal(ip) {
			slog.Info("命中白名单(IP parsed)", slog.String("entry", entry))
			return true
		}
	}
	return false
}

// isWebSocketRequest 是否是websocket请求
func isWebSocketRequest(r *http.Request) bool {
	// Connection 可能是 "Upgrade, keep-alive"
	if !headerHasToken(r.Header, "Connection", "upgrade") {
		return false
	}
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// headerHasToken 检查请求头中是否包含websocket协议头字段
func headerHasToken(h http.Header, key, want string) bool {
	for _, v := range h.Values(key) {
		for _, p := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(p), want) {
				return true
			}
		}
	}
	return false
}

// singleJoinPath req.URI保证只有一个 / 作为分隔符
func singleJoinPath(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	case strings.HasSuffix(a, "/") && strings.HasPrefix(b, "/"):
		return a + b[1:]
	case !strings.HasSuffix(a, "/") && !strings.HasPrefix(b, "/"):
		return a + "/" + b
	default:
		return a + b
	}
}

// canonicalSet 字符串切片转集合
func canonicalSet(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[http.CanonicalHeaderKey(k)] = struct{}{}
	}
	return m
}

// copyHeaders 复制请求头
func copyHeaders(dst, src http.Header, skip map[string]struct{}) {
	for k, vs := range src {
		ck := http.CanonicalHeaderKey(k)
		if skip != nil {
			if _, ok := skip[ck]; ok {
				continue
			}
		}
		for _, v := range vs {
			dst.Add(ck, v)
		}
	}
}

// forwardHTTP 转发http请求
func forwardHTTP(w http.ResponseWriter, r *http.Request, targetURI string) {
	u, err := url.Parse(targetURI)
	if err != nil {
		http.Error(w, "无效的uri: "+err.Error(), http.StatusInternalServerError)
		return
	}
	u.Path = singleJoinPath(u.Path, r.URL.Path)
	u.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(r.Context(), r.Method, u.String(), r.Body)
	if err != nil {
		http.Error(w, "新建转发请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 透传头
	req.Header.Set("X-Security-Key", config.GetProxy().XSecurityKey)
	copyHeaders(req.Header, r.Header, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "http转发出错: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header, nil)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// forwardWebSocket 转发websocket请求
func forwardWebSocket(w http.ResponseWriter, r *http.Request, targetURI string) {
	// 1) 构造后端 ws/wss URL
	wsURL, err := url.Parse(targetURI)
	if err != nil {
		slog.Error("无效的uri", slog.String("Err", err.Error()))
		http.Error(w, "无效的uri: "+err.Error(), http.StatusInternalServerError)
		return
	}
	switch strings.ToLower(wsURL.Scheme) {
	case "http":
		wsURL.Scheme = "ws"
	case "https":
		wsURL.Scheme = "wss"
	}
	wsURL.Path = singleJoinPath(wsURL.Path, r.URL.Path)
	wsURL.RawQuery = r.URL.RawQuery
	// 2) Dial 到后端：过滤会由 Dialer 自动设置/可能导致重复的头
	//    注意：必须使用 Canonical 形式（Sec-Websocket-Key 等）
	skip := canonicalSet(
		"Connection",
		"Upgrade",
		"Sec-WebSocket-Key",
		"Sec-WebSocket-Version",
		"Sec-WebSocket-Extensions",
		"Sec-WebSocket-Accept",
		"Sec-WebSocket-Protocol", // 协议列表单独处理
		"Host",                   // 让 Dialer 根据 URL 设置
	)

	backendHeaders := http.Header{}
	// 透传头
	r.Header.Set("X-Security-Key", config.GetProxy().XSecurityKey)
	copyHeaders(backendHeaders, r.Header, skip)

	// 可选：把客户端请求的子协议传给后端（但不要放到 header，交给 Dialer.Subprotocols）
	var subprotocols []string
	if sp := r.Header.Get("Sec-WebSocket-Protocol"); sp != "" {
		for _, p := range strings.Split(sp, ",") {
			if v := strings.TrimSpace(p); v != "" {
				subprotocols = append(subprotocols, v)
			}
		}
	}

	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  30 * time.Second,
		EnableCompression: false, // 避免压缩带来的复杂性
		Subprotocols:      subprotocols,
	}
	backendConn, _, err := dialer.Dial(wsURL.String(), backendHeaders)
	if err != nil {
		slog.Error("拨号失败...", slog.String("Err", err.Error()))
		http.Error(w, "拨号agent失败: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	// 3) 升级与客户端的连接（不声明 Subprotocol，由于我们只是转发，一般无需协商）
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // 如需安全控制，自行校验
		// Subprotocols: nil // 不与客户端协商子协议，避免与后端不一致
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade client failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// 4) 双向转发
	errc := make(chan error, 2)

	go proxyCopy(errc, clientConn, backendConn) // client -> backend
	go proxyCopy(errc, backendConn, clientConn) // backend -> client

	<-errc // 任一方向断开就退出
}

// proxyCopy websocket数据双向转发
func proxyCopy(errc chan<- error, src, dst *websocket.Conn) {
	for {
		mt, msg, err := src.ReadMessage()
		if err != nil {
			errc <- err
			return
		}
		if err := dst.WriteMessage(mt, msg); err != nil {
			errc <- err
			return
		}
	}
}

//

func forbiddenCmds(cmd string) bool {
	for _, fcmd := range config.GetAgent().ForbiddenCmds {
		if strings.Contains(cmd, fcmd) {
			return true
		}
	}
	return false
}
