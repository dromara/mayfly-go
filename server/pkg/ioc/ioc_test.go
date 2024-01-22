package ioc

import (
	"testing"
)

type UserApp struct {
	UserRepo *UserRepo `inject:""` // inject=""则默认使用字段名作为组件名进行查找并注入

	sysRepo *SysRepo
}

// 通过Inject开头函数注入，组件名为去除Inject字符串后的其余字符串，即SysRepo
func (u *UserApp) InjectSysRepo(sr *SysRepo) {
	u.sysRepo = sr
}

type UserRepo struct {
	Name string `inject:"username"`
}

type SysRepo struct {
	Name string `inject:"sysname"`
}

func TestInject(t *testing.T) {
	Register("哈哈哈", WithComponentName("username"))
	Register("呵呵呵", WithComponentName("sysname"))

	userRepo := &UserRepo{}
	Register(userRepo)

	sysRepo := new(SysRepo)
	Register(sysRepo)

	userApp := new(UserApp)
	Register(userApp)

	if err := InjectComponents(); err != nil {
		println(err.Error())
	}
	println(userApp)
}

func TestInjectWithCname(t *testing.T) {
	Register("哈哈哈", WithComponentName("username"))
	Register("呵呵呵", WithComponentName("sysname"))

	userRepo := &UserRepo{}
	Register(userRepo, WithComponentName("UserRepo"))

	userApp := new(UserApp)
	Register(userApp)

	sysRepo := new(SysRepo)
	Register(sysRepo, WithComponentName("SysRepo"))

	if err := InjectComponents(); err != nil {
		println(err.Error())
	}
	println(userApp)
}
