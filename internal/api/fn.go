package api

import (
	"bufio"
	"cmder/internal/config"
	"log/slog"
	"net/http"
	"strings"
)

func ipWhiteList(r *http.Request) bool {
	ip := strings.Split(r.RemoteAddr, ":")[0]
	slog.Info("来访地址...", slog.String("IP", ip))
	slog.Info("whiteList... ", slog.Any("ips", config.GetProxy().WhiteList))
	for _, rip := range config.GetProxy().WhiteList {
		slog.Info("whiteIp ...", slog.String("IP", rip))
		if strings.TrimSpace(rip) == ip {
			return true
		}
	}
	return false
}

func streamOutput(reader *bufio.Reader, t *task) {
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		t.broadcast(line)
	}
}
