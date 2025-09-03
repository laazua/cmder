package config

import (
	"errors"
	"time"
)

var (
	// 预留命令行参数
	AgentFile string
	agent     *loader[Agent]
)

type Agent struct {
	Addr         string        `yaml:"addr" default:"localhost:5544"`
	TaskNum      int           `yaml:"taskNum" default:"8"`
	ReadTimeout  time.Duration `yaml:"readTimeout" default:"60m"`
	WriteTimeout time.Duration `yaml:"writeTimeout" default:"60m"`
	XSecurityKey string        `yaml:"xSecurityKey" default:"xSecurityKey"`
	WhiteList    []string      `ymal:"whiteList" default:"127.0.0.1"`
}

func (a *Agent) Validate() error {
	if a.Addr == "" {
		return errors.New("addr must not be empty")
	}
	return nil
}

func GetAgent() *Agent {
	if AgentFile == "" {
		AgentFile = "./config.yaml"
	}
	return getConfig(&agent, AgentFile)
}
