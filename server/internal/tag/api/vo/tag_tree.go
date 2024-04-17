package vo

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/utils/collx"
)

type TagTreeVOS []*entity.TagTree

type TagTreeItem struct {
	*entity.TagTree
	Children []*TagTreeItem `json:"children"`
}

func (m *TagTreeVOS) ToTrees(pid uint64) []*TagTreeItem {
	var ttis []*TagTreeItem
	if len(*m) == 0 {
		return ttis
	}

	ttis = collx.ArrayMap(*m, func(tr *entity.TagTree) *TagTreeItem { return &TagTreeItem{TagTree: tr} })
	tagMap := collx.ArrayToMap(ttis, func(item *TagTreeItem) string {
		return item.CodePath
	})

	for _, node := range ttis {
		// 根节点
		if node.IsRoot() {
			continue
		}
		parentCodePath := node.GetParentPath(0)
		parentNode := tagMap[parentCodePath]
		if parentNode != nil {
			parentNode.Children = append(parentNode.Children, node)
		}
	}

	return collx.ArrayFilter(ttis, func(tti *TagTreeItem) bool { return tti.IsRoot() })
}
