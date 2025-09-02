package api

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// ---------------- HTTP 转发 ----------------

func forwardHTTP(w http.ResponseWriter, r *http.Request, targetURI string) {
	u, err := url.Parse(targetURI)
	if err != nil {
		http.Error(w, "bad target uri: "+err.Error(), http.StatusInternalServerError)
		return
	}
	u.Path = singleJoinPath(u.Path, r.URL.Path)
	u.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(r.Context(), r.Method, u.String(), r.Body)
	if err != nil {
		http.Error(w, "new request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 透传头
	copyHeaders(req.Header, r.Header, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "proxy http error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header, nil)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// ---------------- WebSocket 转发 ----------------

func forwardWebSocket(w http.ResponseWriter, r *http.Request, targetURI string) {
	// 1) 构造后端 ws/wss URL
	wsURL, err := url.Parse(targetURI)
	if err != nil {
		http.Error(w, "invalid target uri: "+err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "dial backend failed: "+err.Error(), http.StatusBadGateway)
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

// ---------------- 工具函数 ----------------

func isWebSocketRequest(r *http.Request) bool {
	// Connection 可能是 "Upgrade, keep-alive"
	if !headerHasToken(r.Header, "Connection", "upgrade") {
		return false
	}
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

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

func canonicalSet(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[http.CanonicalHeaderKey(k)] = struct{}{}
	}
	return m
}

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
