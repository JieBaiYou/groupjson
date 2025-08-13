package groupjson

import (
	"errors"
	"fmt"
)

// 错误常量
var (
	ErrNilValue          = errors.New("groupjson: cannot marshal nil value")
	ErrInvalidType       = errors.New("groupjson: cannot marshal non-struct value")
	ErrMaxDepth          = errors.New("groupjson: exceeded maximum recursion depth")
	ErrCircularReference = errors.New("groupjson: circular reference detected")
	ErrUnsupportedType   = errors.New("groupjson: unsupported type for serialization")
	ErrNonStringMapKey   = errors.New("groupjson: map key is not string type")
)

// Error 带路径的错误，用于在错误信息中携带精确位置
type Error struct {
	// Err 原始错误
	Err error
	// Path 发生错误的字段/索引路径，形如 a.b[3]["k"]
	Path string
}

func (e *Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s at path %s", e.Err.Error(), e.Path)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// WrapError 将错误与路径绑定；不做去重与合并，调用方应维护正确路径。
func WrapError(err error, path string) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		if e.Path == "" && path != "" {
			e.Path = path
		}
		return e
	}
	return &Error{Err: err, Path: path}
}
