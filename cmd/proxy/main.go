package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"cmder/internal/api"
	"cmder/internal/config"
)

func main() {

	mux := http.NewServeMux()
	// 这里填写的"/api/cmd/" 是agent端的上层路由
	// agent中路由如下:
	//     /api/cmd/add
	//     /api/cmd/out
	//     /api/cmd/state
	mux.HandleFunc("/", api.Index)
	mux.HandleFunc("/api/cmd/", api.Forward)
	server := http.Server{
		Addr:         config.GetProxy().Addr,
		Handler:      mux,
		ReadTimeout:  config.GetProxy().ReadTimeout,
		WriteTimeout: config.GetProxy().WriteTimeout,
	}

	start := make(chan error, 1)
	quit := make(chan os.Signal, 1)

	// 协程启动服务
	go func() {
		slog.Info("Proxy启动...", slog.String("Addr", config.GetProxy().Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			start <- err
		}
	}()
	// 监听失败和退出信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case err := <-start:
		slog.Error("Proxy启动失败", slog.String("Error", err.Error()))
	case sig := <-quit:
		slog.Info("pubot关闭,并清理资源", slog.String("Signal", sig.String()))
	}
}
