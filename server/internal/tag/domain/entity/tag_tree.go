package entity

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

// 标签树
type TagTree struct {
	model.Model

	Type     TagType `json:"type"`     // 类型： -1.普通标签； 其他值则为对应的资源类型
	Code     string  `json:"code"`     // 标识编码, 若类型不为-1，则为对应资源编码
	CodePath string  `json:"codePath"` // 标识路径，tag1/tag2/tagType1|tagCode/tagType2|yyycode/，非普通标签类型段含有标签类型
	Name     string  `json:"name"`     // 名称
	Remark   string  `json:"remark"`   // 备注说明
}

type TagType int8

const (
	// 标识路径分隔符
	CodePathSeparator = "/"
	// 标签路径资源段分隔符
	CodePathResourceSeparator = "|"

	TagTypeTag     TagType = -1
	TagTypeMachine TagType = TagType(consts.ResourceTypeMachine)
	TagTypeDb      TagType = TagType(consts.ResourceTypeDb) // 数据库实例
	TagTypeRedis   TagType = TagType(consts.ResourceTypeRedis)
	TagTypeMongo   TagType = TagType(consts.ResourceTypeMongo)

	// ----- （单独声明各个资源的授权凭证类型而不统一使用一个授权凭证类型是为了获取登录账号的授权凭证标签(ResourceAuthCertApp.GetAccountAuthCert)时，避免查出所有资源的授权凭证）

	TagTypeMachineAuthCert TagType = 11 // 机器-授权凭证

	TagTypeDbAuthCert TagType = 21 // 数据库-授权凭证
	TagTypeDbName     TagType = 22 // 数据库名
)

func (pt *TagTree) IsRoot() bool {
	// 去除路径两端可能存在的斜杠
	path := strings.Trim(pt.CodePath, "/")
	return len(strings.Split(path, "/")) == 1
}

// GetParentPath 获取父标签路径, 如CodePath = test/test1/test2/  -> index = 0 => test/test1/  index = 1 => test/
func (pt *TagTree) GetParentPath(index int) string {
	return GetParentPath(pt.CodePath, index)
}

// GetTagPath 获取标签段路径，不获取对应资源相关路径
func (pt *TagTree) GetTagPath() string {
	codePath := pt.CodePath

	// 以 资源分隔符"|" 符号对字符串进行分割
	parts := strings.Split(codePath, CodePathResourceSeparator)
	if len(parts) < 2 {
		return codePath
	}

	// 从分割后的第一个子串中提取所需部分
	substringBeforeNumber := parts[0]

	// 找到最后一个 "/" 的位置
	lastSlashIndex := strings.LastIndex(substringBeforeNumber, CodePathSeparator)

	// 如果找到最后一个 "/" 符号，则截取子串
	if lastSlashIndex != -1 {
		return substringBeforeNumber[:lastSlashIndex+1]
	}

	return codePath
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
	CodePath string `json:"codePath" gorm:"-"` // 标签路径
}

func (r *ResourceTag) SetTagInfo(rt ResourceTag) {
	r.CodePath = rt.CodePath
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

// GetCodeByPath 从codePaths中提取指定标签类型的所有tagCode并去重
// 如：codePaths = tag1/tag2/1|xxxcode/11|yyycode/,  tagType = 1 -> xxxcode,  tagType = 11 -> yyycode
func GetCodeByPath(tagType TagType, codePaths ...string) []string {
	var codes []string
	for _, codePath := range codePaths {
		// tag1/tag2/1|xxxcode/11|yyycode，根据 /tagType|resourceCode进行切割
		splStrs := strings.Split(codePath, fmt.Sprintf("%s%d%s", CodePathSeparator, tagType, CodePathResourceSeparator))
		if len(splStrs) < 2 {
			continue
		}

		codes = append(codes, strings.Split(splStrs[1], CodePathSeparator)[0])
	}

	return collx.ArrayDeduplicate[string](codes)
}

// GetParentPath 获取父标签路径, 如CodePath = test/test1/test2/  -> index = 0 => test/test1/  index = 1 => test/
func GetParentPath(codePath string, index int) string {
	// 去除末尾的斜杠
	codePath = strings.TrimSuffix(codePath, CodePathSeparator)

	// 使用 Split 方法将路径按斜杠分割成切片
	paths := strings.Split(codePath, CodePathSeparator)

	// 确保索引在有效范围内
	if index < 0 {
		index = 0
	} else if index > len(paths)-2 {
		index = len(paths) - 2
	}

	// 按索引拼接父标签路径
	parentPath := strings.Join(paths[:len(paths)-index-1], CodePathSeparator)

	return parentPath + CodePathSeparator
}

// GetAllCodePath 根据表情路径获取所有相关的标签codePath
func GetAllCodePath(codePath string) []string {
	// 去除末尾的斜杠
	codePath = strings.TrimSuffix(codePath, CodePathSeparator)

	// 使用 Split 方法将路径按斜杠分割成切片
	paths := strings.Split(codePath, CodePathSeparator)

	var result []string
	var partialPath string
	for _, path := range paths {
		partialPath += path + CodePathSeparator
		result = append(result, partialPath)
	}

	return result
}

// TagPathSection 标签路径段
type TagPathSection struct {
	Type TagType `json:"type"` // 类型： -1.普通标签； 其他值则为对应的资源类型
	Code string  `json:"code"` // 标识编码, 若类型不为-1，则为对应资源编码
}

type TagPathSections []*TagPathSection // 标签段数组

// 转为codePath
func (tps TagPathSections) ToCodePath() string {
	return strings.Join(collx.ArrayMap(tps, func(tp *TagPathSection) string {
		if tp.Type == TagTypeTag {
			return tp.Code
		}
		return fmt.Sprintf("%d%s%s", tp.Type, CodePathResourceSeparator, tp.Code)
	}), CodePathSeparator) + CodePathSeparator
}

// GetTagPathSections 根据标签路径获取路径段落
func GetTagPathSections(codePath string) TagPathSections {
	codePath = strings.TrimSuffix(codePath, CodePathSeparator)
	var sections TagPathSections

	codes := strings.Split(codePath, CodePathSeparator)
	for _, code := range codes {
		typeAndCode := strings.Split(code, CodePathResourceSeparator)
		var tagType TagType
		var tagCode string

		if len(typeAndCode) < 2 {
			tagType = TagTypeTag
			tagCode = typeAndCode[0]
		} else {
			tagType = TagType(cast.ToInt(typeAndCode[0]))
			tagCode = typeAndCode[1]
		}

		sections = append(sections, &TagPathSection{
			Type: tagType,
			Code: tagCode,
		})
	}

	return sections
}
