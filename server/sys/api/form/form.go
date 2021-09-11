package form

// 登录表单
type LoginForm struct {
	Username string `json:"username" binding:"required"`
	Password string `binding:"required"`
	Captcha  string `binding:"required"`
	Cid      string `binding:"required"`
}
