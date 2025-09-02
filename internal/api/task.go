package api

import (
	"fmt"
	"os/exec"
	"sync"

	"cmder/internal/config"
)

// 在 package 级别定义一个全局信号量
// 限制最大并发 goroutine 数量，比如 100
var goroutineLimit = make(chan struct{}, config.GetAgent().TaskNum*2)

// 封装一个 helper
func runLimited(fn func()) {
	goroutineLimit <- struct{}{} // 获取一个名额
	go func() {
		defer func() { <-goroutineLimit }() // 释放名额
		fn()
	}()
}

var tasks = newFixedSizeMap()

type task struct {
	Id       string
	Cmd      *exec.Cmd
	State    string
	ExitCode int
	Mu       sync.Mutex
	started  bool
}

type fixedSizeMap struct {
	maxSize int
	data    map[string]*task
	mu      sync.RWMutex
}

func newFixedSizeMap() *fixedSizeMap {
	size := config.GetAgent().TaskNum
	return &fixedSizeMap{
		maxSize: size,
		data:    make(map[string]*task, size),
	}
}

// Set 添加元素，如果超过大小则返回错误
func (f *fixedSizeMap) Set(key string, value *task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.data) >= f.maxSize {
		if _, exists := f.data[key]; !exists {
			return fmt.Errorf("map is full, max size: %d", f.maxSize)
		}
	}
	f.data[key] = value
	return nil
}

// Get 获取元素
func (f *fixedSizeMap) Get(key string) (*task, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	value, exists := f.data[key]
	return value, exists
}

// Delete 删除元素
func (f *fixedSizeMap) Delete(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key)
}

// Size 返回当前元素数量
func (f *fixedSizeMap) Size() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.data)
}

func (f *fixedSizeMap) All() []string {
	var tks []string
	if f.Size() == 0 {
		return nil
	}
	for _, v := range f.data {
		tks = append(tks, v.Id)
	}
	return tks
}

// MaxSize 返回最大容量
func (f *fixedSizeMap) MaxSize() int {
	return f.maxSize
}
