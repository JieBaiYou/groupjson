package groupjson

import (
	"errors"
	"fmt"
)

// 库定义的错误类型
var (
	// ErrNilValue 传入nil值时的错误。
	ErrNilValue = errors.New("groupjson: cannot marshal nil value")

	// ErrInvalidValue 传入无效值类型时的错误。
	ErrInvalidValue = errors.New("groupjson: value is not valid")

	// ErrMaxDepth 超过最大递归深度时的错误。
	ErrMaxDepth = errors.New("groupjson: exceeded maximum recursion depth")

	// ErrInvalidType 传入非结构体类型时的错误。
	ErrInvalidType = errors.New("groupjson: cannot marshal non-struct value")

	// ErrGeneratorFail 代码生成失败时的错误。
	ErrGeneratorFail = errors.New("groupjson: code generation failed")
)

// Error 带位置信息的错误类型，用于精确定位问题。
type Error struct {
	Err  error  // 原始错误
	Path string // 错误位置路径
}

// Error 实现error接口，返回格式化的错误消息。
func (e *Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s at path %s", e.Err.Error(), e.Path)
	}
	return e.Err.Error()
}

// Unwrap 支持errors.Is和errors.As的错误链。
func (e *Error) Unwrap() error {
	return e.Err
}

// WrapError 包装错误并添加路径信息，便于定位问题来源。
func WrapError(err error, path string) *Error {
	if e, ok := err.(*Error); ok {
		if e.Path == "" {
			e.Path = path
		} else if path != "" {
			e.Path = path + "." + e.Path
		}
		return e
	}
	return &Error{Err: err, Path: path}
}
