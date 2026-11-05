// Package config 应用配置与适配器。
//
// 本包负责：
//   - 配置加载（viper + flag + 环境变量）
//   - 配置结构体定义
//   - 配置向适配器转换的工具函数（ProtocolBaseURL）
package config

import (
	"strings"
)

// ProtocolBaseURL 把 ":14000" 形式监听地址转换为 BFF 可访问的 base URL。
func ProtocolBaseURL(listen string) string {
	addr := listen
	if addr == "" {
		addr = ":14000"
	}
	host := "127.0.0.1"
	if strings.HasPrefix(addr, ":") {
		return "http://" + host + addr
	}
	if strings.Contains(addr, ":") {
		return "http://" + addr
	}
	return "http://" + addr + ":14000"
}