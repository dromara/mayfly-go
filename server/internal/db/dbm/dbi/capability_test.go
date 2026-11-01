package dbi

import (
	"testing"

	"mayfly-go/internal/pkg/tlsx"

	"github.com/stretchr/testify/assert"
)

// TestCapabilities_Enum 锁定能力枚举化的关键语义：集合构造/删减的不可变性、字符串入口一致性、零值安全与清单稳定序。
func TestCapabilities_Enum(t *testing.T) {
	all := NewAllCapabilities()
	assert.True(t, all.Has(FeatSchemas))

	// Without 返回副本，不改原集合
	noSchema := all.Without(FeatSchemas)
	assert.False(t, noSchema.Has(FeatSchemas))
	assert.True(t, all.Has(FeatSchemas), "Without 不得修改原能力集合")

	// With 追加扩展对象能力，同样返回副本不改原集合
	withViews := all.With(FeatViews)
	assert.True(t, withViews.Has(FeatViews))
	assert.False(t, all.Has(FeatViews), "With 不得修改原能力集合")

	// WithNamespace 链式补充命名空间层次，不改原值
	ns := all.WithNamespace(NamespaceHierarchy{HasDatabase: true, HasSchema: true})
	assert.True(t, ns.NamespaceHierarchy.HasSchema)
	assert.False(t, all.NamespaceHierarchy.HasSchema, "WithNamespace 不得修改原能力集合")

	// 字符串入口与枚举一致；版本特性名不属于静态能力集合
	assert.True(t, all.HasStaticFeature("schemas"))
	assert.False(t, all.HasStaticFeature("window_functions"))

	// NewCapabilities 白名单语义 + 零值安全
	only := NewCapabilities(FeatComments, FeatDDLExport)
	assert.True(t, only.Has(FeatComments))
	assert.False(t, only.Has(FeatIndexes))
	var zero MetadataCapabilities
	assert.False(t, zero.Has(FeatSchemas))

	// 能力清单升序稳定，便于协商比对
	assert.Equal(t, []string{"comments", "indexes"}, NewCapabilities(FeatIndexes, FeatComments).SupportedFeatures())
}

// TestCapabilities_TLSModes 锁定 TLS 能力声明的两条不变量：
// 声明档位必须同时置上 FeatTLS；未声明的档位一律判不支持（保存期据此拒绝）。
func TestCapabilities_TLSModes(t *testing.T) {
	// 声明前：TLS 能力位与档位都为空
	base := NewCapabilities(FeatComments)
	assert.False(t, base.Has(FeatTLS))
	assert.Empty(t, base.TLSModes)

	withTLS := base.WithTLSModes(tlsx.ModeRequire, tlsx.ModeVerifyFull)
	assert.True(t, withTLS.Has(FeatTLS), "声明档位必须同时置上 FeatTLS")
	assert.True(t, withTLS.HasStaticFeature("tls"))
	assert.True(t, withTLS.SupportsTLSMode(tlsx.ModeRequire))
	assert.False(t, withTLS.SupportsTLSMode(tlsx.ModeVerifyCA), "未声明的档位必须判不支持")
	assert.False(t, base.Has(FeatTLS), "WithTLSModes 不得修改原能力集合")

	// 明文档恒定可用，即使方言没有声明任何加密档位
	assert.True(t, base.SupportsTLSMode(tlsx.ModeDisable))
	assert.True(t, base.SupportsTLSMode(""))

	// 建造链顺序无关：Without/With 必须透传 TLSModes，否则链上一旦删减能力档位就丢
	chained := NewCapabilities(FeatIndexes).WithTLSModes(tlsx.ModeVerifyCA).Without(FeatIndexes).With(FeatComments)
	assert.True(t, chained.SupportsTLSMode(tlsx.ModeVerifyCA))
	assert.True(t, chained.Has(FeatTLS))
	assert.False(t, chained.Has(FeatIndexes))
	assert.True(t, chained.Has(FeatComments))
}
