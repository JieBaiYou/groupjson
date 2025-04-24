package groupjson

import (
	"errors"
	"fmt"
	"strings"
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

	// ErrCircularReference 检测到循环引用时的错误。
	ErrCircularReference = errors.New("groupjson: circular reference detected")

	// ErrUnsupportedType 不支持的类型错误。
	ErrUnsupportedType = errors.New("groupjson: unsupported type for serialization")

	// ErrNonStringMapKey Map键不是字符串类型的错误。
	ErrNonStringMapKey = errors.New("groupjson: map key is not string type")
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
		// 已经是包装过的错误
		if e.Path == "" {
			// 原错误没有路径，直接使用当前路径
			e.Path = path
		} else if path != "" {
			// 防止路径重复
			if e.Path == path || strings.HasSuffix(e.Path, "."+path) || strings.HasPrefix(e.Path, path+".") {
				// 路径已经包含在内，不需要添加
			} else {
				// 前置当前路径
				e.Path = path + "." + e.Path
			}
		}
		return e
	}
	return &Error{Err: err, Path: path}
}
