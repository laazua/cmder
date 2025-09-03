package api

import (
	"io"
	"os/exec"
	"sync"

	"github.com/gorilla/websocket"
)

const maxLogBuffer = 200 // 日志缓存行数

type task struct {
	Id      string
	Cmd     *exec.Cmd
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	started bool

	mu       sync.Mutex
	clients  map[*websocket.Conn]struct{} // 多个 WS 客户端
	finished bool

	logBuffer [][]byte // 最近日志缓存
}

func newTask(id string, cmd *exec.Cmd) (*task, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	return &task{
		Id:        id,
		Cmd:       cmd,
		stdout:    stdout,
		stderr:    stderr,
		clients:   make(map[*websocket.Conn]struct{}),
		logBuffer: make([][]byte, 0, maxLogBuffer),
	}, nil
}

func (t *task) addClient(conn *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 如果任务已结束，直接推送历史日志并关闭
	if t.finished {
		for _, line := range t.logBuffer {
			conn.WriteMessage(websocket.TextMessage, line)
		}
		conn.WriteMessage(websocket.TextMessage, []byte("任务已结束"))
		conn.Close()
		return
	}

	// 推送历史日志
	for _, line := range t.logBuffer {
		conn.WriteMessage(websocket.TextMessage, line)
	}

	t.clients[conn] = struct{}{}
}

func (t *task) broadcast(msg []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 写入日志缓存
	t.appendLog(msg)

	// 广播给所有客户端
	for conn := range t.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			conn.Close()
			delete(t.clients, conn)
		}
	}
}

func (t *task) appendLog(msg []byte) {
	if len(t.logBuffer) >= maxLogBuffer {
		t.logBuffer = t.logBuffer[1:] // 丢弃最旧
	}
	// 复制一份避免 slice 被复用
	cp := make([]byte, len(msg))
	copy(cp, msg)
	t.logBuffer = append(t.logBuffer, cp)
}

func (t *task) closeAll(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 任务结束消息也写入缓存
	t.appendLog([]byte(msg))

	for conn := range t.clients {
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		conn.Close()
	}
	t.clients = nil
	t.finished = true
}
