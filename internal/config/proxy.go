package config

import (
	"errors"
	"time"
)

var (
	// 预留命令行参数
	ProxyFile string
	proxy     *loader[Proxy]
)

type Proxy struct {
	Addr         string        `yaml:"addr" default:"localhost:5533"`
	ReadTimeout  time.Duration `yaml:"readTimeout" default:"30m"`
	WriteTimeout time.Duration `yaml:"writeTimeout" default:"30m"`
	XSecurityKey string        `yaml:"xSecurityKey" default:"xSecurityKey"`
	WhiteList    []string      `ymal:"whiteList" default:"127.0.0.1"`
	Targets      []Target      `yaml:"targets"`
}

type Target struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

func (p *Proxy) Validate() error {
	if p.Addr == "" {
		return errors.New("addr must not be empty")
	}
	if p.Targets == nil {
		return errors.New("targets must not be empty")
	}
	return nil
}

func GetProxy() *Proxy {
	if ProxyFile == "" {
		ProxyFile = "./config.yaml"
	}
	return getConfig(&proxy, ProxyFile)
}
