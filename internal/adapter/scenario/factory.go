package scenario

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// Factory 为任意 resource.Kind 产出符合协议的 payload（以通用 map 形式）。
//
// 分层纪律：factory 不依赖 storage、transport；那都是 engine 的事。
// 同一 (seed, kind) 下 factory 行为确定性——便于黄金样本测试。
type Factory struct {
	idGen *ids.Generator
	rng   *rand.Rand
	mu    sync.Mutex
}

// NewFactory 构造一个 factory，RNG 用 seed 初始化。
// seed=0 时退化为当前 epoch 纳秒（生产用）；测试场景使用固定 seed 以保证 JSON 可复现。
func NewFactory(idGen *ids.Generator, seed int64) *Factory {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Factory{
		idGen: idGen,
		rng:   rand.New(rand.NewSource(seed)),
	}
}

// Build 为指定 Kind 生成一个资源 payload。
//
// payload 形态：
//   - 顶层 <Kind>List{<Kind>Object:[item]}（协议标准信封）
//   - 内层对象使用 CamelCase 字段名（Person / Face / MotorVehicle 等）
//
// 调用方（engine、notifier）可自行从 payload 中抽取 inner 对象。
func (f *Factory) Build(kind resource.Kind) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().UTC().Format("20060115092800XXZ07:00")
	// Format GAT-compatible "YYYYMMDDHHMMSSmmm" using a strict, fixed-width form:
	now = time.Now().UTC().Format("20060102150405.000") // millis
	now = strings.ReplaceAll(strings.ReplaceAll(now, ".", ""), " ", "T")
	if len(now) < 17 {
		now = time.Now().UTC().Format("20060102150405") + "000"
	}
	id := f.idGen.DeviceID()

	base := map[string]any{
		"AgeOfBirth":  0,
		"IsActive":    true,
		"IdentityNo":  "",
		"IdentityType": 0,
		"MantleType":  0,
	}

	switch kind {
	case resource.KindPerson:
		return f.wrapPerson(kind, id, now, base)
	case resource.KindFace:
		return f.wrapFace(kind, id, now)
	case resource.KindMotorVehicle:
		return f.wrapMotorVehicle(kind, id, now)
	case resource.KindNonMotorVehicle:
		return f.wrapNonMotorVehicle(kind, id, now)
	case resource.KindThing:
		return f.wrapThing(kind, id, now)
	case resource.KindScene:
		return f.wrapScene(kind, id, now)
	case resource.KindVideoSlice:
		return f.wrapVideoSlice(kind, id, now)
	case resource.KindImage:
		return f.wrapImage(kind, id, now)
	case resource.KindFile:
		return f.wrapFile(kind, id, now)
	case resource.KindCase:
		return f.wrapCase(kind, id, now)
	case resource.KindVideoLabel:
		return f.wrapVideoLabel(kind, id, now)
	case resource.KindAnalysisRule:
		return f.wrapAnalysisRule(kind, id, now)
	}
	return map[string]any{
		string(kind) + "List": map[string]any{
			string(kind) + "Object": []any{},
		},
	}
}

func (f *Factory) wrap(kind resource.Kind, obj map[string]any) map[string]any {
	listKey := string(kind) + "List"
	objKey := string(kind) + "Object"
	return map[string]any{
		listKey: map[string]any{
			objKey: []any{obj},
		},
	}
}

func (f *Factory) wrapPerson(kind resource.Kind, id, now string, base map[string]any) map[string]any {
	obj := cloneMap(base)
	obj["PersonID"] = id
	obj["Name"] = pick(f.rng, []string{"张三", "李四", "王五", "赵七", "Lucy", "Mark", "Anna"})
	obj["Sex"] = []int{1, 2}[f.rng.Intn(2)]
	obj["Nationality"] = "中国"
	obj["Nation"] = pick(f.rng, []string{"汉", "回", "满", "维"})
	obj["Birthday"] = fmt.Sprintf("%d", 1960+f.rng.Intn(45))
	obj["Province"] = "北京市"
	obj["City"] = "北京市"
	obj["Address"] = "朝阳区某街道"
	obj["Tel"] = fmt.Sprintf("138%08d", f.rng.Intn(100000000))
	obj["SourceID"] = f.idGen.UUID()
	obj["DeviceID"] = id
	obj["ShotTime"] = now
	obj["AppearTime"] = now
	obj["DisappearTime"] = now
	obj["IsVictim"] = 0
	obj["IsSuspect"] = 0
	obj["IsDriver"] = 0
	obj["Description"] = "模拟生成的行人记录"
	return f.wrap(kind, obj)
}

func (f *Factory) wrapFace(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"FaceID":       id,
		"SourceID":     f.idGen.UUID(),
		"DeviceID":     id,
		"ShotTime":     now,
		"AppearTime":   now,
		"DisappearTime": now,
		"Feature":      base64Short(),
		"SubImageList": map[string]any{
			"SubImageObject": []any{
				map[string]any{"ImageID": f.idGen.UUID(), "Data": "https://example.com/sim/face.jpg"},
			},
		},
		"FaceRect":  map[string]any{"Left": 10, "Top": 20, "Right": 60, "Bottom": 70},
		"Score":     0.85,
		"IsVictim":  0,
		"IsSuspect": 0,
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapMotorVehicle(kind resource.Kind, id, now string) map[string]any {
	plate := pick(f.rng, []string{"京A12345", "京B88888", "沪C66666", "粤D99999"})
	obj := map[string]any{
		"MotorVehicleID": id,
		"SourceID":       f.idGen.UUID(),
		"DeviceID":       id,
		"PlateNo":        plate,
		"PlateColor":     "蓝",
		"VehicleColor":   pick(f.rng, []string{"白", "黑", "红", "灰"}),
		"VehicleBrand":   pick(f.rng, []string{"BMW", "Audi", "Toyota", "BYD"}),
		"VehicleType":    "小型汽车",
		"ShotTime":       now,
		"AppearTime":     now,
		"DisappearTime":  now,
		"IsSuspect":      0,
		"Speed":          f.rng.Intn(120),
		"Direction":      pick(f.rng, []string{"东", "西", "南", "北"}),
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapNonMotorVehicle(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"NonMotorVehicleID": id,
		"SourceID":          f.idGen.UUID(),
		"DeviceID":          id,
		"Category":          pick(f.rng, []string{"自行车", "电动车", "摩托车"}),
		"Color":             pick(f.rng, []string{"白", "黑", "红"}),
		"Brand":             "SimBike",
		"ShotTime":          now,
		"AppearTime":        now,
		"DisappearTime":     now,
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapThing(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"ThingID":   id,
		"SourceID":  f.idGen.UUID(),
		"DeviceID":  id,
		"Kind":      pick(f.rng, []string{"包", "手机", "伞"}),
		"Color":     pick(f.rng, []string{"黑", "棕", "红"}),
		"ShotTime":  now,
		"AppearTime": now,
		"DisappearTime": now,
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapScene(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"SceneID":    id,
		"SourceID":   f.idGen.UUID(),
		"DeviceID":   id,
		"SceneType":  pick(f.rng, []string{"广场", "路口", "商场", "车站"}),
		"ShotTime":   now,
		"AppearTime": now,
		"DisappearTime": now,
		"PeopleNum":  f.rng.Intn(50),
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapVideoSlice(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"VideoSliceID": id,
		"SourceID":     f.idGen.UUID(),
		"DeviceID":     id,
		"VideoURL":     "rtsp://sim.example.com/slice",
		"StartTime":    now,
		"EndTime":      now,
		"Duration":     10,
		"SliceSize":    1024,
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapImage(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"ImageID":   id,
		"SourceID":  f.idGen.UUID(),
		"DeviceID":  id,
		"ShotTime":  now,
		"Data":      "https://example.com/sim/image.jpg",
		"FileFormat": "JPEG",
		"FileSize":  4096,
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapFile(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"FileID":   id,
		"SourceID": f.idGen.UUID(),
		"DeviceID": id,
		"FileName": fmt.Sprintf("sim_%d.jpg", f.rng.Intn(99999)),
		"FileFormat": "JPEG",
		"FileSize": 8192,
		"ShotTime": now,
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapCase(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"CaseID":     id,
		"SourceID":   f.idGen.UUID(),
		"DeviceID":   id,
		"CaseName":   "模拟案件",
		"CaseType":   "普通",
		"ShotTime":   now,
		"Description": "由 GAT 1400 模拟器生成",
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapVideoLabel(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"VideoLabelID": id,
		"SourceID":     f.idGen.UUID(),
		"DeviceID":     id,
		"LabelName":    pick(f.rng, []string{"Normal", "Fight", "Fall", "Crowd"}),
		"ShotTime":     now,
		"Confidence":   0.92,
	}
	return f.wrap(kind, obj)
}

func (f *Factory) wrapAnalysisRule(kind resource.Kind, id, now string) map[string]any {
	obj := map[string]any{
		"AnalysisRuleID": id,
		"SourceID":       f.idGen.UUID(),
		"DeviceID":       id,
		"RuleName":       "sim-rule",
		"RuleType":       pick(f.rng, []string{"intrusion", "loitering", "crowd"}),
		"Enabled":        true,
		"Updated":        now,
	}
	return f.wrap(kind, obj)
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func pick(rng *rand.Rand, xs []string) string { return xs[rng.Intn(len(xs))] }

func base64Short() string {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 64)
	for i := range b {
		b[i] = alpha[rng.Intn(len(alpha))]
	}
	return string(b)
}