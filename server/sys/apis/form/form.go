package form

// 登录表单
type LoginForm struct {
	Username string `valid:"Required"`
	Password string `valid:"Required"`
}
