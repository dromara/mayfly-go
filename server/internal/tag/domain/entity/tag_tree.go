package entity

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/pkg/model"
	"strings"
)

// 标签树
type TagTree struct {
	model.Model

	Pid      uint64  `json:"pid"`
	Type     TagType `json:"type"`     // 类型： -1.普通标签； 其他值则为对应的资源类型
	Code     string  `json:"code"`     // 标识编码, 若类型不为-1，则为对应资源编码
	CodePath string  `json:"codePath"` // 标识路径
	Name     string  `json:"name"`     // 名称
	Remark   string  `json:"remark"`   // 备注说明
}

type TagType int8

const (
	// 标识路径分隔符
	CodePathSeparator = "/"

	TagTypeTag     TagType = -1
	TagTypeMachine TagType = TagType(consts.TagResourceTypeMachine)
	TagTypeDb      TagType = TagType(consts.TagResourceTypeDb)
	TagTypeRedis   TagType = TagType(consts.TagResourceTypeRedis)
	TagTypeMongo   TagType = TagType(consts.TagResourceTypeMongo)

	// ----- （单独声明各个资源的授权凭证类型而不统一使用一个授权凭证类型是为了获取登录账号的授权凭证标签(ResourceAuthCertApp.GetAccountAuthCert)时，避免查出所有资源的授权凭证）

	TagTypeMachineAuthCert TagType = 11 // 机器-授权凭证
	TagTypeDbAuthCert      TagType = 21 // DB-授权凭证
)

// GetRootCode 获取根路径信息
func (pt *TagTree) GetRootCode() string {
	return strings.Split(pt.CodePath, CodePathSeparator)[0]
}

// GetParentPath 获取父标签路径, 如CodePath = test/test1/test2/  -> index = 0 => test/test1/  index = 1 => test/
func (pt *TagTree) GetParentPath(index int) string {
	// 去除末尾的斜杠
	codePath := strings.TrimSuffix(pt.CodePath, "/")

	// 使用 Split 方法将路径按斜杠分割成切片
	paths := strings.Split(codePath, "/")

	// 确保索引在有效范围内
	if index < 0 {
		index = 0
	} else if index > len(paths)-2 {
		index = len(paths) - 2
	}

	// 按索引拼接父标签路径
	parentPath := strings.Join(paths[:len(paths)-index-1], "/")

	return parentPath + "/"
}

// 标签接口资源，如果要实现资源结构体填充标签信息，则资源结构体需要实现该接口
type ITagResource interface {
	// 获取资源code
	GetCode() string

	// 赋值标签基本信息
	SetTagInfo(rt ResourceTag)
}

// 资源关联的标签信息
type ResourceTag struct {
	TagId   uint64 `json:"tagId" gorm:"-"`
	TagPath string `json:"tagPath" gorm:"-"` // 标签路径
}

func (r *ResourceTag) SetTagInfo(rt ResourceTag) {
	r.TagId = rt.TagId
	r.TagPath = rt.TagPath
}

// 资源标签列表
type ResourceTags struct {
	Tags []ResourceTag `json:"tags" gorm:"-"`
}

func (r *ResourceTags) SetTagInfo(rt ResourceTag) {
	if r.Tags == nil {
		r.Tags = make([]ResourceTag, 0)
	}
	r.Tags = append(r.Tags, rt)
}
