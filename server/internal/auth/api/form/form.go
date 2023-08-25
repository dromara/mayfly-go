package form

type LoginForm struct {
	Username string `json:"username" binding:"required"`
	Password string `binding:"required"`
	Captcha  string `json:"captcha"`
	Cid      string `json:"cid"`
}

type OtpVerfiy struct {
	OtpToken string `json:"otpToken" binding:"required"`
	Code     string `json:"code" binding:"required"`
}
