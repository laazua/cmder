package api

import (
	"bufio"
	"cmder/internal/config"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

// ---------------- 工具函数 ----------------

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
