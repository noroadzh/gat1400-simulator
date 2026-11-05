// Package generator 生成场景推送用的伪造资源对象。
//
// FakePerson / FakeFace 等工厂产出符合 GA/T 1400.4 主键约定的 map，
// 可选嵌入 media 占位 JPEG（Base64）。
package generator

import (
	"fmt"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/media"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// Options 控制生成细节。
type Options struct {
	DeviceID   string
	Seq        int
	WithImage  bool
	ShotTime   time.Time
}

// Fake 按 Kind 生成一条资源对象；未知 Kind 返回含通用 ID 的 map。
func Fake(kind resource.Kind, opt Options) map[string]any {
	if opt.ShotTime.IsZero() {
		opt.ShotTime = time.Now().UTC()
	}
	idField := resource.IDOf(kind)
	id := fmt.Sprintf("%s-%06d", kind, opt.Seq)
	obj := map[string]any{
		idField:    id,
		"DeviceID": opt.DeviceID,
		"ShotTime": opt.ShotTime.Format(time.RFC3339),
	}
	switch kind {
	case resource.KindPerson:
		obj["Name"] = fmt.Sprintf("Person-%d", opt.Seq)
		obj["Gender"] = "0"
	case resource.KindFace:
		obj["FaceAppearTime"] = opt.ShotTime.Format(time.RFC3339)
	case resource.KindMotorVehicle:
		obj["PlateNo"] = fmt.Sprintf("京A%05d", opt.Seq%100000)
	}
	if opt.WithImage {
		obj["SubImageList"] = map[string]any{
			"SubImageInfoObject": []map[string]any{{
				"Type":    "14",
				"Format":  "Jpeg",
				"Data":    media.PlaceholderJPEGBase64(),
				"FileFormat": "Jpeg",
			}},
		}
	}
	return obj
}
