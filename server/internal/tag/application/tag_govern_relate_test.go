package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGovernLookupIgnoresOperatorVisibility 治理关联查询不得按操作者的标签可见性收窄。
//
// 治理回答的是「这个资源被怎么管」，是资源属性；若按谁在操作来过滤标签路径，
// 当资源同时挂在操作者看不到的标签下、而策略恰好打在那个标签上时，这次操作就静默不受治理。
// 这里用源码结构守：治理入口既不取登录账号，也不调可见性过滤
func TestGovernLookupIgnoresOperatorVisibility(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(".", "tag_tree_relate.go"))
	require.NoError(t, err)
	source := string(content)

	start := strings.Index(source, "func (tr *tagTreeRelateAppImpl) GetGovernRelateIds")
	require.Greater(t, start, -1, "找不到治理关联查询实现")
	body := source[start : strings.Index(source[start:], "\n}")+start]

	require.NotContains(t, body, "GetLoginAccount", "治理查询不应关心谁在操作")
	require.NotContains(t, body, "filterCodePaths", "治理查询不应按标签可见性收窄")
	require.NotContains(t, body, "AdminId", "不应为超管开特例：非超管同样要看到全部治理配置")
	require.Contains(t, body, "GetAllPath()", "祖先标签路径仍要展开，否则打在分组上的策略匹配不到")
}

// TestNoStaleRelateApiLeft 旧的按可见性过滤的关联查询不能留成第二入口：
// 留着就会有人在新调用点顺手选错，把治理查询写成受可见性影响的那条
func TestNoStaleRelateApiLeft(t *testing.T) {
	for _, file := range []string{"tag_tree_relate.go", "../../flow/application/procdef.go", "../../machine/application/machine_cmd_conf.go"} {
		content, err := os.ReadFile(filepath.Clean(file))
		require.NoError(t, err)
		require.NotContains(t, string(content), "GetRelateIds(", filepath.Clean(file)+" 仍引用已废弃的关联查询入口")
	}
}
