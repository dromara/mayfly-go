package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	// account
	LogAccountCreate:       "Create an account",
	LogChangePassword:      "Change password",
	LogAccountChangeStatus: "Change account status",
	LogResetOtpSecret:      "Reset otp secret",
	LogAccountDelete:       "Deleting an account",
	LogAssignUserRoles:     "Assigning user roles",

	ErrUsernameExist:                "Username is already exists",
	ErrOldPasswordWrong:             "Old password wrong",
	ErrAccountPasswordNotFollowRule: "Password must be at least 8 characters long and contain upper/lower case + number + special symbols",

	// role
	LogRoleSave:           "Save role",
	LogRoleDelete:         "Deleting role",
	LogAssignRoleResource: "Role association menu permissions",

	// menu resource
	LogResourceSave:         "Save menu permissions",
	LogResourceDelete:       "Deleting menu permissions",
	LogChangeResourceStatus: "Change menu permissions status",
	LogSortResource:         "Sort menu permissions",

	ErrResourceCodeInvalid: "code cannot contain ','",
	ErrResourceCodeExist:   "The code already exists",

	// sysconf
	LogSaveSysConfig: "Save system configuration",
}
