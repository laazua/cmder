package api

import (
	"errors"
	"sync"
)

type taskManager struct {
	mu    sync.Mutex
	tasks map[string]*task
}

var ErrTooManyTasks = errors.New("已经达到了运行任务的最大数量")
var tasks = &taskManager{tasks: make(map[string]*task)}

func (m *taskManager) Set(id string, t *task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.tasks) >= 10 { // 可加并发上限
		return ErrTooManyTasks
	}
	m.tasks[id] = t
	return nil
}

func (m *taskManager) Get(id string) (*task, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	return t, ok
}

func (m *taskManager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, id)
}

func (m *taskManager) All() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.tasks))
	for id := range m.tasks {
		ids = append(ids, id)
	}
	return ids
}
