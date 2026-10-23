package model

import (
	"fmt"
	"mayfly-go/pkg/consts"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/stringx"
	"reflect"
	"regexp"
	"strings"
)

// orderByPattern 允许的排序片段：单个或表限定的列名，可选 ASC/DESC
var orderByPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?(\s+([Aa][Ss][Cc]|[Dd][Ee][Ss][Cc]))?$`)

// QueryCond 查询条件构造器。
//
// 条件按追加顺序保存而非放进 map：同一条逻辑查询必须生成恒定的 SQL 文本与参数顺序，
// 否则慢 SQL 归因、驱动侧语句缓存、日志与回归比对都会随每次遍历随机化而失去可比性。
type QueryCond struct {
	selectColumns []string // 查询的列信息
	condModel     any      // 条件模型

	wheres []WhereClause // 条件片段，按追加顺序生效

	orderBy []string

	// err 记录构造期发现的非法输入（如非法排序片段）。
	//
	// 这里不 panic 也不返回 error，是为了保住链式调用的可读性；由执行层（gormx）在生成
	// 语句时通过 AddError 转为普通错误返回，绝不带着非法片段去访问数据库
	err error

	// includeDeleted 为 true 时不再自动附加逻辑删除过滤，用于回收站、审计回放等需要读取已删除行的场景
	includeDeleted bool

	dest any // 结果集指针
}

// WhereClause 单条 WHERE 片段与其绑定参数
type WhereClause struct {
	Expr string // SQL 片段，如 "instance_id = ?"
	Args []any  // 片段里的绑定参数
}

// Clone 返回条件的浅拷贝。
//
// 用于在不污染调用方传入对象的前提下追加临时条件（如游标遍历每批的主键边界、分页每批的偏移）；
// 切片均重新分配，因此追加不会回写原条件
func (q *QueryCond) Clone() *QueryCond {
	if q == nil {
		return NewCond()
	}
	copied := *q
	copied.wheres = append([]WhereClause(nil), q.wheres...)
	copied.orderBy = append([]string(nil), q.orderBy...)
	copied.selectColumns = append([]string(nil), q.selectColumns...)
	return &copied
}

// NewCond 构建查询条件
func NewCond() *QueryCond {
	return &QueryCond{}
}

// NewModelCond 新建模型条件（使用模型中非零值作为条件）
//
// 零值字段不会成为条件（与 GORM 的结构体条件语义一致），因此按 0/"" 过滤必须改用
// EqStrict；主键为 0 的实体作为条件时，等值条件由调用方显式给出更可靠
func NewModelCond(condModel any) *QueryCond {
	return &QueryCond{condModel: condModel}
}

// 设置结果集绑定的指针
func (q *QueryCond) Dest(dest any) *QueryCond {
	q.dest = dest
	return q
}

// Columns 设置查询的列投影，整体替换而非追加。
//
// 显式列参数为空时保持既有投影不变：基座读取入口（如 SelectByCond 的可变列参数）
// 以「未指定列」表示沿用条件里已声明的投影，展开成零参的 Columns() 若清空投影，
// 调用方的声明会被静默丢弃。入参切片会复制一份，调用方事后改动自己的切片不影响本条件
func (q *QueryCond) Columns(columns ...string) *QueryCond {
	if len(columns) == 0 {
		return q
	}
	q.selectColumns = append([]string(nil), columns...)
	return q
}

func (q *QueryCond) OrderByDesc(column string) *QueryCond {
	return q.OrderBy(stringx.Camel2Snake(column) + " DESC")
}

func (q *QueryCond) OrderByAsc(column string) *QueryCond {
	return q.OrderBy(stringx.Camel2Snake(column) + " ASC")
}

// OrderBy 追加排序片段，可多次调用形成多级排序。
//
// 排序片段无法用占位符参数化（它是标识符而非值），因此只接受「列名[.列名] [ASC|DESC]」
// 形态：仓储层普遍把 orderBy 作为可变参一路透传上来，一旦将来某处把请求参数接进来，
// 不做校验就会变成拼接进 SQL 的注入面
func (q *QueryCond) OrderBy(orderBy ...string) *QueryCond {
	for _, o := range orderBy {
		if !orderByPattern.MatchString(strings.TrimSpace(o)) {
			q.setErr(fmt.Errorf("invalid order by fragment: %q", o))
			continue
		}
		q.orderBy = append(q.orderBy, strings.TrimSpace(o))
	}
	return q
}

// GetErr 返回构造期记录的非法输入
func (q *QueryCond) GetErr() error {
	return q.err
}

func (q *QueryCond) setErr(err error) {
	if q.err == nil {
		q.err = err
	}
}

// Eq 等于 =（值为零值时跳过该条件，用于「传了才过滤」的可选条件）
func (q *QueryCond) Eq(column string, val any) *QueryCond {
	return q.Cond(consts.Eq, column, val, true)
}

// EqStrict 等于 =，零值也作为条件（如按 status=0、is_deleted=1 过滤）
func (q *QueryCond) EqStrict(column string, val any) *QueryCond {
	return q.Cond(consts.Eq, column, val, false)
}

// Like LIKE %xx%，空串跳过
func (q *QueryCond) Like(column string, val string) *QueryCond {
	if val == "" {
		return q
	}
	return q.Cond(consts.Like, column, "%"+val+"%", false)
}

// RLike LIKE xx%，空串跳过
func (q *QueryCond) RLike(column string, val string) *QueryCond {
	if val == "" {
		return q
	}
	return q.Cond(consts.Like, column, val+"%", false)
}

// In 属于。集合的取值语义必须区分「没传」与「传了空集合」，否则可选过滤会退化成恒不命中：
//   - nil 集合     视为未提供过滤条件，整条跳过
//   - 非 nil 空集合 视为「集合里没有任何值」，生成恒假条件，绝不可跳过（跳过等于放开全表）
func (q *QueryCond) In(column string, val any) *QueryCond {
	if isNilCollection(val) {
		return q
	}
	if isEmptyCollection(val) {
		return q.And(alwaysFalseExpr)
	}
	return q.Cond(consts.In, column, val, false)
}

// NotIn 不属于。没有需要排除的值（nil 或空集合）时整条跳过，即「不排除任何行」；
// 生成 `NOT IN ()`/`NOT IN (NULL)` 会让结果集恒空，与调用方意图相反
func (q *QueryCond) NotIn(column string, val any) *QueryCond {
	if isNilCollection(val) || isEmptyCollection(val) {
		return q
	}
	return q.Cond(consts.NotIn, column, val, false)
}

// Ne 不等于 !=（零值跳过：与 Eq 同属「传了才过滤」的可选条件族）
func (q *QueryCond) Ne(column string, val any) *QueryCond {
	return q.Cond(consts.Ne, column, val, true)
}

// Gt 大于 >（零值跳过，适用于表单可选的下限；需要恒生效请用带零值的严格入口）
func (q *QueryCond) Gt(column string, val any) *QueryCond {
	return q.Cond(consts.Gt, column, val, true)
}

// Ge 大于等于 >=（零值跳过）
func (q *QueryCond) Ge(column string, val any) *QueryCond {
	return q.Cond(consts.Ge, column, val, true)
}

// Lt 小于 <（零值跳过）
func (q *QueryCond) Lt(column string, val any) *QueryCond {
	return q.Cond(consts.Lt, column, val, true)
}

// Le 小于等于 <=（零值跳过）
func (q *QueryCond) Le(column string, val any) *QueryCond {
	return q.Cond(consts.Le, column, val, true)
}

// IsNull 列值为 NULL。Eq(column, nil) 会被当作「未提供条件」跳过，判空必须用本方法
func (q *QueryCond) IsNull(column string) *QueryCond {
	return q.And(stringx.Camel2Snake(column) + " IS NULL")
}

// NotNull 列值非 NULL
func (q *QueryCond) NotNull(column string) *QueryCond {
	return q.And(stringx.Camel2Snake(column) + " IS NOT NULL")
}

// Between 闭区间过滤；任一端为零值则退化为单端比较，两端都空则整条跳过。
//
// 与 Eq 族同为零值跳过语义：下界为 0 的区间（如 [0, 10]）的 0 会被当作「未提供」丢弃，
// 需要含零下界的闭区间时改用 And 显式给出两端
func (q *QueryCond) Between(column string, from, to any) *QueryCond {
	snakeColumn := stringx.Camel2Snake(column)
	fromBlank, toBlank := anyx.IsBlank(from), anyx.IsBlank(to)
	switch {
	case fromBlank && toBlank:
		return q
	case fromBlank:
		return q.And(snakeColumn+" <= ?", to)
	case toBlank:
		return q.And(snakeColumn+" >= ?", from)
	default:
		return q.And(snakeColumn+" BETWEEN ? AND ?", from, to)
	}
}

// IncludeDeleted 同时读取逻辑删除的数据（回收站、审计回放等），关闭自动软删过滤
func (q *QueryCond) IncludeDeleted() *QueryCond {
	q.includeDeleted = true
	return q
}

// And 追加原始 WHERE 片段。片段文本相同的多次追加会同时生效（SQL 以 AND 连接），
// 因此重复追加同一条件不会像 map 承载那样静默丢掉先前条件
func (q *QueryCond) And(columnOrQuery string, val ...any) *QueryCond {
	q.wheres = append(q.wheres, WhereClause{Expr: columnOrQuery, Args: val})
	return q
}

// Cond 追加一个运算符条件。
//
// 列名统一走 Camel2Snake，因此调用方既可以写实体字段名（instanceId）也可以写物理列名
func (q *QueryCond) Cond(cond, column string, val any, skipBlank bool) *QueryCond {
	// 零值跳过
	if skipBlank && anyx.IsBlank(val) {
		return q
	}
	return q.And(stringx.Camel2Snake(column)+" "+cond+" ?", val)
}

// HasCondOnColumn 判断调用方是否已就该列显式表过态。
//
// 自动附加的过滤（如逻辑删除过滤）不得覆盖或重复调用方已写明的条件，
// 否则「查已删除数据」这类意图会被静默改写
func (q *QueryCond) HasCondOnColumn(column string) bool {
	for i := range q.wheres {
		if exprStartsWithColumn(q.wheres[i].Expr, column) {
			return true
		}
	}
	if q.condModel != nil && columnHasValueInModel(q.condModel, stringx.Camel2Snake(column)) {
		return true
	}
	return false
}

// exprStartsWithColumn 判断 WHERE 片段的首部标识符是否指向 column。
//
// 识别契约：片段首部就是目标列名（结构化构造器 Eq/Like/... 与常规手写片段均如此），可能带
// 表限定前缀（"t.is_deleted = 1"）。按标识符边界取首部而非匹配「列名后紧跟空白」的形态，
// "is_deleted=1"、"IsDeleted >= ?" 与 "t.is_deleted" 同样识别，且不会把相近列名
// （is_deleted_x）误认成目标列。函数开头或前导括号的复杂片段不由本方法识别
func exprStartsWithColumn(expr, column string) bool {
	i := 0
	for i < len(expr) && isASCIISpace(expr[i]) {
		i++
	}
	start := i
	for i < len(expr) && isSQLIdent(expr[i]) {
		i++
	}
	ident := expr[start:i]
	if dot := strings.LastIndexByte(ident, '.'); dot >= 0 {
		ident = ident[dot+1:]
	}
	if ident == "" {
		return false
	}
	// column 也归一到蛇形再比：调用方拿实体字段名（IsDeleted）提问时，
	// 与拿物理列名提问同样成立
	return strings.EqualFold(ident, column) ||
		strings.EqualFold(stringx.Camel2Snake(ident), stringx.Camel2Snake(column))
}

// isSQLIdent SQL 标识符与表限定符的合法字符（字母、数字、下划线、点）
func isSQLIdent(c byte) bool {
	return c == '_' || c == '.' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isASCIISpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// HasConds 是否存在调用方声明的过滤条件（条件片段，或条件模型里有非零字段）。
//
// 由基座自动附加的条件（如软删过滤）不算：它不是调用方的意图，拿它当「有条件」会让整表
// 操作看起来合法
func (q *QueryCond) HasConds() bool {
	if q == nil {
		return false
	}
	if len(q.wheres) > 0 {
		return true
	}
	return condModelHasValue(q.condModel)
}

// condModelHasValue 条件模型里是否有会变成条件的内容（GORM 只把非零字段/非空键变成条件）。
//
// 看不懂的形态一律当作有条件：这里宁可放过也不能把正常调用误拦，真正的无条件更新还有
// 驱动层的全表操作拦截兜底
func condModelHasValue(condModel any) bool {
	rv := indirectValue(condModel)
	if !rv.IsValid() {
		return false
	}
	switch rv.Kind() {
	case reflect.Map:
		return rv.Len() > 0
	case reflect.Struct:
		rt := rv.Type()
		for i := 0; i < rt.NumField(); i++ {
			if rt.Field(i).PkgPath != "" {
				continue
			}
			if !anyx.IsBlank(rv.Field(i).Interface()) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// indirectValue 剥掉指针取得底层值；入参为 nil 或 nil 指针时返回零 Value
func indirectValue(value any) reflect.Value {
	if value == nil {
		return reflect.Value{}
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Value{}
		}
		rv = rv.Elem()
	}
	return rv
}

// GetWheres 按追加顺序返回条件片段。
//
// 返回的是内部切片（零拷贝），调用方只读勿改：改它等于改本条件
func (q *QueryCond) GetWheres() []WhereClause {
	return q.wheres
}

func (q *QueryCond) GetCondModel() any {
	return q.condModel
}

// GetSelectColumns 返回列投影。
//
// 返回的是内部切片（零拷贝），调用方只读勿改：改它等于改本条件
func (q *QueryCond) GetSelectColumns() []string {
	return q.selectColumns
}

func (q *QueryCond) GetOrderBy() []string {
	return q.orderBy
}

// IsDeletedIgnored 是否已声明需要读取逻辑删除的数据
func (q *QueryCond) IsDeletedIgnored() bool {
	return q.includeDeleted
}

// HasDest 是否已绑定结果集（条件模型也可充当结果集，见 GetDest）
func (q *QueryCond) HasDest() bool {
	if !anyx.IsBlank(q.dest) || !anyx.IsBlank(q.condModel) {
		return true
	}
	return false
}

// 获取输出目标结果集指针，若dest为空，则使用condModel。
//
// 调用方须先用 HasDest 判空并返回错误，基座不应因用法问题 panic 整个进程
func (q *QueryCond) GetDest() any {
	if !anyx.IsBlank(q.dest) {
		return q.dest
	}
	if !anyx.IsBlank(q.condModel) {
		return q.condModel
	}
	return nil
}

// alwaysFalseExpr 恒假条件，用于表达「集合内无任何值」这类与「未提供条件」截然不同的语义
const alwaysFalseExpr = "1 = 0"

// isNilCollection 判断入参是否为 nil 切片/数组（区分未提供与提供空集合）
func isNilCollection(val any) bool {
	if val == nil {
		return true
	}
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Kind() == reflect.Slice && rv.IsNil()
	}
	return false
}

// isEmptyCollection 判断入参是否为长度 0 的切片/数组
func isEmptyCollection(val any) bool {
	if val == nil {
		return true
	}
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len() == 0
	}
	return false
}

// columnHasValueInModel 判断条件模型里该列是否被赋了非零值（GORM 只会把非零字段变成条件）。
//
// 嵌入结构体的字段会被 GORM 展平成表列参与条件，判定必须同样下钻：软删列（is_deleted）
// 恰好位于实体嵌入的 model.Model 里，漏扫会让「用条件模型查已删除数据」被自动软删
// 过滤覆盖，结果恒为空
func columnHasValueInModel(condModel any, snakeColumn string) bool {
	return structFieldHasColumnValue(indirectValue(condModel), snakeColumn)
}

func structFieldHasColumnValue(rv reflect.Value, snakeColumn string) bool {
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return false
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if fieldIgnoredByORM(field) {
			continue
		}
		if fieldEmbedded(field) {
			if structFieldHasColumnValue(rv.Field(i), snakeColumn) {
				return true
			}
			continue
		}
		if columnOfField(field) == snakeColumn && !anyx.IsBlank(rv.Field(i).Interface()) {
			return true
		}
	}
	return false
}

// fieldEmbedded 该字段是否会被 GORM 展平进所属结构体参与条件：
// 匿名嵌入结构体默认展平，显式 gorm:"embedded" 开启、gorm:"embedded:false" 关闭
func fieldEmbedded(field reflect.StructField) bool {
	if field.Type.Kind() != reflect.Struct {
		return false
	}
	embedded, explicit := false, false
	for _, opt := range strings.Split(field.Tag.Get("gorm"), ";") {
		name, value, _ := strings.Cut(opt, ":")
		if name == "embedded" {
			explicit = true
			embedded = value != "false"
		}
	}
	if explicit {
		return embedded
	}
	return field.Anonymous
}

// fieldIgnoredByORM gorm:"-" 标注的字段不参与条件，判它在等于把被 ORM 忽略的列
// 当成调用方已表态
func fieldIgnoredByORM(field reflect.StructField) bool {
	for _, opt := range strings.Split(field.Tag.Get("gorm"), ";") {
		if opt == "-" {
			return true
		}
	}
	return false
}

// columnOfField 取结构体字段对应的物理列名（优先 gorm column，否则蛇形字段名）
func columnOfField(field reflect.StructField) string {
	if tag := field.Tag.Get("gorm"); tag != "" {
		for _, seg := range strings.Split(tag, ";") {
			if name, ok := strings.CutPrefix(seg, "column:"); ok && name != "" {
				return name
			}
		}
	}
	return stringx.Camel2Snake(field.Name)
}
