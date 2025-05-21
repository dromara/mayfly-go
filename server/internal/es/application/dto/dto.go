package dto

import (
	"mayfly-go/internal/es/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
)

type SaveEsInstance struct {
	EsInstance   *entity.EsInstance
	AuthCerts    []*tagentity.ResourceAuthCert
	TagCodePaths []string
}
