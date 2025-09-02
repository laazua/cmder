package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader 泛型配置加载器
type loader[T any] struct {
	config     *T
	configPath string
	once       sync.Once
	loadErr    error
}

// newLoader 创建新的配置加载器实例
func newLoader[T any](configPath string) *loader[T] {
	return &loader[T]{
		configPath: configPath,
	}
}

// load 加载配置
func (l *loader[T]) load() error {
	l.once.Do(func() {
		filePath := filepath.Join(".", l.configPath)
		data, err := os.ReadFile(filePath)
		if err != nil {
			l.loadErr = fmt.Errorf("读取配置文件失败: %v", err)
			return
		}

		var config T
		if err := yaml.Unmarshal(data, &config); err != nil {
			l.loadErr = fmt.Errorf("解析配置文件失败: %v", err)
			return
		}

		// 调用验证方法（如果存在）
		if validator, ok := any(&config).(interface{ Validate() error }); ok {
			if err := validator.Validate(); err != nil {
				l.loadErr = fmt.Errorf("配置验证失败: %v", err)
				return
			}
		}

		l.config = &config
	})

	return l.loadErr
}

// get 获取配置实例
func (l *loader[T]) get() *T {
	if l.loadErr != nil {
		panic(fmt.Sprintf("配置未正确加载: %v", l.loadErr))
	}
	return l.config
}

// getConfig 通用配置获取函数
func getConfig[T any](loaderRef **loader[T], path string) *T {
	if *loaderRef != nil {
		return (*loaderRef).get()
	}

	l := newLoader[T](path)
	if err := l.load(); err != nil {
		panic(err)
	}

	*loaderRef = l
	return l.get()
}
