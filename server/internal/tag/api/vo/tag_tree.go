package vo

import "time"

type TagTreeVO struct {
	Id         int       `json:"id"`
	Pid        int       `json:"pid"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	CodePath   string    `json:"codePath"`
	Remark     string    `json:"remark"`
	Creator    string    `json:"creator"`
	CreateTime time.Time `json:"createTime"`
	Modifier   string    `json:"modifier"`
	UpdateTime time.Time `json:"updateTime"`
}

type TagTreeVOS []TagTreeVO

type TagTreeItem struct {
	TagTreeVO
	Children []TagTreeItem `json:"children"`
}

func (m *TagTreeVOS) ToTrees(pid int) []TagTreeItem {
	var resourceTree []TagTreeItem

	list := m.findChildren(pid)
	if len(list) == 0 {
		return resourceTree
	}

	for _, v := range list {
		Children := m.ToTrees(int(v.Id))
		resourceTree = append(resourceTree, TagTreeItem{v, Children})
	}

	return resourceTree
}

func (m *TagTreeVOS) findChildren(pid int) []TagTreeVO {
	child := []TagTreeVO{}

	for _, v := range *m {
		if v.Pid == pid {
			child = append(child, v)
		}
	}
	return child
}
