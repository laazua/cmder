// 因为Agent和Proxy的配置类的字段相似
// 所有让Agent和Proxy实现本文件中的接口
// 则可以在获取配置项时根据指定的类型来获取该类型的具体配置项

package config

import "time"

// WhiteListProvider 定义需要提供白名单的配置
type WhiteListProvider interface {
	GetWhiteList() []string
}

// KeyProvider 提供获取安全密钥的方法
type KeyProvider interface {
	GetXSecurityKey() string
}

// TimeRestrictedProvider提供时间段控制访问
type TimeRestrictedProvider interface {
	GetAcEndTime() time.Duration
	GetAcStartTime() time.Duration
}
