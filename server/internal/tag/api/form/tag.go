package form

import "mayfly-go/internal/tag/domain/entity"

type TagTree struct {
	*entity.TagTree

	Pid uint64 `json:"pid"`
}

type MovingTag struct {
	FromPath string `json:"fromPath" binding:"required"`
	ToPath   string `json:"toPath" binding:"required"`
}
