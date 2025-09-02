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
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func AddCmd(w http.ResponseWriter, r *http.Request) {
	slog.Info("/api/cmd/run ...")
	var req struct {
		Name string `json:"name"`
		Cmd  string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求参数错误", http.StatusBadRequest)
		return
	}

	taskId := uuid.New().String()
	cmd := exec.Command("bash", "-c", req.Cmd)

	tk := &task{
		Id:    taskId,
		Cmd:   cmd,
		State: "created",
	}
	if err := tasks.Set(taskId, tk); err != nil {
		slog.Error("新增任务失败", slog.String("Err", err.Error()))
		http.Error(w, "已经达到了运行任务的最大数量", http.StatusTooManyRequests)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"task_id": taskId})
}

func OutCmd(w http.ResponseWriter, r *http.Request) {
	slog.Info("/api/cmd/out ...")
	taskId := r.URL.Query().Get("task_id")
	rtask, ok := tasks.Get(taskId)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	rtask.Mu.Lock()
	stdout, _ := rtask.Cmd.StdoutPipe()
	stderr, _ := rtask.Cmd.StderrPipe()
	if !rtask.started {
		if err := rtask.Cmd.Start(); err != nil {
			rtask.Mu.Unlock()
			http.Error(w, "运行任务失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
		rtask.State = "running"
		rtask.started = true
	}
	rtask.Mu.Unlock()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// 获取标准输出
	runLimited(func() {
		stdoutReader := bufio.NewReader(stdout)
		for {
			line, _, err := stdoutReader.ReadLine()
			if err != nil {
				break
			}
			conn.WriteMessage(websocket.TextMessage, line)
		}
	})

	// 获取标准错误
	runLimited(func() {
		stderrReader := bufio.NewReader(stderr)
		for {
			line, _, err := stderrReader.ReadLine()
			if err != nil {
				break
			}
			conn.WriteMessage(websocket.TextMessage, line)
		}
	})

	// 等待进程结束
	err = rtask.Cmd.Wait()

	rtask.Mu.Lock()
	defer rtask.Mu.Unlock()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				rtask.ExitCode = status.ExitStatus()
			}
		}
		rtask.State = "failed"
	} else {
		if status, ok := rtask.Cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			rtask.ExitCode = status.ExitStatus()
		}
		rtask.State = "success"
	}

	// 执行完成后清理
	tasks.Delete(taskId)
}

func ListTask(w http.ResponseWriter, r *http.Request) {
	slog.Info("/api/cmd/list ...")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"target": r.URL.Query().Get("name"),
		"tasks":  tasks.All(),
	})
}
