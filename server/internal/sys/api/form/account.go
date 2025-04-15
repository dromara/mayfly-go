package form

import "mayfly-go/pkg/model"

type AccountCreateForm struct {
	model.ExtraData

	Id       uint64 `json:"id"`
	Name     string `json:"name" binding:"required,max=16" msg:"required=name cannot be blank,max=The maximum length of a name cannot exceed 16 characters"`
	Username string `json:"username" binding:"pattern=account_username"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password"`
}

type AccountUpdateForm struct {
	model.ExtraData

	Name     string  `json:"name" binding:"max=16"` // 姓名
	Username string  `json:"username" binding:"omitempty,pattern=account_username"`
	Mobile   string  `json:"mobile"`
	Email    string  `json:"email" binding:"omitempty,email"`
	Password *string `json:"password"`
}

type AccountChangePasswordForm struct {
	Username    string `json:"username"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
