package form

import "mayfly-go/internal/tag/domain/entity"

type TagTree struct {
	*entity.TagTree

	Pid uint64 `json:"pid"`
}
