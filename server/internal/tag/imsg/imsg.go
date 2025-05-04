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
	// tag
	LogTagSave = iota + consts.ImsgNumTag
	LogTagDelete
	LogTagMove

	ErrTagCodeInvalid
	ErrNoAdminCreateTag
	ErrNoPermissionCreateTag
	ErrTagCodePathLikeExist
	ErrNoPermissionOpResource
	ErrNoPermissionDeleteTag
	ErrConflictingCodePath

	// team
	LogTeamSave
	LogTeamDelete
	LogTeamAddMember
	LogTeamRemoveMember

	ErrNameExist
	ErrMemeberExist

	// ac
	LogAcShowPwd
	LogAcSave
	LogAcDelete

	ErrPublicAcRelated
	ErrResourceNoBindAc
	ErrAcNameExist
	ErrResourceTagNotExist
	ErrResourceNotExist
	ErrPublicAcNotAllowModifyType
	ErrPublicAcNotExist
)
