package entity

import (
	"fmt"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

// 标签树
type TagTree struct {
	model.Model

	Type     TagType `json:"type" gorm:"not null;default:-1;comment:类型： -1.普通标签； 1机器  2db 3redis 4mongo"`        // 类型： -1.普通标签； 其他值则为对应的资源类型
	Code     string  `json:"code" gorm:"not null;size:50;index:idx_tag_code;comment:标识符"`                        // 标识编码, 若类型不为-1，则为对应资源编码
	CodePath string  `json:"codePath" gorm:"not null;size:700;index:idx_tag_code_path,length:255;comment:标识符路径"` // 标识路径，tag1/tag2/tagType1|tagCode/tagType2|yyycode/，非普通标签类型段含有标签类型
	Name     string  `json:"name" gorm:"size:50;comment:名称"`                                                     // 名称
	Remark   string  `json:"remark" gorm:"size:255;"`                                                            // 备注说明
}

type TagType int8

const (
	// 标识路径分隔符
	CodePathSeparator = "/"
	// 标签路径资源段分隔符
	CodePathResourceSeparator = "|"

	TagTypeTag        TagType = -1
	TagTypeMachine    TagType = TagType(consts.ResourceTypeMachine)
	TagTypeDbInstance TagType = TagType(consts.ResourceTypeDbInstance) // 数据库实例
	TagTypeEsInstance TagType = TagType(consts.ResourceTypeEsInstance) // es实例
	TagTypeRedis      TagType = TagType(consts.ResourceTypeRedis)
	TagTypeMongo      TagType = TagType(consts.ResourceTypeMongo)
	TagTypeAuthCert   TagType = TagType(consts.ResourceTypeAuthCert) // 授权凭证类型

	TagTypeDb TagType = 22 // 数据库名
)

// 标签接口资源，如果要实现资源结构体填充标签信息，则资源结构体需要实现该接口
type ITagResource interface {
	// 获取资源code
	GetCode() string

	// 赋值标签基本信息
	SetTagInfo(rt ResourceTag)
}

// 资源关联的标签信息
type ResourceTag struct {
	TagId    uint64 `json:"tagId" gorm:"-"`
	CodePath string `json:"codePath" gorm:"-"` // 标签路径
}

func (r *ResourceTag) SetTagInfo(rt ResourceTag) {
	r.CodePath = rt.CodePath
	r.TagId = rt.TagId
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

// CodePath 标签编号路径 如: tag1/tag2/resourceType1|xxxcode/resourceType2|yyycode/
type CodePath string

// GetTag 获取标签段路径，不获取对应资源相关路径
func (codePath CodePath) GetTag() CodePath {
	// 以 资源分隔符"|" 符号对字符串进行分割
	parts := strings.Split(string(codePath), CodePathResourceSeparator)
	if len(parts) < 2 {
		return codePath
	}

	// 从分割后的第一个子串中提取所需部分
	substringBeforeNumber := parts[0]

	// 找到最后一个 "/" 的位置
	lastSlashIndex := strings.LastIndex(substringBeforeNumber, CodePathSeparator)

	// 如果找到最后一个 "/" 符号，则截取子串
	if lastSlashIndex != -1 {
		return CodePath(substringBeforeNumber[:lastSlashIndex+1])
	}

	return codePath
}

// GetParent 获取父标签编号路径, 如CodePath = test/test1/test2/  -> parentLevel = 0 => test/test1/  parentLevel = 1 => test/
func (cp CodePath) GetParent(parentLevel int) CodePath {
	codePath := string(cp)
	// 去除末尾的斜杠
	codePath = strings.TrimSuffix(codePath, CodePathSeparator)

	// 使用 Split 方法将路径按斜杠分割成切片
	paths := strings.Split(codePath, CodePathSeparator)

	// 确保索引在有效范围内
	if parentLevel < 0 {
		parentLevel = 0
	} else if parentLevel > len(paths)-2 {
		parentLevel = len(paths) - 2
	}

	// 按索引拼接父标签路径
	parentPath := strings.Join(paths[:len(paths)-parentLevel-1], CodePathSeparator)

	return CodePath(parentPath + CodePathSeparator)
}

// GetPathSections 根据标签编号路径获取路径段落
func (cp CodePath) GetPathSections() PathSections {
	codePath := string(cp)
	codePath = strings.TrimSuffix(codePath, CodePathSeparator)
	var sections PathSections

	codes := strings.Split(codePath, CodePathSeparator)
	path := ""
	for _, code := range codes {
		path += code + CodePathSeparator

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

		sections = append(sections, &PathSection{
			Type: tagType,
			Code: tagCode,
			Path: path,
		})
	}

	return sections
}

// GetCode 从codePath中提取指定标签类型的code
// 如：codePath = tag1/tag2/1|xxxcode/11|yyycode/,  tagType = 1 -> xxxcode,  tagType = 11 -> yyycode
func (cp CodePath) GetCode(tagType TagType) string {
	type2Section := collx.ArrayToMap(cp.GetPathSections(), func(section *PathSection) TagType {
		return section.Type
	})
	section := type2Section[tagType]
	if section == nil {
		return ""
	}
	return section.Code
}

// GetAllPath 根据codePath获取所有相关的标签codePath，如 test1/test2/ -> test1/  test1/test2/
func (cp CodePath) GetAllPath() []string {
	return collx.ArrayMap(cp.GetPathSections(), func(section *PathSection) string {
		return section.Path
	})
}

// CanAccess 判断该标签路径是否允许访问操作指定标签路径，即是否为指定标签路径的父级路径。cp通常为用户拥有的标签路径
//
//	// cp = tag1/tag2/  codePath = tag1/tag2/test/  -> true
//	// cp = tag1/tag2/  codePath = tag1/ -> false
func (cp CodePath) CanAccess(codePath string) bool {
	return strings.HasPrefix(codePath, string(cp))
}

// PathSection 标签路径段
type PathSection struct {
	Type TagType `json:"type"` // 类型： -1.普通标签； 其他值则为对应的资源类型
	Code string  `json:"code"` // 编码, 若类型不为-1，则为对应资源编码
	Path string  `json:"path"` // 当前路径段对应的完整编号路径
}

type PathSections []*PathSection // 标签段数组

// ToCodePath 转为codePath
func (tps PathSections) ToCodePath() string {
	return strings.Join(collx.ArrayMap(tps, func(tp *PathSection) string {
		if tp.Type == TagTypeTag {
			return tp.Code
		}
		return fmt.Sprintf("%d%s%s", tp.Type, CodePathResourceSeparator, tp.Code)
	}), CodePathSeparator) + CodePathSeparator
}

func (tps PathSections) GetSection(tagType TagType) []*PathSection {
	return collx.ArrayFilter(tps, func(tp *PathSection) bool {
		return tp.Type == tagType
	})
}

// GetCodesByCodePaths 从codePaths中提取指定标签类型的所有tagCode并去重
// 如：codePaths = tag1/tag2/1|xxxcode/11|yyycode/,  tagType = 1 -> xxxcode,  tagType = 11 -> yyycode
func GetCodesByCodePaths(tagType TagType, codePaths ...string) []string {
	var codes []string
	for _, codePath := range codePaths {
		code := CodePath(codePath).GetCode(tagType)
		if code == "" {
			continue
		}
		codes = append(codes, code)
	}

	return collx.ArrayDeduplicate[string](codes)
}
