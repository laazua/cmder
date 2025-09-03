package config

// WhiteListProvider 定义需要提供白名单的配置
type WhiteListProvider interface {
	GetWhiteList() []string
}

// KeyProvider 提供获取安全密钥的方法
type KeyProvider interface {
	GetXSecurityKey() string
}
