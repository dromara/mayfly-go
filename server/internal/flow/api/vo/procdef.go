package vo

import (
	"mayfly-go/internal/flow/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
)

type Procdef struct {
	tagentity.RelateTags // 标签信息
	entity.Procdef

	MsgTmplId *uint64 `json:"msgTmplId" gorm:"-"` // 消息模板ID
}

func (p *Procdef) GetRelateId() uint64 {
	return p.Id
}
