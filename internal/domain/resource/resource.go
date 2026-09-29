// Package resource 定义 GA/T 1400.4 REST 接口所交换的资源（Resource）聚合。
//
// 该包刻意保持精简：只放所有资源类型共用的字段（ID、时间戳、位置、图片引用等）。
// 各资源的具体 schema（Person / Face / MotorVehicle ...）由 generator adapter
// 以 JSON / XML 形式产出，domain 层把它们当作不透明 payload，
// 从而能够接受第三方平台返回的任意结构。
package resource

import "time"

// Kind 资源类别枚举。值 MUST 保持稳定——它们与 REST URI 段、场景引擎 pacing bucket 一一对应。
type Kind string

const (
	KindPerson          Kind = "Person"          // 人员
	KindFace            Kind = "Face"            // 人脸
	KindMotorVehicle    Kind = "MotorVehicle"    // 机动车
	KindNonMotorVehicle Kind = "NonMotorVehicle" // 非机动车
	KindThing           Kind = "Thing"           // 物品
	KindScene           Kind = "Scene"           // 场景
	KindVideoSlice      Kind = "VideoSlice"      // 视频片段
	KindImage           Kind = "Image"           // 图片
	KindFile            Kind = "File"            // 文件
	KindCase            Kind = "Case"            // 案件
	KindVideoLabel      Kind = "VideoLabel"      // 视频标签
	KindAnalysisRule    Kind = "AnalysisRule"    // 分析规则
)

// AllKinds HTTP 路由器与场景引擎遍历资源类别时使用的有序列表。
// 修改时 MUST 与上方 Kind* 常量同步。
var AllKinds = []Kind{
	KindPerson, KindFace, KindMotorVehicle, KindNonMotorVehicle, KindThing,
	KindScene, KindVideoSlice, KindImage, KindFile, KindCase,
	KindVideoLabel, KindAnalysisRule,
}

// CollectionOf 返回 GA/T 1400.4 协议中该 Kind 对应的复数 URI 段。
//
// 返回空字符串表示"无 URI 映射"——通常对应合成型 Kind。
// 调用方拿到空字符串时应跳过该 Kind，不为其注册路由。
func CollectionOf(k Kind) string {
	switch k {
	case KindPerson:
		return "Persons"
	case KindFace:
		return "Faces"
	case KindMotorVehicle:
		return "MotorVehicles"
	case KindNonMotorVehicle:
		return "NonMotorVehicles"
	case KindThing:
		return "Things"
	case KindScene:
		return "Scenes"
	case KindVideoSlice:
		return "VideoSlices"
	case KindImage:
		return "Images"
	case KindFile:
		return "Files"
	case KindCase:
		return "Cases"
	case KindVideoLabel:
		return "VideoLabels"
	case KindAnalysisRule:
		return "AnalysisRules"
	}
	return ""
}

// IDOf 返回某 Kind 在 JSON 协议体中主键字段的名称（如 PersonID / FaceID）。
//
// 用于：
//   - HTTP API 生成 ResponseStatus.Id 时取值
//   - 从通用 payload 中抽取主键字段
func IDOf(k Kind) string {
	switch k {
	case KindPerson:
		return "PersonID"
	case KindFace:
		return "FaceID"
	case KindMotorVehicle:
		return "MotorVehicleID"
	case KindNonMotorVehicle:
		return "NonMotorVehicleID"
	case KindThing:
		return "ThingID"
	case KindScene:
		return "SceneID"
	case KindVideoSlice:
		return "VideoSliceID"
	case KindImage:
		return "ImageID"
	case KindFile:
		return "FileID"
	case KindCase:
		return "CaseID"
	case KindVideoLabel:
		return "VideoLabelID"
	case KindAnalysisRule:
		return "AnalysisRuleID"
	}
	return ""
}

// IsValid 报告 k 是否为已识别的 Kind 常量。
// 用于服务层校验外部传入的 Kind 字符串。
func IsValid(k Kind) bool {
	for _, x := range AllKinds {
		if x == k {
			return true
		}
	}
	return false
}

// Envelope Collection POST 响应使用的标准封装。
//
// GA/T 1400.4 协议存在两种变体：裸对象与列表封装。
// 本结构对应列表封装（包 N 个对象），HTTP API 同时接受两种形态。
type Envelope struct {
	Kind      Kind            `json:"-"`
	Items     []any           `json:"-"`
	Metadata  map[string]any  `json:"-"`
	Payload   map[string]any  `json:"-"`
	CreatedAt time.Time       `json:"-"`
}