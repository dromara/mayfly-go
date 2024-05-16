package vo

import (
	"mayfly-go/internal/tag/application/dto"
)

type TagTreeVOS []*dto.SimpleTagTree

type TagTreeItem struct {
	*dto.SimpleTagTree
	Children []*TagTreeItem `json:"children"`
}

func (m *TagTreeVOS) ToTrees(pid uint64) []*TagTreeItem {
	var ttis []*TagTreeItem
	if len(*m) == 0 {
		return ttis
	}

	tagMap := make(map[string]*TagTreeItem)
	var roots []*TagTreeItem
	for _, tag := range *m {
		tti := &TagTreeItem{SimpleTagTree: tag}
		tagMap[tag.CodePath] = tti
		ttis = append(ttis, tti)
		if tti.IsRoot() {
			roots = append(roots, tti)
			tti.Root = true
		}
	}

	for _, node := range ttis {
		// 根节点
		if node.Root {
			continue
		}
		parentCodePath := node.GetParentPath(0)
		parentNode := tagMap[parentCodePath]
		if parentNode != nil {
			parentNode.Children = append(parentNode.Children, node)
		}
	}

	return roots
}
