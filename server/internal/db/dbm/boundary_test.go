package dbm_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
)

// allowedInternalDepPrefixes 允许内核依赖的内部包前缀：
//   - mayfly-go/internal/pkg/：内部通用工具库（无业务语义，等同顶层 pkg）
//   - mayfly-go/internal/db/dbm/：内核自身
//
// 除此之外的一切 mayfly-go/internal/... 均属业务模块（machine、flow 等）或 db 业务分层
// （api/application/domain/infra/imsg），内核一律不得依赖。
var allowedInternalDepPrefixes = []string{
	"mayfly-go/internal/pkg/",
	"mayfly-go/internal/db/dbm/",
}

// TestDbmKernelImportBoundary 守护数据库内核（dbm）的依赖纯净性：
// 内核只能依赖标准库、第三方库、通用库（pkg、internal/pkg）与 dbm 自身，
// 禁止依赖任何业务模块（如 machine）或 db 业务分层（api/application/domain/infra/imsg）。
//
// 这是一道回归护栏——内核与业务模块解耦后，若后续迭代重新引入跨模块依赖，
// 此测试立即失败，从而把「内核可独立复用、可脱离业务单测」这一架构不变量固化进 CI，
// 而非依赖人工 review 的临时记忆。
func TestDbmKernelImportBoundary(t *testing.T) {
	root, err := os.Getwd() // 测试运行目录即 dbm 包目录，向下覆盖全部子包
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	var violations []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 只审查内核的生产代码；测试文件允许为构造用例而依赖任意包
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range file.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(p, "mayfly-go/internal/") {
				continue // 标准库、第三方、顶层 pkg 不在禁止之列
			}
			if hasAnyPrefix(p, allowedInternalDepPrefixes) {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			violations = append(violations, filepath.ToSlash(rel)+" -> "+p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk dbm: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("dbm kernel must stay free of business-module dependencies; found illegal internal imports:\n%s",
			strings.Join(violations, "\n"))
	}
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// TestCapabilityDeclarationMatchesImplementation 能力「声明⟺实现」一致性护栏：
// 方言若声明某扩展对象能力，其 MetadataProvider 必须真实实现对应能力，防止谎报导致前端渲染出
// 树节点却空返回/panic。校验精确到 kind 级：数据驱动自 dbi.ObjectKindFeatures——
// 声明 FeatViews/FeatSequences/... 的对象能力，其 MetadataNavigator.SupportedKinds() 必须含对应 kind；
// 声明 FeatTableRelations 则必须实现 ForeignKeyProvider。新增对象类别只需在 ObjectKindFeatures 加一行。
func TestCapabilityDeclarationMatchesImplementation(t *testing.T) {
	for _, dt := range dbi.GetRegisteredDbTypes() {
		backend := dbi.GetBackend(dt)
		if backend == nil {
			continue
		}
		caps := backend.GetCapabilities()

		declaresRelation := caps.Has(dbi.FeatTableRelations)
		declaresKind := false
		for f := range dbi.ObjectKindFeatures {
			if caps.Has(f) {
				declaresKind = true
				break
			}
		}
		if !declaresRelation && !declaresKind {
			continue // 未声明对象能力的方言无需构造 provider 佐证
		}

		provider := safeProvider(backend, dt)
		if provider == nil {
			t.Errorf("dialect %q declares object capabilities but its MetadataProvider cannot be constructed", dt)
			continue
		}
		if declaresRelation {
			if _, ok := provider.(dbi.ForeignKeyProvider); !ok {
				t.Errorf("dialect %q declares %q but its provider does not implement dbi.ForeignKeyProvider", dt, "relation")
			}
		}
		if declaresKind {
			nav, ok := provider.(dbi.MetadataNavigator)
			if !ok {
				t.Errorf("dialect %q declares object capabilities but its provider does not implement dbi.MetadataNavigator", dt)
				continue
			}
			supported := nav.SupportedKinds()
			for f, kind := range dbi.ObjectKindFeatures {
				if caps.Has(f) && !slices.Contains(supported, kind) {
					t.Errorf("dialect %q declares %q but MetadataNavigator.SupportedKinds() lacks kind %q", dt, f, kind)
				}
			}
		}
	}
}

// safeProvider 以伪连接构造 provider 引用（仅取对象、不执行任何查询）；构造期 panic 视为不合规，返回 nil。
func safeProvider(backend dbi.DbBackend, dt dbi.DbType) (provider dbi.MetadataProvider) {
	defer func() {
		if recover() != nil {
			provider = nil
		}
	}()
	return backend.GetMetadataProvider(&dbi.DbInfo{Type: dt, Backend: backend})
}

// TestEveryDialectDeclaresNamespace 护栏：每个已注册方言都必须声明非零的命名空间层次。
//
// NewAllCapabilities/NewCapabilities 只注入能力集合、不设置 NamespaceHierarchy，方言 override
// GetCapabilities 时若忘记 WithNamespace，会静默退化为「无 database 无 schema」的单层模型，
// 并被 /capabilities 协商端点原样暴露给前端。此断言把该不变量固化，杜绝此类回归逃逸。
func TestEveryDialectDeclaresNamespace(t *testing.T) {
	for _, dt := range dbi.GetRegisteredDbTypes() {
		backend := dbi.GetBackend(dt)
		if backend == nil {
			continue
		}
		ns := backend.GetCapabilities().NamespaceHierarchy
		if !ns.HasDatabase && !ns.HasSchema && !ns.HasCatalog {
			t.Errorf("dialect %q declares empty NamespaceHierarchy (GetCapabilities override missing WithNamespace?)", dt)
		}
	}
}
