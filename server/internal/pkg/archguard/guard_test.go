package archguard

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 架构守卫：把「只有人肉审查才会发现」的分层约束变成可执行门禁。
//
// 规则不设例外、不设允许清单：存量违规已全部改掉，因此这里是无条件断言。留一份
// 「已知违规文件列表」等于把债务制度化——它只会在某天被顺手加一行，而加进去的那一行永远不会被还。
// 新增违规直接红；确有需要放宽时必须改本文件并由评审承担后果。

// mustHaveTests 这些包必须存在测试文件：基座一旦没有回归用例，修过的缺陷会静默回流
var mustHaveTests = []string{
	"pkg/base",
	"pkg/gormx",
	"pkg/model",
	"pkg/contextx",
}

func serverRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("unable to locate the module root (go.mod) from the test working directory")
		}
		dir = parent
	}
}

func goFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == "node_modules" || name == ".git" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk go files failed: %v", err)
	}
	return files
}

func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

func codeLines(content string) []string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			lines[i] = ""
		}
	}
	return lines
}

// TestNoGlobalDbOutsideGormx 全局连接只能由 gormx 的会话层读取，starter 是唯一赋值点。
//
// 一旦别处拿 global.Db 读写，就同时绕过了三件事：ctx 上的事务传播（事务内写入落到另一条连接）、
// 审计填充、软删过滤。规则区分两种用法而不是整体放行 starter：赋值一次是必要的，读取永远是逃逸
func TestNoGlobalDbOutsideGormx(t *testing.T) {
	root := serverRoot(t)
	var violations []string

	for _, dir := range []string{"pkg", "internal", "migration"} {
		for _, path := range goFiles(t, filepath.Join(root, dir)) {
			rel := mustRel(root, path)
			if isTestFile(path) {
				continue
			}
			// 会话层读取全局连接是它唯一的职责所在
			if strings.HasPrefix(rel, "pkg/gormx/") {
				continue
			}
			for i, line := range codeLines(readFile(t, path)) {
				if !strings.Contains(line, "global.Db") {
					continue
				}
				// 赋值语句只允许出现在启动装配处
				if strings.Contains(line, "global.Db =") && rel == "pkg/starter/starter.go" {
					continue
				}
				violations = append(violations, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
			}
		}
	}

	assertNoViolations(t, "全局连接不得在 gormx 之外被使用：写库请用 gormx.DB(ctx)，事务经 ctx 传播；global.Db 仅允许 pkg/starter/starter.go 赋值一次", violations)
}

// TestBusinessLayersDontTouchOrm 应用层与接口层不得依赖 ORM，也不得拼裸 SQL。
//
// 出现这种依赖说明基座缺能力：本轮就是这样长出 7 处裸 SQL 与手写表名的。要么补基座入口
// （Patch/Exists/声明式 Join），要么明确记为债务并加进允许清单，不能放任扩散
func TestBusinessLayersDontTouchOrm(t *testing.T) {
	root := serverRoot(t)
	lazyMarkers := []string{".Raw(", "ExecBySQL(", "SelectBySQL(", "global.Db"}
	var violations []string

	for _, dir := range []string{"application", "api"} {
		err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			rel := mustRel(root, path)
			if !strings.Contains(rel, "/"+dir+"/") || !strings.HasSuffix(rel, ".go") || isTestFile(path) {
				return nil
			}
			if imported(path, "gorm.io/gorm") {
				violations = append(violations, rel+": imports gorm.io/gorm")
			}
			for i, line := range codeLines(readFile(t, path)) {
				for _, marker := range lazyMarkers {
					if strings.Contains(line, marker) {
						violations = append(violations, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s failed: %v", dir, err)
		}
	}

	assertNoViolations(t, "application/api 层不得出现 ORM 依赖或裸 SQL，请改用基座入口", violations)
}

// TestNoContextTODOInBusiness 业务层不得用 context.TODO() 写库。
//
// 无上下文的写入既拿不到登录账号（创建人/修改人退化成系统账号），也拿不到 traceId，
// 事后无法归因。请求内请用请求 ctx，请求结束后继续跑的后台动作请用 contextx.Detach
func TestNoContextTODOInBusiness(t *testing.T) {
	root := serverRoot(t)
	var violations []string

	for _, path := range goFiles(t, filepath.Join(root, "internal")) {
		rel := mustRel(root, path)
		if isTestFile(path) {
			continue
		}
		for i, line := range codeLines(readFile(t, path)) {
			if strings.Contains(line, "context.TODO()") {
				violations = append(violations, fmt.Sprintf("%s:%d", rel, i+1))
			}
		}
	}

	assertNoViolations(t, "业务层写库必须带上下文：请求内用请求 ctx，后台动作用 contextx.Detach", violations)
}

// TestCorePackagesHaveTests 基座包必须有测试文件。
//
// 这三层此前一个测试都没有，是本轮十几个静默正确性缺陷能长期存在的直接原因
func TestCorePackagesHaveTests(t *testing.T) {
	root := serverRoot(t)
	var violations []string

	for _, pkg := range mustHaveTests {
		dir := filepath.Join(root, pkg)
		entries, err := os.ReadDir(dir)
		if err != nil {
			violations = append(violations, pkg+": directory missing")
			continue
		}
		found := false
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, pkg+": no *_test.go")
		}
	}

	assertNoViolations(t, "基座包必须有回归用例", violations)
}

// TestNoCtxlessReadsInsideTxClosure 事务闭包内不得调用不收上下文的基座读入口。
//
// 基座读入口目前还不接受 ctx（那是未完成的那部分改造）：事务内的读会去连接池另要一条连接，
// 于是它必然读不到本事务尚未提交的写入 —— 查重、差异比较、引用校验这类判断会按旧数据放行，
// 这是正确性问题而不是性能问题。第二重代价在可用性上：事务占着一条连接不放，并发一高，事务内
// 的读就排队等空闲连接；元数据库默认是本地文件库（SQLite），部署把 max-open-conns 调小之后就
// 是长时间不返回，单连接的极端配置下甚至是永远等不到。两个后果都不报错
// 两个后果都不报错，所以这条约束必须由守卫固定，而不是靠人记住
//
// 判定按读入口的方法名：仓储方法包装了这些读（自身收 ctx 但内部不用）在文本上看不见，因此新增
// 这类包装时，读要么留在事务之前，要么把 ctx 真正接到读上。
//
// 事务闭包内登记的「提交后动作」（AfterCommit / RunAfterCommit）闭包**不算违规**：它在事务提交
// 之后执行，动作 ctx 也已脱事务绑定，此时去连接池读是正确姿势。不放开它会把唯一正确的写法判成错
func TestNoCtxlessReadsInsideTxClosure(t *testing.T) {
	root := serverRoot(t)
	var violations []string

	for _, path := range goFiles(t, filepath.Join(root, "internal")) {
		if isTestFile(path) {
			continue
		}
		src := readFile(t, path)
		rel := mustRel(root, path)
		safe := afterCommitSpans(src)
		for _, span := range txClosureSpans(src) {
			for _, read := range ctxlessReadEntries {
				offset := strings.Index(src[span.bodyStart:span.bodyEnd], read)
				if offset < 0 {
					continue
				}
				at := span.bodyStart + offset
				if coveredByAny(safe, at) {
					continue
				}
				violations = append(violations, fmt.Sprintf("%s:%d: %s 在事务闭包内调用，读不会走事务所在的那条连接",
					rel, lineOf(src, at), strings.TrimSuffix(read, "(")))
			}
		}
	}

	assertNoViolations(t, "事务闭包内不得调用不收上下文的基座读入口：把读移到事务之前，或让读真正接上 ctx", violations)
}

// ctxlessReadEntries 基座里所有「不收上下文」的读入口方法名。新增同类入口必须同步补进来，
// 否则守卫会漏掉它
var ctxlessReadEntries = []string{
	"GetById(", "GetByIds(", "GetByCond(",
	"SelectById(", "SelectByIds(", "SelectByCond(", "SelectOneByCond(",
	"SelectModelByCond(", "SelectByQuery(", "ListByCond(", "ListByCondToAny(",
	"PageByCond(", "CountByCond(",
}

type closureSpan struct {
	bodyStart int
	bodyEnd   int
}

// afterCommitSpans 找出登记「提交后动作」的闭包体范围，这些位置里的读不受事务连接约束
func afterCommitSpans(src string) []closureSpan {
	var spans []closureSpan
	for _, marker := range []string{"AfterCommit(", "RunAfterCommit("} {
		spans = append(spans, closureSpansFor(src, marker)...)
	}
	return spans
}

func coveredByAny(spans []closureSpan, at int) bool {
	for _, sp := range spans {
		if at >= sp.bodyStart && at < sp.bodyEnd {
			return true
		}
	}
	return false
}

// closureSpansFor 找出 marker(...) 之后每个 func(...) 闭包体的字节范围
func closureSpansFor(src, marker string) []closureSpan {
	var spans []closureSpan
	search := 0
	for {
		hit := strings.Index(src[search:], marker)
		if hit < 0 {
			break
		}
		after := search + hit + len(marker)
		fnRel := strings.Index(src[after:], "func(")
		if fnRel < 0 {
			search = after
			continue
		}
		fnAt := after + fnRel
		braceRel := strings.IndexByte(src[fnAt:], '{')
		if braceRel < 0 {
			search = fnAt
			continue
		}
		brace := fnAt + braceRel
		end := matchingBrace(src, brace)
		if end < 0 {
			search = brace
			continue
		}
		spans = append(spans, closureSpan{bodyStart: brace + 1, bodyEnd: end})
		search = end
	}
	return spans
}

// txClosureSpans 找出所有 .Tx(...) / .TxWith(...) 调用中事务闭包体的字节范围
func txClosureSpans(src string) []closureSpan {
	var spans []closureSpan
	for _, marker := range []string{".Tx(", ".TxWith("} {
		spans = append(spans, closureSpansFor(src, marker)...)
	}
	return spans
}

// matchingBrace 从 open 处的左花括号起配对，跳过字符串、字符、原始字符串与行注释里的花括号
func matchingBrace(src string, open int) int {
	depth := 0
	var inString, inChar, inRaw, inLineComment bool
	for i := open; i < len(src); i++ {
		c := src[i]
		if inLineComment {
			if c == '\n' {
				inLineComment = false
			}
			continue
		}
		if inRaw {
			if c == '`' {
				inRaw = false
			}
			continue
		}
		if inString {
			if c == '\\' {
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		if inChar {
			if c == '\\' {
				i++
				continue
			}
			if c == '\'' {
				inChar = false
			}
			continue
		}
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			inLineComment = true
		case c == '`':
			inRaw = true
		case c == '"':
			inString = true
		case c == '\'':
			inChar = true
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// lineOf 返回字节偏移所在的行号（1 基）
func lineOf(src string, at int) int {
	return 1 + strings.Count(src[:at], "\n")
}

func imported(path, pkg string) bool {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
	if err != nil {
		return false
	}
	for _, spec := range parsed.Imports {
		if strings.Trim(spec.Path.Value, `"`) == pkg {
			return true
		}
	}
	return false
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s failed: %v", path, err)
	}
	return string(content)
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func assertNoViolations(t *testing.T, guidance string, violations []string) {
	t.Helper()
	if len(violations) == 0 {
		return
	}
	t.Errorf("架构守卫失败：%s\n%s", guidance, strings.Join(violations, "\n"))
}
