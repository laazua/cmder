package api

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"net/http"
	"os/exec"
	"syscall"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func AddCmd(w http.ResponseWriter, r *http.Request) {
	if !ipWhiteList(r) {
		slog.Info("你不允许访问")
		http.Error(w, "你想干嘛", http.StatusForbidden)
		return
	}
	slog.Info("/api/cmd/run ...")
	var req struct {
		Cmd string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求参数错误", http.StatusBadRequest)
		return
	}

	taskId := uuid.New().String()
	cmd := exec.Command("bash", "-c", req.Cmd)

	tk, err := newTask(taskId, cmd)
	if err != nil {
		http.Error(w, "初始化任务失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tasks.Set(taskId, tk); err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"task_id": taskId})
}

func OutCmd(w http.ResponseWriter, r *http.Request) {
	if !ipWhiteList(r) {
		slog.Info("你不允许访问")
		http.Error(w, "你想干嘛", http.StatusForbidden)
		return
	}
	slog.Info("/api/cmd/out ...")
	taskId := r.URL.Query().Get("task_id")
	rtask, ok := tasks.Get(taskId)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	rtask.addClient(conn)

	// 启动任务，只会执行一次
	rtask.mu.Lock()
	if !rtask.started {
		if err := rtask.Cmd.Start(); err != nil {
			rtask.mu.Unlock()
			http.Error(w, "运行任务失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
		rtask.started = true

		// 异步读取 stdout/stderr 并广播
		go streamOutput(bufio.NewReader(rtask.stdout), rtask)
		go streamOutput(bufio.NewReader(rtask.stderr), rtask)

		// 等待进程退出
		go func() {
			err := rtask.Cmd.Wait()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					if _, ok := exitErr.Sys().(syscall.WaitStatus); ok {
						rtask.closeAll("命令运行异常退出")
					}
				}
			} else {
				if _, ok := rtask.Cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
					rtask.closeAll("命令运行正常退出")
				}
			}
			tasks.Delete(taskId)
		}()
	}
	rtask.mu.Unlock()
}

func ListTask(w http.ResponseWriter, r *http.Request) {
	if !ipWhiteList(r) {
		slog.Info("你不允许访问")
		http.Error(w, "你想干嘛", http.StatusForbidden)
		return
	}
	slog.Info("/api/cmd/ids ...")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"target": r.URL.Query().Get("name"),
		"tasks":  tasks.All(),
	})
}
