package wire

import "encoding/json"

// jsonMarshal / jsonUnmarshal 包级可替换的桩：避免在 wire 包内到处重复 encoding/json。
// 同时便于将来切换到 json-iterator、sonic 等更快的编解码器时一行改动即可。
var (
	jsonMarshal   = func(v any) ([]byte, error) { return json.Marshal(v) }
	jsonUnmarshal = func(b []byte, v any) error { return json.Unmarshal(b, v) }
)