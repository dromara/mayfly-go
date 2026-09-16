package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogLabelSave:   "Label - Save Label",
	LogLabelDelete: "Label - Delete Label",

	ErrLabelKeyRequired:     "label key is required",
	ErrLabelValueRequired:   "label value is required",
	ErrLabelDuplicate:       "label [{{.key}}:{{.value}}] already exists",
	ErrLabelInUse:           "label [{{.key}}:{{.value}}] is bound to {{.count}} resource(s) and cannot be deleted, please unbind first",
	ErrLabelKeyValueChanged: "label key and value cannot be modified after creation, please delete and recreate if changes are needed",
}
