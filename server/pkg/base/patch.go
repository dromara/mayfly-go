package base

import (
	"errors"
	"fmt"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"slices"
	"strings"

	"gorm.io/gorm"
)

// 更新补丁相关错误
var (

	// ErrInvalidPatchColumn 列名不是合法标识符。构造期即拒绝并携带该错误，而不是留给下游
	// schema 校验兜住：否则「白名单被绕过」与「列不存在」两种故障在调用方眼里无法区分
	ErrInvalidPatchColumn = errors.New("invalid column name in update patch")

	// ErrEmptyPatch 空补丁：调用方十有八九是漏设了列，而不是真要更新零个列。
	// 让它报错，而不是走成「0 行受影响」的正常态，否则漏设会被静默吞掉
	ErrEmptyPatch = errors.New("update patch has no column to set")
	// ErrReservedPatchColumn 主键与逻辑删除列由基座维护，不允许通过补丁改写
	ErrReservedPatchColumn = errors.New("update patch touches a reserved column (primary key / soft delete)")
)

// patchIdentifierPattern 合法的列名标识符。
//
// Inc/Expr 无法用占位符表达列名（列名是标识符不是值），因此必须先过白名单再拼进 SQL 片段
var patchIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

type patchKind int8

const (
	patchInvalid patchKind = iota // 列名非法的占位变更，落库前一律拒绝
	patchSet                      // 直接赋值为给定值（允许零值，这是它与实体形态更新的根本区别）
	patchInc                      // 列 = 列 + delta，数据库侧原子自增，避免读-改-写竞态
	patchExpr                     // 列 = 任意 SQL 表达式
)

// patchOp 补丁里的一条列变更
type patchOp struct {
	kind   patchKind
	column string
	value  any
	expr   string
	args   []any
}

// Patch 更新补丁：显式的「列 → 值」集合。
//
// 基座的实体形态更新（UpdateById/UpdateByCond 传实体）沿用的是「只写非零字段」语义，这在
// 「把备注清空、把状态归零」这类需求上做不到且不会报错——字段是零值时它根本不在 SET 子句里。
// 补丁存在的意义就是让「写哪些列」由调用方明确说出口，而不是由取值是不是零值来推断
type Patch struct {
	ops []patchOp
}

// NewPatch 创建空补丁
func NewPatch() Patch {
	return Patch{}
}

// Set 设置某列的值，零值（0、""、nil、false）同样会写进 SET 子句
func (p Patch) Set(column string, value any) Patch {
	return p.append(patchOp{kind: patchSet, column: column, value: value})
}

// Sets 批量设置列值。键的顺序不固定（来自 map），因此写入前会按列名排序，
// 保证同一份入参生成的 SQL 文本恒定
func (p Patch) Sets(values map[string]any) Patch {
	columns := make([]string, 0, len(values))
	for column := range values {
		columns = append(columns, column)
	}
	// 一次成型而非逐列调用 Set：Set 走值语义的 append，每列都会整体复制已积累的
	// ops（防派生分支共享底层数组的必要代价），N 列即 O(N²) 次复制；预分配足量后
	// 追加到自有数组，「分支间不共享底层数组」的语义不变而复制只有一次
	ops := make([]patchOp, len(p.ops), len(p.ops)+len(values))
	copy(ops, p.ops)
	for _, column := range sortedColumns(columns) {
		ops = append(ops, validatedOp(patchOp{kind: patchSet, column: column, value: values[column]}))
	}
	p.ops = ops
	return p
}

// Inc 数据库侧原子自增（col = col + delta）。
//
// 用于计数器、用量累加这类「读改写会丢并发更新」的列。走这一条路径后，业务不再需要为原子累加
// 手写 UPDATE 语句，也不再有人把这类累加塞进应用层读-改-写
func (p Patch) Inc(column string, delta any) Patch {
	return p.append(patchOp{kind: patchInc, column: column, value: delta})
}

// Expr 用 SQL 表达式赋值，表达式里的 ? 由 args 绑定。
//
// 表达式本身必须是参数化写法（如 "id"、"CONCAT(name, ?)"），列名不接受调用方拼接
func (p Patch) Expr(column, expr string, args ...any) Patch {
	return p.append(patchOp{kind: patchExpr, column: column, expr: expr, args: args})
}

// Len 补丁包含的列变更数量
func (p Patch) Len() int {
	return len(p.ops)
}

// Columns 返回补丁涉及的列名（已按名称排序），供基座做列存在性校验
func (p Patch) Columns() []string {
	columns := make([]string, 0, len(p.ops))
	for _, op := range p.ops {
		columns = append(columns, op.column)
	}
	return sortedColumns(columns)
}

func (p Patch) append(op patchOp) Patch {
	// 必须整体复制而不是直接 append：Patch 是值语义的构建器，「从同一份基补丁派生几个分支」正是
	// 这个形态邀请的用法，而 append 在容量有富余时会写进共享的底层数组，派生出的分支因此互相
	// 改写（base 有 3 列时 p1=base.Set("x")、p2=base.Set("y")，p1 里的 x 会变成 y），且不报任何错。
	// 补丁都是个位数量级，复制的代价远小于这类静默写错列的代价
	ops := make([]patchOp, len(p.ops), len(p.ops)+1)
	copy(ops, p.ops)
	p.ops = append(ops, validatedOp(op))
	return p
}

// validatedOp 规范化列名并过标识符白名单。
//
// Inc/Expr 会把列名拼进 SQL 片段（列名是标识符，无法参数化），Set 的列名同样会出现在
// SET 子句里，因此所有变更统一校验，避免调用方把外部输入当成列名传进来
func validatedOp(op patchOp) patchOp {
	op.column = normalizePatchColumn(op.column)
	if !patchIdentifierPattern.MatchString(op.column) {
		// 记为一条「非法列」变更而非静默丢弃：静默丢弃会让调用方以为更新成功了。
		// 列名保留在变更里，落库拒绝时才指得出是哪一列非法
		op = patchOp{kind: patchInvalid, column: op.column}
	}
	return op
}

// toValues 把补丁编译成 GORM 的更新值（列名 -> 值/表达式）。
func (p Patch) toValues() (map[string]any, error) {
	if len(p.ops) == 0 {
		return nil, ErrEmptyPatch
	}

	values := make(map[string]any, len(p.ops)+3)
	for _, op := range p.ops {
		if op.kind == patchInvalid {
			return nil, fmt.Errorf("%w: %q", ErrInvalidPatchColumn, op.column)
		}
		if isReservedPatchColumn(op.column) {
			return nil, fmt.Errorf("%w: %s", ErrReservedPatchColumn, op.column)
		}
		switch op.kind {
		case patchSet:
			values[op.column] = op.value
		case patchInc:
			// 列名已过白名单，拼接安全；数值仍走占位符绑定
			values[op.column] = gorm.Expr(op.column+" + ?", op.value)
		case patchExpr:
			values[op.column] = gorm.Expr(op.expr, op.args...)
		}
	}

	// 修改审计不在这里补：map 形态的更新由 gormx.UpdateByCond 统一按实体真实拥有的列补齐。
	// 补丁只描述「调用方声明的业务列」，列存在性校验（Columns）与写入形态（toValues）因此
	// 保持同一口径，不会出现校验放过一个实体根本没有的审计列的情况

	return values, nil
}

// normalizePatchColumn 允许调用方写实体字段名（camelCase）或物理列名（snake_case）
func normalizePatchColumn(column string) string {
	trimmed := strings.TrimSpace(column)
	// 表限定列（如 "t.status"）保持原样，其余统一蛇形化
	if strings.Contains(trimmed, ".") {
		segments := strings.Split(trimmed, ".")
		for i, segment := range segments {
			segments[i] = toSnakeColumn(segment)
		}
		return strings.Join(segments, ".")
	}
	return toSnakeColumn(trimmed)
}

func toSnakeColumn(column string) string {
	return stringx.Camel2Snake(column)
}

func isReservedPatchColumn(column string) bool {
	switch column {
	case model.IdColumn, model.DeletedColumn, model.DeleteTimeColumn:
		return true
	}
	return false
}

func sortedColumns(columns []string) []string {
	sorted := make([]string, len(columns))
	copy(sorted, columns)
	slices.Sort(sorted)
	return sorted
}
