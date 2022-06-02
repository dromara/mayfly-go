package model

type AppContext struct {
}

type LoginAccount struct {
	Id       uint64
	Username string
}

type Permission struct {
	CheckToken bool   // 是否检查token
	Code       string // 权限码
	Name       string // 描述
}
