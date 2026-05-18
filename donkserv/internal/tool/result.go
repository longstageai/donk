package tool

import (
	"fmt"
	"time"
)

// Result 工具执行结果
// 包含执行状态、数据和错误信息
type Result struct {
	Success       bool           // 是否成功
	Data          any            // 返回数据
	Error         *ToolError     // 错误信息
	ExecutionTime time.Duration  // 执行时间
	Metadata      map[string]any // 额外的元数据信息
}

// NewResult 创建成功结果
func NewResult(data any) *Result {
	return &Result{
		Success:  true,
		Data:     data,
		Metadata: make(map[string]any),
	}
}

// NewErrorResult 创建错误结果
func NewErrorResult(err *ToolError) *Result {
	return &Result{
		Success:  false,
		Error:    err,
		Metadata: make(map[string]any),
	}
}

// NewErrorResultWithMsg 创建带错误消息的结果
func NewErrorResultWithMsg(code, message string, details ...any) *Result {
	err := NewToolError(code, message, details...)
	return &Result{
		Success:  false,
		Error:    err,
		Metadata: make(map[string]any),
	}
}

// SetExecutionTime 设置执行时间
func (r *Result) SetExecutionTime(d time.Duration) {
	r.ExecutionTime = d
}

// SetMetadata 设置元数据
func (r *Result) SetMetadata(key string, value any) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}
	r.Metadata[key] = value
}

// String 将结果转换为字符串
func (r *Result) String() string {
	if r.Error != nil {
		return r.Error.Error()
	}
	if r.Data == nil {
		return ""
	}
	switch v := r.Data.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetData 获取数据并转换为指定类型
func (r *Result) GetData(dst any) error {
	if !r.Success {
		return fmt.Errorf("result is not successful")
	}

	// 简单类型转换
	switch d := dst.(type) {
	case *string:
		*d = r.String()
	case *any:
		*d = r.Data
	default:
		return fmt.Errorf("unsupported type")
	}
	return nil
}
