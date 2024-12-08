package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	// account
	LogAccountCreate:       "创建账号",
	LogChangePassword:      "修改密码",
	LogAccountChangeStatus: "修改账号状态",
	LogResetOtpSecret:      "重置OTP密钥",
	LogAccountDelete:       "删除账号",
	LogAssignUserRoles:     "关联用户角色",

	ErrUsernameExist:                "用户名已存在",
	ErrOldPasswordWrong:             "旧密码错误",
	ErrAccountPasswordNotFollowRule: "密码强度必须8个字符以上且包含字⺟⼤⼩写+数字+特殊符号",

	// role
	LogRoleSave:           "保存角色",
	LogRoleDelete:         "删除角色",
	LogAssignRoleResource: "角色关联菜单权限",

	// menu resource
	LogResourceSave:         "保存菜单权限",
	LogResourceDelete:       "删除菜单权限",
	LogChangeResourceStatus: "修改菜单权限状态",
	LogSortResource:         "菜单权限排序",

	ErrResourceCodeInvalid: "code不能包含','",
	ErrResourceCodeExist:   "该编号已存在",

	// sysconf
	LogSaveSysConfig: "保存系统配置",
}
