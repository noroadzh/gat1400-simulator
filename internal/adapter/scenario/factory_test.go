package scenario

import (
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// newTestFactory 创建一个固定 seed 的 Factory，用于黄金样本测试。
func newTestFactory(t *testing.T) *Factory {
	t.Helper()
	gen := ids.NewGenerator(0, 0)
	return NewFactory(gen, 20260501)
}

func TestFactory_BuildAllKindsAreShaped(t *testing.T) {
	f := newTestFactory(t)
	for _, kind := range []resource.Kind{
		resource.KindPerson,
		resource.KindFace,
		resource.KindMotorVehicle,
		resource.KindNonMotorVehicle,
		resource.KindThing,
		resource.KindScene,
		resource.KindVideoSlice,
		resource.KindImage,
		resource.KindFile,
		resource.KindCase,
		resource.KindVideoLabel,
		resource.KindAnalysisRule,
	} {
		t.Run(string(kind), func(t *testing.T) {
			payload := f.Build(kind)
			listKey := string(kind) + "List"
			objKey := string(kind) + "Object"
			if _, ok := payload[listKey]; !ok {
				t.Fatalf("missing envelope key %q", listKey)
			}
			list, ok := payload[listKey].(map[string]any)
			if !ok {
				t.Fatalf("%q is not a map", listKey)
			}
			objs, ok := list[objKey].([]any)
			if !ok || len(objs) != 1 {
				t.Fatalf("%q not a single-item slice", objKey)
			}
		})
	}
}

func TestFactory_BuildIsDeterministicPerSeed(t *testing.T) {
	gen1 := ids.NewGenerator(0, 0)
	gen2 := ids.NewGenerator(0, 0)
	f1 := NewFactory(gen1, 42)
	f2 := NewFactory(gen2, 42)
	p1 := f1.Build(resource.KindPerson)
	p2 := f2.Build(resource.KindPerson)
	p1obj := p1["PersonList"].(map[string]any)["PersonObject"].([]any)[0].(map[string]any)
	p2obj := p2["PersonList"].(map[string]any)["PersonObject"].([]any)[0].(map[string]any)
	if p1obj["Name"] != p2obj["Name"] || p1obj["Sex"] != p2obj["Sex"] || p1obj["Nation"] != p2obj["Nation"] {
		t.Fatalf("non-deterministic attributes: %#v vs %#v", p1obj, p2obj)
	}
}

func TestFactory_TimeStampFormat(t *testing.T) {
	f := newTestFactory(t)
	p := f.Build(resource.KindPerson)
	const layout = "20060102150405"
	plist := p["PersonList"].(map[string]any)
	pobj := plist["PersonObject"].([]any)[0].(map[string]any)
	got, ok := pobj["ShotTime"].(string)
	if !ok || len(got) < len(layout) {
		t.Fatalf("ShotTime missing or wrong type: %#v", pobj["ShotTime"])
	}
	if _, err := time.Parse(layout, got[:14]); err != nil {
		t.Fatalf("ShotTime prefix %q not in %s: %v", got[:14], layout, err)
	}
}