package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	// tag
	LogTagSave:   "Tag Tree - Save",
	LogTagDelete: "Tag Tree - Delete",
	LogTagMove:   "Tag Tree - Move",

	ErrTagCodeInvalid:         "code cannot contain '/'",
	ErrNoAdminCreateTag:       "Non-administrators cannot add a root tag",
	ErrNoPermissionCreateTag:  "No permission to add the tag",
	ErrTagCodePathLikeExist:   "A tag at the beginning of the tag path already exists, modify the code",
	ErrNoPermissionOpResource: "You do not have permission to manipulate this resource",
	ErrNoPermissionDeleteTag:  "You do not have permission to delete the tag",
	ErrConflictingCodePath:    "There are conflicting code paths",

	// team
	LogTeamSave:         "Team - Save",
	LogTeamDelete:       "Team - Delete",
	LogTeamAddMember:    "Team - Adding new members",
	LogTeamRemoveMember: "Team - Removing members",

	ErrNameExist:    "The team name already exists",
	ErrMemeberExist: "Member already exists",

	// ac
	LogAcShowPwd: "AC - View ciphertext",
	LogAcSave:    "AC - Save",
	LogAcDelete:  "AC - Delete",

	ErrPublicAcRelated:            "The public authorization credential has been associated",
	ErrResourceNoBindAc:           "Resources need to be bound to at least one authorization credential, which cannot be removed",
	ErrAcNameExist:                "Authorization credential name [{{.acName}}] already exists",
	ErrResourceTagNotExist:        "The resource tag [{{.resourcecode}}] does not exist. Check that the resource number is correct",
	ErrResourceNotExist:           "The resource information associated with the authorization credential does not exist, please check the resource code",
	ErrPublicAcNotAllowModifyType: "Public authorization credentials do not allow credential types to be modified",
	ErrPublicAcNotExist:           "Public authorization credentials [{{.acName}}] do not exist",
}
