package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogLabelSave:   "标签 - 保存标签",
	LogLabelDelete: "标签 - 删除标签",

	ErrLabelKeyRequired:     "标签键不能为空",
	ErrLabelValueRequired:   "标签值不能为空",
	ErrLabelDuplicate:       "标签 [{{.key}}:{{.value}}] 已存在",
	ErrLabelInUse:           "标签 [{{.key}}:{{.value}}] 已被 {{.count}} 处资源绑定，无法删除，请先解除绑定",
	ErrLabelKeyValueChanged: "标签键和标签值创建后不可修改，如需更改请删除后重新创建",
}
