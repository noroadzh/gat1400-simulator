// Package response 定义 GA/T 1400.4 每个响应消息必须携带的标准 ResponseStatus 信封。
//
// 编码由 HTTP 适配器负责，domain 层保持传输无关性，
// 这样同一份 ResponseStatus 既能走 JSON 也能走 XML。
package response

import "time"

// StatusCode 标准化的响应状态码（与 GB/T 28181 StatusCode 体系类似）。
//
// 这里只枚举协议常用的子集；未知码仍可通过 int 字段序列化，避免被默默丢弃。
type StatusCode int

const (
	CodeOK              StatusCode = 0 // 成功
	CodeInvalid         StatusCode = 1 // 请求参数无效
	CodeInvalidDeviceID StatusCode = 2 // DeviceID 非法
	CodeInvalidTime     StatusCode = 3 // 时间戳非法
	CodeNotFound        StatusCode = 4 // 资源不存在
	CodeFailure         StatusCode = 5 // 服务器处理失败
	CodeUnauthorized    StatusCode = 6 // 未通过认证
	CodeConflict        StatusCode = 7 // 资源冲突（如重复注册）
)

// ResponseStatus 标准外壳。Id 可选，指向本次请求新建/修改的实体。
type ResponseStatus struct {
	StatusCode  StatusCode `json:"StatusCode"`
	StatusString string     `json:"StatusString"`
	Id          string     `json:"Id,omitempty"`
	Time        time.Time  `json:"Time"`
}

// OK 返回一个成功状态。id 可为空字符串。
func OK(id string) ResponseStatus {
	return ResponseStatus{
		StatusCode:  CodeOK,
		StatusString: "OK",
		Id:          id,
		Time:        time.Now().UTC(),
	}
}

// Error 返回一个失败状态，含码、ID。
//
// reason 为可读说明，会被原样写入 StatusString；调用方负责做长度截断等防御。
func Error(id string, code StatusCode, reason string) ResponseStatus {
	return ResponseStatus{
		StatusCode:  code,
		StatusString: reason,
		Id:          id,
		Time:        time.Now().UTC(),
	}
}