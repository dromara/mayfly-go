package dto

import "mayfly-go/internal/docker/domain/entity"

type SaveContainer struct {
	Container    *entity.Container
	TagCodePaths []string
}
