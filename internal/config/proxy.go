package config

import (
	"errors"
	"time"
)

var (
	// 预留命令行参数
	ProxyFile string
	proxy     *Loader[Proxy]
)

type Proxy struct {
	Addr         string        `yaml:"addr" default:"localhost:5533"`
	ReadTimeout  time.Duration `yaml:"readTimeout" default:"30m"`
	WriteTimeout time.Duration `yaml:"writeTimeout" default:"30m"`
	XSecurityKey string        `yaml:"xSecurityKey" default:"xSecurityKey"`
	WhiteList    []string      `yaml:"whiteList"`
	Targets      []Target      `yaml:"targets"`
}

type Target struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

func (p *Proxy) Validate() error {
	if p.Addr == "" {
		return errors.New("监听地址不能为空")
	}
	if len(p.Targets) == 0 {
		return errors.New("目标主机配置不能为空")
	}
	if len(p.WhiteList) == 0 {
		return errors.New("主机白名单列表不能为空")
	}
	return nil
}

func (p *Proxy) GetWhiteList() []string {
	return p.WhiteList
}

func (p *Proxy) GetXSecurityKey() string {
	return p.XSecurityKey
}

func GetProxy() *Proxy {
	if ProxyFile == "" {
		ProxyFile = "./config.yaml"
	}
	return getConfig(&proxy, ProxyFile)
}
