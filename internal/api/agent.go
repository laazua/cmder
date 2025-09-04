package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// AddCmd 添加命令任务（原有）
func AddCmd(w http.ResponseWriter, r *http.Request) {
	slog.Info("/api/cmd/run ...")
	var req struct {
		Cmd string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求参数错误", http.StatusBadRequest)
		return
	}
	// 检查是否是封禁的命令
	if forbiddenCmds(req.Cmd) {
		http.Error(w, "封禁的命令,请联系管理员", http.StatusForbidden)
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

// OutCmd 执行任务并获取输出（原有）
func OutCmd(w http.ResponseWriter, r *http.Request) {
	slog.Info("/api/cmd/out ...")
	taskId := r.URL.Query().Get("task_id")
	rtask, ok := tasks.Get(taskId)
	if !ok {
		http.Error(w, "任务未找到", http.StatusNotFound)
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
						rtask.closeAll("=============== 命令运行异常退出 ===============")
					}
				}
			} else {
				if _, ok := rtask.Cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
					rtask.closeAll("=============== 命令运行正常退出 ===============")
				}
			}
			tasks.Delete(taskId)
		}()
	}
	rtask.mu.Unlock()
}

// ListTask 查询添加了哪些命令任务（原有）
func ListTask(w http.ResponseWriter, r *http.Request) {
	slog.Info("/api/cmd/ids ...")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"target": r.URL.Query().Get("name"),
		"tasks":  tasks.All(),
	})
}

// RunScriptWS 通过 WebSocket 输入脚本并执行
// 协议：客户端建立 WS 后可立即收到 {"task_id": "...", "status":"started"}；
// 然后把脚本文本（可多次分片发送）以 TextMessage 发送；
// 发送 "__EOF__" 表示脚本发送完毕，服务器关闭 stdin 并等待退出；
// 期间实时把 stdout/stderr 的行回写给客户端；退出后发送 {"status":"exit","code":0} 并关闭 WS。
func RunScriptWS(w http.ResponseWriter, r *http.Request) {
	slog.Info("/api/cmd/runws ...")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	taskID := uuid.New().String()

	// 用 bash -s 从 stdin 读取脚本
	cmd := exec.Command("bash", "-s")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("init stdin failed: "+err.Error()))
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("init stdout failed: "+err.Error()))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("init stderr failed: "+err.Error()))
		return
	}

	// 告知客户端 task_id
	_ = conn.WriteJSON(map[string]any{
		"task_id": taskID,
		"status":  "started",
		"time":    time.Now().Format(time.RFC3339),
	})

	if err := cmd.Start(); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("start failed: "+err.Error()))
		return
	}

	// 读客户端脚本文本 -> 写入 bash stdin
	doneWrite := make(chan struct{})
	go func() {
		defer close(doneWrite)
		defer func() { _ = stdin.Close() }()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				// 客户端断开或读取失败，直接结束写端
				return
			}
			if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
				continue
			}
			// 约定 "__EOF__" 作为脚本结束
			if bytes.Equal(msg, []byte("__EOF__")) {
				return
			}
			// 写入脚本内容，并确保以换行结尾
			if _, err := stdin.Write(msg); err != nil {
				return
			}
			if len(msg) == 0 || msg[len(msg)-1] != '\n' {
				_, _ = stdin.Write([]byte("\n"))
			}
		}
	}()

	// 实时把 stdout/stderr 行回写给客户端
	go func() {
		reader := bufio.NewReader(stdout)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				return
			}
			_ = conn.WriteMessage(websocket.TextMessage, line)
		}
	}()
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				return
			}
			_ = conn.WriteMessage(websocket.TextMessage, line)
		}
	}()

	// 等待客户端完成发送
	<-doneWrite

	// 等待进程退出并回传退出码
	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				_ = conn.WriteJSON(map[string]any{"status": "exit", "code": ws.ExitStatus()})
				return
			}
		}
		_ = conn.WriteJSON(map[string]any{"status": "exit", "error": err.Error()})
		return
	}
	if ps := cmd.ProcessState; ps != nil {
		if ws, ok := ps.Sys().(syscall.WaitStatus); ok {
			_ = conn.WriteJSON(map[string]any{"status": "exit", "code": ws.ExitStatus()})
			return
		}
	}
	_ = conn.WriteJSON(map[string]any{"status": "exit", "code": 0})
}
