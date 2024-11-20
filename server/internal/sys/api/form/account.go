package form

type AccountCreateForm struct {
	Id       uint64 `json:"id"`
	Name     string `json:"name" binding:"required,max=16" msg:"required=name cannot be blank,max=The maximum length of a name cannot exceed 16 characters"`
	Username string `json:"username" binding:"pattern=account_username"`
	Password string `json:"password"`
}

type AccountUpdateForm struct {
	Name     string  `json:"name" binding:"max=16"` // 姓名
	Username string  `json:"username" binding:"omitempty,pattern=account_username"`
	Password *string `json:"password"`
}

type AccountChangePasswordForm struct {
	Username    string `json:"username"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
