package groupjson

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	// 不能序列化nil值
	ErrNilValue = errors.New("groupjson: cannot marshal nil value")
	// 无效值, 必须是结构体或指向结构体的指针
	ErrInvalidValue = errors.New("groupjson: invalid value, must be struct or pointer to struct")
	// 超出最大递归深度
	ErrMaxDepth = errors.New("groupjson: exceeded maximum recursion depth")
	// 不能序列化此类型
	ErrInvalidType = errors.New("groupjson: cannot marshal this type")
	// 代码生成失败
	ErrGeneratorFail = errors.New("groupjson: code generation failed")
)

// Error 封装了GroupJSON库的错误
type Error struct {
	Err  error
	Path string
}

// Error 实现error接口
func (e *Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s at path %s", e.Err.Error(), e.Path)
	}
	return e.Err.Error()
}

// Unwrap 实现错误链
func (e *Error) Unwrap() error {
	return e.Err
}

// WrapError 封装一个错误并添加路径信息
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
