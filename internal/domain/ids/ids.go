// Package ids 生成 GA/T 1400 协议中使用的各类标识符。
//
// 核心是 20 位 DeviceID，其布局借鉴 GB/T 28181 的 20 位 ID 编码风格
//（行政区划 / 行业 / 类型 / 网络 / 序列），保证生成的 ID 在视觉上"看起来真实"。
//
// 安全约束：
//   - Nonce 一律使用 crypto/rand 生成，严禁使用 time.Now() 等可预测源
//   - DeviceID 通过互斥锁保证并发安全，单进程内全局唯一
package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Generator 线程安全地签发符合协议规范的标识符。
// siteCode 与 industryCode 在构造时固定，使同一进程签发的所有 ID 共享相同前缀。
type Generator struct {
	mu       sync.Mutex // 保护 seq 自增与未来其他共享字段
	siteCode uint32     // 区划码，8 位十进制
	indCode  uint32     // 行业代码，2 位（130=公安视频图像对应行业 30）
	typeCode uint32     // 设备类型代码，2 位
	seq      uint32     // 6 位滚动序列号
}

// NewGenerator 构造一个 ID 生成器。
//
// 参数：
//   - siteCode：8 位行政区划码（如 41000000 表示河南省）
//   - industryCode：行业代码，2 位
//
// 内部会对输入做 clamp 限幅，确保不会溢出区段长度。
func NewGenerator(siteCode, industryCode uint32) *Generator {
	return &Generator{
		siteCode: clamp(siteCode, 0, 99_999_999),
		indCode:  clamp(industryCode, 0, 99),
		typeCode: 11, // 11 = 采集设备
	}
}

func clamp(v, lo, hi uint32) uint32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// DeviceID 签发一个新的、符合规范的 20 位 DeviceID，布局如下：
//
//	SSSSSSSS + II + TT + NN + SSSSSS
//	区划码   + 行业 + 类型 + 网络 + 序号
//
// GA/T 1400 / GB/T 28181 风格布局：分别占 8 / 2 / 2 / 2 / 6 位，总计 20 位。
// 序号部分在 0..999999 之间循环自增，溢出后回绕。
func (g *Generator) DeviceID() string {
	g.mu.Lock()
	g.seq = (g.seq + 1) % 1_000_000
	seq := g.seq
	g.mu.Unlock()

	return fmt.Sprintf("%08d%02d%02d%02d%06d",
		g.siteCode, g.indCode%100, g.typeCode%100, 01, seq)
}

// UUID 返回无连字符的小写 UUID 字符串。
//
// 协议规定 NotificationId、SubImageInfo 等字段使用无连字符 UUID（32 位 hex）。
// 内部直接复用 google/uuid 实现，符合 RFC 4122 v4。
func (g *Generator) UUID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// Nonce 返回 32 位十六进制随机字符串，用于 HTTP Digest 认证的服务端随机数。
//
// 实现使用 crypto/rand 作为熵源，保证密码学安全强度。
// 每次调用生成新值，并由 storage 层持久化以支撑重放保护。
func (g *Generator) Nonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NonceKey Nonce 持久化层的结构化主键：值 + 签发时间。
type NonceKey struct {
	Value    string    // Nonce 字符串
	IssuedAt time.Time // 服务端签发时间，用于过期清理
}

// SubscribeID 返回 12 位大写字母数字的订阅 ID，作为 Subscribe / SubscribeNotification 的主键。
//
// 取 UUID 的前 12 位并转大写，碰撞概率在单进程规模下可忽略。
func (g *Generator) SubscribeID() string {
	return strings.ToUpper(g.UUID()[:12])
}