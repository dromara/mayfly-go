package sqldraft

// 依赖方向守卫。
//
// 为什么需要它：db 模块只允许经中立端口包 ai/pkg/textgen 触达 AI 能力。一旦有人
// 顺手 import ai/application 或 ai/agent，编译照样通过（Go 只禁止循环，不禁止
// 「本可绕开的分层引用」），但依赖方向就反了：ai/application/resource 已经依赖
// db/application，这条引用一旦跨过 ai 的内层包，就是一条随时会成环的隐形边。
//
// 静态断言比注释可靠：注释会在下一次改动里过期，测试不会。

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// forbiddenAiImports db 模块禁止引用的 ai 内层包（中立端口包 ai/pkg/* 不在此列）
var forbiddenAiImports = []string{
	"mayfly-go/internal/ai/application",
	"mayfly-go/internal/ai/agent",
	"mayfly-go/internal/ai/api",
	"mayfly-go/internal/ai/config",
	"mayfly-go/internal/ai/session",
	"mayfly-go/internal/ai/tools",
	"mayfly-go/internal/ai/memory",
	"mayfly-go/internal/ai/protocol",
}

func TestDbModuleDoesNotImportAiInternals(t *testing.T) {
	root := filepath.Join("..", "..") // internal/db
	checked := 0
	err := walkGoFiles(root, func(path string) error {
		imports, err := parseImports(path)
		if err != nil {
			return err
		}
		checked++
		for _, imp := range imports {
			for _, forbidden := range forbiddenAiImports {
				if imp == forbidden || strings.HasPrefix(imp, forbidden+"/") {
					t.Errorf("%s imports %q: db 模块只允许经 ai/pkg/textgen 触达 AI 能力", path, imp)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if checked == 0 {
		t.Fatal("no go file scanned: 守卫扫不到文件等于没有守卫")
	}
}

// 中立端口包必须是叶子包：只依赖标准库与 eino，不得反向依赖任何业务模块。
func TestTextgenPortStaysLeaf(t *testing.T) {
	root := filepath.Join("..", "..", "..", "ai", "pkg", "textgen")
	seen := 0
	err := walkGoFiles(root, func(path string) error {
		imports, err := parseImports(path)
		if err != nil {
			return err
		}
		seen++
		for _, imp := range imports {
			if strings.HasPrefix(imp, "mayfly-go/") {
				t.Errorf("%s imports %q: textgen 必须是叶子包，否则依赖倒置失效", path, imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if seen == 0 {
		t.Fatal("textgen 包下没有扫到文件")
	}
}

// walkGoFiles 遍历目录下的 .go 文件（跳过测试数据与隐藏目录）
func walkGoFiles(root string, visit func(path string) error) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// 根路径本身可能是 ".."，不能按「名字以点开头」跳过（那会把根当成隐藏目录）
			if path == root {
				return nil
			}
			name := entry.Name()
			if name == "testdata" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		return visit(path)
	})
}

// parseImports 取一个文件的 import 路径（字符串常量去引号）
func parseImports(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		unquoted, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imports = append(imports, unquoted)
	}
	return imports, nil
}
