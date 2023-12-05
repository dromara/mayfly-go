package vo

import (
	"mayfly-go/internal/tag/domain/entity"
)

type TagTreeVOS []*entity.TagTree

type TagTreeItem struct {
	*entity.TagTree
	Children []TagTreeItem `json:"children"`
}

func (m *TagTreeVOS) ToTrees(pid uint64) []TagTreeItem {
	var resourceTree []TagTreeItem

	list := m.findChildren(pid)
	if len(list) == 0 {
		return resourceTree
	}

	for _, v := range list {
		Children := m.ToTrees(v.Id)
		resourceTree = append(resourceTree, TagTreeItem{v, Children})
	}

	return resourceTree
}

func (m *TagTreeVOS) findChildren(pid uint64) []*entity.TagTree {
	child := []*entity.TagTree{}

	for _, v := range *m {
		if v.Pid == pid {
			child = append(child, v)
		}
	}
	return child
}
