package config

import (
	"errors"
	"time"
)

var (
	// 预留命令行参数
	AgentFile string
	AgentC    *Loader[Agent]
)

type Agent struct {
	Addr         string        `yaml:"addr" default:"localhost:5544"`
	TaskNum      int           `yaml:"taskNum" default:"8"`
	ReadTimeout  time.Duration `yaml:"readTimeout" default:"60m"`
	WriteTimeout time.Duration `yaml:"writeTimeout" default:"60m"`
	XSecurityKey string        `yaml:"xSecurityKey" default:"xSecurityKey"`
	WhiteList    []string      `yaml:"whiteList"`
}

func (a *Agent) Validate() error {
	if a.Addr == "" {
		return errors.New("监听地址不能为空")
	}
	if len(a.WhiteList) == 0 {
		return errors.New("主机白名单不能为空")
	}
	return nil
}

func (a *Agent) GetWhiteList() []string {
	return a.WhiteList
}

func (a *Agent) GetXSecurityKey() string {
	return a.XSecurityKey
}

func GetAgent() *Agent {
	if AgentFile == "" {
		AgentFile = "./config.yaml"
	}
	return getConfig(&AgentC, AgentFile)
}
