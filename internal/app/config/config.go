// Package config 定义模拟器的 YAML 可序列化配置。
//
// 独立为一个小包的好处：
//   - main.go 中无需 import yaml.v3，保持入口干净
//   - 代码库其余部分只依赖这一个稳定的 Config 类型
//   - 支持多文件叠加覆盖（用于 dev / test / prod 不同配置分层）
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the merged configuration consumed by the rest of the process.
type Config struct {
	Node             NodeConfig     `yaml:"node"`
	Protocol         ProtocolConfig `yaml:"protocol"`
	Control          ControlConfig  `yaml:"control"`
	Storage          StorageConfig  `yaml:"storage"`
	Auth             AuthConfig     `yaml:"auth"`
	ScenariosDir     string         `yaml:"scenariosDir"`
	ScenarioSeed     int64          `yaml:"scenarioSeed"`
	KeepaliveInterval time.Duration  `yaml:"keepaliveInterval"` // 心跳周期，0=禁用
}

// NodeConfig 存储 20 位 DeviceID 生成器使用的区划码 / 行业代码。
// SiteCode 为前 8 位；IndustryCode 默认 130（社会公共安全 / 视频图像行业）。
type NodeConfig struct {
	SiteCode     int `yaml:"siteCode"`
	IndustryCode int `yaml:"industryCode"`
}

// ProtocolConfig 控制 GA/T 1400 REST 服务端监听地址（:19001）。
type ProtocolConfig struct {
	Listen string `yaml:"listen"`
}

// ControlConfig 控制 Web BFF 监听地址（:19000）。
type ControlConfig struct {
	Listen string `yaml:"listen"`
}

// StorageConfig 控制 nonce 与抓包使用的 SQLite 数据库文件路径。
type StorageConfig struct {
	Path string `yaml:"path"`
}

// AuthConfig 控制向客户端出示的 Digest 认证凭据（realm / username / password / qop）。
type AuthConfig struct {
	Realm    string `yaml:"realm"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Qop      string `yaml:"qop"`
}

// Default 返回进程内默认值。值与 YAML schema 对应，
// 调用方可以在文件配置和代码配置之间切换而不改变字段名。
func Default() *Config {
	return &Config{
		Node: NodeConfig{
			SiteCode:     4100000000,
			IndustryCode: 130,
		},
		Protocol: ProtocolConfig{Listen: ":14000"},
		Control:  ControlConfig{Listen: ":14080"},
		Storage:  StorageConfig{Path: "./data/gat1400.db"},
		Auth: AuthConfig{
			Realm:    "com.gat1400.simulator",
			Username: "admin",
			Password: "admin",
			Qop:      "auth",
		},
		ScenariosDir:      "./configs/scenarios",
		KeepaliveInterval: 30 * time.Second,
	}
}

// Load 返回叠加后的配置：先取 Default()，再依次用 YAML 文件覆盖。
// 不存在的文件静默跳过；首次解析失败触发错误。
func Load(paths ...string) (*Config, error) {
	c := Default()
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("config: read %s: %w", p, err)
		}
		if err := yaml.Unmarshal(b, c); err != nil {
			return nil, fmt.Errorf("config: parse %s: %w", p, err)
		}
	}
	return c, nil
}
