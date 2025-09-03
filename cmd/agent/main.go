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
	mux.HandleFunc("POST /api/cmd/add", api.Key(api.AddCmd))
	mux.HandleFunc("GET /api/cmd/out", api.Key(api.OutCmd))
	mux.HandleFunc("GET /api/cmd/ids", api.Key(api.ListTask))
	// 资源占用情况调试
	// go func() {
	// 	for {
	// 		fmt.Println("当前协程数量：", runtime.NumGoroutine())
	// 		time.Sleep(5 * time.Second)
	// 	}

	// }()
	server := http.Server{
		Addr:         config.GetAgent().Addr,
		Handler:      mux,
		ReadTimeout:  config.GetAgent().ReadTimeout,
		WriteTimeout: config.GetAgent().WriteTimeout,
	}

	start := make(chan error, 1)
	quit := make(chan os.Signal, 1)

	// 协程启动服务
	go func() {
		slog.Info("Agent启动...", slog.String("Addr", config.GetAgent().Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			start <- err
		}
	}()
	// 监听失败和退出信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case err := <-start:
		slog.Error("Agent启动失败", slog.String("Error", err.Error()))
	case sig := <-quit:
		slog.Info("Agent关闭,并清理资源", slog.String("Signal", sig.String()))
	}
}
