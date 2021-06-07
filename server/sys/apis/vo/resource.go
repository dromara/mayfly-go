package vo

import "time"

type AccountResourceVO struct {
	Id   int     `json:"id"`
	Pid  int     `json:"pid"`
	Name *string `json:"name"`
	Code *string `json:"code"`
	Type int8    `json:"type"`
	Meta string  `json:"meta"`
}

// 账号拥有的资源vo
type AccountResourceVOList []AccountResourceVO

type resourceItem struct {
	AccountResourceVO
	Children []resourceItem `json:"children"`
}

func (m *AccountResourceVOList) ToTrees(pid int) []resourceItem {
	var resourceTree []resourceItem

	list := m.findChildren(pid)
	if len(list) == 0 {
		return resourceTree
	}

	for _, v := range list {
		Children := m.ToTrees(int(v.Id))
		resourceTree = append(resourceTree, resourceItem{v, Children})
	}

	return resourceTree
}

func (m *AccountResourceVOList) findChildren(pid int) []AccountResourceVO {
	child := []AccountResourceVO{}

	for _, v := range *m {
		if v.Pid == pid {
			child = append(child, v)
		}
	}
	return child
}

// 系统管理/资源管理
type ResourceManageVO struct {
	Id         int        `json:"id"`
	Pid        int        `json:"pid"`
	Name       string     `json:"name"`
	Type       int        `json:"type"`
	Status     int        `json:"status"`
	Creator    string     `json:"creator"`
	CreateTime *time.Time `json:"createTime"`
}

type ResourceManageVOList []ResourceManageVO

type resourceManageItem struct {
	ResourceManageVO
	Children []resourceManageItem `json:"children"`
}

func (m *ResourceManageVOList) ToTrees(pid int) []resourceManageItem {
	var resourceTree []resourceManageItem

	list := m.findChildren(pid)
	if len(list) == 0 {
		return resourceTree
	}

	for _, v := range list {
		Children := m.ToTrees(int(v.Id))
		resourceTree = append(resourceTree, resourceManageItem{v, Children})
	}

	return resourceTree
}

func (m *ResourceManageVOList) findChildren(pid int) []ResourceManageVO {
	child := []ResourceManageVO{}

	for _, v := range *m {
		if v.Pid == pid {
			child = append(child, v)
		}
	}
	return child
}
