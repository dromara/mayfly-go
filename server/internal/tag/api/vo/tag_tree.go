package vo

import (
	"mayfly-go/internal/tag/application/dto"
	"strings"
)

type TagTreeVOS []*dto.SimpleTagTree

type TagTreeItem struct {
	*dto.SimpleTagTree
	Children []*TagTreeItem `json:"children"`
	NamePath string         `json:"namePath"`
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
		// 建立父子关系
		if !node.Root {
			parentCodePath := node.GetParentPath()
			if parentNode := tagMap[parentCodePath]; parentNode != nil {
				parentNode.Children = append(parentNode.Children, node)
			}
		}
		// 构建名称路径（由根到当前节点，用 / 分隔，末尾带 /，与 codePath 结构对齐）
		var names []string
		cur := node
		for {
			names = append(names, cur.Name)
			if cur.Root {
				break
			}
			parent := tagMap[cur.GetParentPath()]
			if parent == nil {
				break
			}
			cur = parent
		}
		for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
			names[i], names[j] = names[j], names[i]
		}
		node.NamePath = strings.Join(names, "/") + "/"
	}

	return roots
}
