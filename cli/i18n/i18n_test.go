package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"testing"
)

// declaredMsgIds 解析 keys.go，取出全部声明的消息 id 字面量。
//
// 只靠人肉保证「加了常量也补齐两种语言文案」迟早会漏：漏一种语言时 T() 会静默回退
// 成另一语言或空串，影响命令输出却没有任何编译/运行报错
func declaredMsgIds(t *testing.T) []string {
	t.Helper()

	if _, err := os.Stat("keys.go"); err != nil {
		t.Fatal("keys.go not found in the package dir; this guard would silently pass with zero ids")
	}

	file, err := parser.ParseFile(token.NewFileSet(), "keys.go", nil, 0)
	if err != nil {
		t.Fatalf("parse keys.go: %v", err)
	}

	var ids []string
	ast.Inspect(file, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, v := range vs.Values {
			lit, ok := v.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				continue
			}
			ids = append(ids, s)
		}
		return true
	})
	return ids
}

// 判据自身失效（解析不到键）时下面两条会空跑变绿，所以先证明它确实扫到了足量键
func TestGuardFindsKeys(t *testing.T) {
	if ids := declaredMsgIds(t); len(ids) < 200 {
		t.Fatalf("only %d msg ids parsed from keys.go, the guard is likely broken", len(ids))
	}
}

func TestEveryKeyHasBothLocales(t *testing.T) {
	for _, id := range declaredMsgIds(t) {
		zhText, hasZh := zhCN[id]
		enText, hasEn := en[id]
		if !hasZh {
			t.Errorf("key %q missing in zh_cn.go", id)
			continue
		}
		if !hasEn {
			t.Errorf("key %q missing in en.go", id)
			continue
		}
		if zhText == "" || enText == "" {
			t.Errorf("key %q has an empty translation (zh=%q en=%q)", id, zhText, enText)
		}
	}
}

func TestLocaleKeySetsMatch(t *testing.T) {
	for id := range zhCN {
		if _, ok := en[id]; !ok {
			t.Errorf("key %q exists in zh but not in en", id)
		}
	}
	for id := range en {
		if _, ok := zhCN[id]; !ok {
			t.Errorf("key %q exists in en but not in zh", id)
		}
	}
}
