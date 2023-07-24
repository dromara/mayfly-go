package form

type AccountCreateForm struct {
	Id       uint64 `json:"id"`
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required,min=4,max=16"`
	Password string `json:"password"`
}

type AccountUpdateForm struct {
	Name     string  `json:"name" binding:"max=16"` // 姓名
	Username string  `json:"username" binding:"max=20"`
	Password *string `json:"password"`
}

type AccountChangePasswordForm struct {
	Username    string `json:"username"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
