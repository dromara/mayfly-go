package form

type AccountCreateForm struct {
	Username *string `json:"username" binding:"required,min=4,max=16"`
}

type AccountUpdateForm struct {
	Password *string `json:"password" binding:"required,min=6,max=16"`

	RePassword *string `json:"rePassword" binding:"required,min=4,max=16"`
}
