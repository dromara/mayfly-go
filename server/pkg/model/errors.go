package model

import "errors"

// ErrNotFound 表示按条件未查询到记录。
//
// 基座在查询入口把驱动层的未找到错误翻译成这个哨兵并保留原错误，因此业务判空不必再引用
// ORM 的错误值（此前 infra 与 application 层散落着 errors.Is(err, gorm.ErrRecordNotFound)，
// 等于把 ORM 选型渗进了业务逻辑），只需 errors.Is(err, model.ErrNotFound)
var ErrNotFound = errors.New("record not found")

// IsNotFound 判断错误是否为「未查询到记录」
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
