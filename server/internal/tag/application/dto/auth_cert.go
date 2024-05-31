package dto

import "mayfly-go/internal/tag/domain/entity"

type RelateAuthCert struct {
	ResourceCode string

	// 资源标签类型
	ResourceType entity.TagType

	// 空数组则为删除该资源绑定的授权凭证
	AuthCerts []*entity.ResourceAuthCert
}
