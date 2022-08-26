package form

type AccountCreateForm struct {
	Id       uint64
	Username *string `json:"username" binding:"required,min=4,max=16"`
	Password *string `json:"password"`
}

type AccountUpdateForm struct {
	Password *string `json:"password" binding:"min=6,max=16"`
}

type AccountChangePasswordForm struct {
	Username    string `json:"username"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
