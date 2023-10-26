package biz

import (
	"testing"
)

func TestErrIsNil(t *testing.T) {
	// ErrIsNil(NewBizErr("xxx is error"))
	// ErrIsNil(NewBizErr("xxx is error"), "格式错误")
	// ErrIsNil(NewBizErr("xxx is error"), "格式错误: %s, %d", "xxx", 12)
}
