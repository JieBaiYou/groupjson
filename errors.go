package groupjson

import (
	"errors"
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
