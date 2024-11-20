package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	// tag
	LogTagSave:   "标签树-保存信息",
	LogTagDelete: "标签树-删除信息",
	LogTagMove:   "标签树-移动标签",

	ErrTagCodeInvalid:         "标识符不能包含'/'",
	ErrNoAdminCreateTag:       "非管理员无法添加根标签",
	ErrNoPermissionCreateTag:  "无权添加该标签",
	ErrTagCodePathLikeExist:   "已存在该标签路径开头的标签, 请修改该标识code",
	ErrNoPermissionOpResource: "您无权操作该资源",
	ErrNoPermissionDeleteTag:  "您无权删除该标签",
	ErrConflictingCodePath:    "存在冲突的编号路径",

	// team
	LogTeamSave:         "团队-保存信息",
	LogTeamDelete:       "团队-删除信息",
	LogTeamAddMember:    "团队-新增成员",
	LogTeamRemoveMember: "团队-移除成员",

	ErrNameExist:    "团队名已存在",
	ErrMemeberExist: "成员已存在",

	// ac
	LogAcShowPwd: "授权凭证-查看密文",
	LogAcSave:    "授权凭证-保存",
	LogAcDelete:  "授权凭证-删除",

	ErrPublicAcRelated:            "该公共授权凭证已被关联",
	ErrResourceNoBindAc:           "资源至少需要绑定一个授权凭证，无法删除该凭证",
	ErrAcNameExist:                "授权凭证名称[{{.acName}}]已存在",
	ErrResourceTagNotExist:        "资源标签[{{.resourceCode}}]不存在, 请检查资源编号是否正确",
	ErrResourceNotExist:           "该授权凭证关联的资源信息不存在, 请检查资源编号",
	ErrPublicAcNotAllowModifyType: "公共授权凭证不允许修改凭证类型",
	ErrPublicAcNotExist:           "公共授权凭证[{{.acName}}]不存在",
}
