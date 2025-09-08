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
	Addr            string        `yaml:"addr" default:"localhost:5533"`       // 监听地址
	ReadTimeout     time.Duration `yaml:"readTimeout" default:"30m"`           // http读超时
	WriteTimeout    time.Duration `yaml:"writeTimeout" default:"30m"`          // http写超时
	XSecurityKey    string        `yaml:"xSecurityKey" default:"xSecurityKey"` // 密钥
	AccessStartTime time.Duration `yaml:"accessStartTime" default:"9h"`        // 允许访问开始时间
	AccessEndTime   time.Duration `yaml:"accessEndTime" default:"18h"`         // 允许访问结束时间
	WhiteList       []string      `yaml:"whiteList"`                           // IP白名单
	Targets         []Target      `yaml:"targets"`
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
func (p *Proxy) GetAcStartTime() time.Duration {
	return p.AccessStartTime
}
func (p *Proxy) GetAcEndTime() time.Duration {
	return p.AccessEndTime
}

func GetProxy() *Proxy {
	if ProxyFile == "" {
		ProxyFile = "./config.yaml"
	}
	return getConfig(&proxy, ProxyFile)
}
