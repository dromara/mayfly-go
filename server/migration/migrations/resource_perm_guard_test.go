package migrations

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"mayfly-go/migration/migrations/migtest"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// permissionCodePattern 路由上的权限码声明。全仓 89 处声明都是字面量字符串，
// 没有拼接或常量引用，因此静态提取是完备的；将来若出现动态码这里会提取不到，
// 由下方「数量下限」断言兜住
var permissionCodePattern = regexp.MustCompile("RequiredPermissionCode\\(\"([^\"]+)\"\\)")

// scanRoutePermissionCodes 从后端源码里提取所有路由声明的权限码。
// 测试的 cwd 是 migration/migrations，源码根在两层之上；路径全部由本文件常量拼成，无外部输入
func scanRoutePermissionCodes(t *testing.T) []string {
	t.Helper()
	root := filepath.Join("..", "..", "internal")
	require.DirExists(t, root, "未找到后端源码目录，扫描权限码的前提已失效")

	set := map[string]bool{}
	require.NoError(t, filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, m := range permissionCodePattern.FindAllStringSubmatch(string(src), -1) {
			set[m[1]] = true
		}
		return nil
	}))

	codes := make([]string, 0, len(set))
	for c := range set {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

// seededResourceCodes 迁移链跑完后资源表里的全部权限码
func seededResourceCodes(t *testing.T, db *gorm.DB) map[string]bool {
	t.Helper()
	var codes []string
	require.NoError(t, db.Table("t_sys_resource").Where("is_deleted = 0").Pluck("code", &codes).Error)
	set := make(map[string]bool, len(codes))
	for _, c := range codes {
		set[c] = true
	}
	return set
}

// TestAllRoutePermissionCodesAreSeeded 后端路由声明的每个权限码，都必须在跑完迁移链的
// t_sys_resource 里有对应行。
//
// 这条守卫盯的是本项目最容易静默出事的一类缺陷：权限码是 fail-closed 的，路由一旦声明
// RequiredPermissionCode 而资源表没有对应行，除已被授予该码的角色外所有人都会 403，
// 而界面上只表现为「点了没反应」，不报权限错、不留日志，排查成本极高。
//
// 靠人肉核对「路由加了码、迁移也补了行」必然有漏网（本次新增的活动会话两个码就是同批交付的），
// 所以直接从源码提取路由声明，与迁移结果做集合比对：漏一行这里就红
func TestAllRoutePermissionCodesAreSeeded(t *testing.T) {
	codes := scanRoutePermissionCodes(t)
	// 数量下限：扫描口径若因目录改名或正则失配而空转，集合比对会「全部通过」，
	// 因此必须显式钉住确实扫到了东西（当前全仓 89 处声明、去重后 60 余个码）
	require.Greater(t, len(codes), 50, "权限码扫描结果异常偏少，扫描口径可能已失效")

	db := migtest.OpenSqlite(t)
	require.NoError(t, migtest.NewChainMigrator(db, migtest.Concat(All()...)).Migrate())

	seeded := seededResourceCodes(t, db)
	var missing []string
	for _, c := range codes {
		if !seeded[c] {
			missing = append(missing, c)
		}
	}
	require.Empty(t, missing,
		"这些权限码在后端路由上声明了，但迁移没有对应的资源行，相关功能会对所有人 403")
}

// TestSeededResourceCodesAreUnique 同一个权限码只能对应一个资源行。
//
// insertSysResource 是按 id 查重后跳过的：若两个迁移给不同 code 用了同一个 id，后一条会被
// 静默跳过，于是「资源行存在」的表象下 code 其实是别人的。这类冲突在库层面不报错
// （主键各不相同），只能靠集合比对发现
func TestSeededResourceCodesAreUnique(t *testing.T) {
	db := migtest.OpenSqlite(t)
	require.NoError(t, migtest.NewChainMigrator(db, migtest.Concat(All()...)).Migrate())

	var seeded []string
	require.NoError(t, db.Table("t_sys_resource").Where("is_deleted = 0").Pluck("code", &seeded).Error)
	require.NotEmpty(t, seeded, "迁移链跑完不该一条资源都没有")

	seen := map[string]int{}
	for _, c := range seeded {
		seen[c]++
	}
	var dup []string
	for c, n := range seen {
		if n > 1 {
			dup = append(dup, c)
		}
	}
	sort.Strings(dup)
	require.Empty(t, dup, "同一权限码对应了多个资源行，授权与鉴权会指向不确定的那一条")
}
