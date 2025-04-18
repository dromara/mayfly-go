package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	// account
	LogAccountCreate = iota + consts.ImsgNumSys
	LogChangePassword
	LogAccountChangeStatus
	LogResetOtpSecret
	LogAccountDelete
	LogAssignUserRoles

	ErrUsernameExist
	ErrOldPasswordWrong
	ErrAccountPasswordNotFollowRule

	// role
	LogRoleSave
	LogRoleDelete
	LogAssignRoleResource

	// menu resource
	LogResourceSave
	LogResourceDelete
	LogChangeResourceStatus
	LogSortResource

	ErrResourceCodeInvalid
	ErrResourceCodeExist

	// sysconf
	LogSaveSysConfig
)
