package sqlexec

// 列级访问控制的判定与结果裁剪。
//
// 两类场景的处置不同，这是本文件最重要的约定：
//
//	交互查询：显式写出无权列 → 拒绝；`*` → 由结果集裁剪掉无权列（用户没点名要那一列）；
//	产物级（整表导出 / 复制表 / 整列采样）：只要表上存在无权列 → 整条拒绝。
//	        产物少一列比报错更危险：导出的文件会看起来「完整」，恢复或交出去才暴露缺列。
//
// 为什么显式列引用要拒绝而不是裁剪：列被静默拿掉后，`SELECT phone AS p` 这类别名会让
// 名字对不上，用户以为拿到了数据其实是空的——不如直接说清楚没有权限。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

// ColumnRef 语句里显式写出的一处列引用。
//
// Table 允许为空：SELECT 列表里 `t.col` 拿得到表别名，但别名到真表的映射要跨 JOIN 解析；
// 这里按「任一目标表命中即拒」处理，宁可多拦一次，也不让 JOIN 写法成为旁路。
type ColumnRef struct {
	Table  string
	Column string
}

// columnsFor 返回这些表上「当前账号无权」的列（列名小写）。
//
// tables 为空表示不限定表（整库判定）：调用方必须明确意图，别拿空表清单当「全部放行」。
func (s *tableAclSnapshot) columnsFor(accountRoles []uint64, rolesUnresolved bool, tables []string) map[string]struct{} {
	denied := make(map[string]struct{})
	if s == nil || len(s.columns) == 0 {
		return denied
	}
	matched := func(roleIds []uint64) bool {
		if len(roleIds) == 0 || rolesUnresolved {
			return true
		}
		for _, roleId := range roleIds {
			for _, owned := range accountRoles {
				if roleId == owned {
					return true
				}
			}
		}
		return false
	}

	collect := func(tableKey string) {
		for column, roleIds := range s.columns[tableKey] {
			if matched(roleIds) {
				denied[column] = struct{}{}
			}
		}
	}

	if len(tables) == 0 {
		for tableKey := range s.columns {
			collect(tableKey)
		}
		return denied
	}
	for _, table := range tables {
		if tableKey := aclTableKey(table); tableKey != "" {
			collect(tableKey)
		}
	}
	return denied
}

// accountRoles 当前账号角色；accountId 为 0 时 resolved=false（身份未知）
func (a *tableAclAppImpl) accountRoles(ctx context.Context) (roleIds []uint64, resolved bool, err error) {
	la := contextx.GetLoginAccount(ctx)
	if la == nil || la.Id == 0 {
		return nil, false, nil
	}
	roleIds, err = a.accountRoleIds(la.Id)
	if err != nil {
		logx.ErrorfContext(ctx, "column acl: failed to load account roles, denying by default: %s", err.Error())
		return nil, true, errorx.NewBizI(ctx, imsg.ErrTableAclRolesUnavailable)
	}
	return roleIds, true, nil
}

// DeniedColumns 当前账号在这些表上无权看到的列名集合（小写）
func (a *tableAclAppImpl) DeniedColumns(ctx context.Context, dbId uint64, tables []string) (map[string]struct{}, error) {
	if dbId == 0 {
		return map[string]struct{}{}, nil
	}
	snap, err := a.snapshot(ctx, dbId)
	if err != nil {
		return nil, err
	}
	if snap == nil || len(snap.columns) == 0 {
		return map[string]struct{}{}, nil
	}
	roleIds, resolved, err := a.accountRoles(ctx)
	if err != nil {
		return nil, err
	}
	return snap.columnsFor(roleIds, !resolved, tables), nil
}

// AssertColumnsAllowed 语句里显式引用了无权列即拒绝。
func (a *tableAclAppImpl) AssertColumnsAllowed(ctx context.Context, dbId uint64, tables []string, refs []ColumnRef) error {
	if dbId == 0 || len(refs) == 0 {
		return nil
	}
	denied, err := a.DeniedColumns(ctx, dbId, tables)
	if err != nil {
		return err
	}
	if len(denied) == 0 {
		return nil
	}
	for _, ref := range refs {
		if _, ok := denied[aclTableKey(ref.Column)]; ok {
			return errorx.NewBizI(ctx, imsg.ErrColumnAccessDenied, "column", ref.Column)
		}
	}
	return nil
}

// AssertWholeTableAllowed 产物级场景：表上存在当前账号无权的列即拒绝。
func (a *tableAclAppImpl) AssertWholeTableAllowed(ctx context.Context, dbId uint64, tables []string) error {
	if dbId == 0 || len(tables) == 0 {
		return nil
	}
	denied, err := a.DeniedColumns(ctx, dbId, tables)
	if err != nil {
		return err
	}
	if len(denied) == 0 {
		return nil
	}
	columns := make([]string, 0, len(denied))
	for column := range denied {
		columns = append(columns, column)
	}
	return errorx.NewBizI(ctx, imsg.ErrWholeTableReadDenied, "columns", strings.Join(columns, ", "))
}

// AssertExportAllowed 导出场景：显式列按列判；星号（会带上全部列）要求整表可读。
//
// 导出不能像交互查询那样「裁掉了事」：CSV/Excel 少一列时用户看不出异常，
// 等到恢复或对外提供数据才暴露，那时已经无法追溯是哪一次导出缺的。
func (a *tableAclAppImpl) AssertExportAllowed(ctx context.Context, dbId uint64, tables []string, stmt sqlstmt.Stmt) error {
	// 表级是独立闸门：列级只覆盖「列规则」，表级 deny/白名单必须单独判
	if err := a.AssertTablesAllowed(ctx, dbId, tables); err != nil {
		return err
	}
	if err := a.AssertColumnsAllowed(ctx, dbId, tables, statementColumnRefs(stmt)); err != nil {
		return err
	}
	selectStmt, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok || !selectHasStar(selectStmt) {
		return nil
	}
	return a.AssertWholeTableAllowed(ctx, dbId, tables)
}

// HasColumnRules 该库是否配置了列级规则
func (a *tableAclAppImpl) HasColumnRules(ctx context.Context, dbId uint64) (bool, error) {
	if dbId == 0 {
		return false, nil
	}
	snap, err := a.snapshot(ctx, dbId)
	if err != nil {
		return false, err
	}
	return snap != nil && len(snap.columns) > 0, nil
}

// ListColumns 某库的列级规则
func (a *tableAclAppImpl) ListColumns(dbId uint64) ([]*entity.DbColumnAcl, error) {
	return a.columnAclRepo.ListByDb(dbId)
}

// SaveColumn 保存列级规则（同库同表同列只保留一条）
func (a *tableAclAppImpl) SaveColumn(ctx context.Context, acl *entity.DbColumnAcl) error {
	acl.Table = strings.TrimSpace(acl.Table)
	acl.Column = strings.TrimSpace(acl.Column)
	if acl.DbId == 0 || acl.Table == "" || acl.Column == "" {
		return errorx.NewBizI(ctx, imsg.ErrColumnAclColumnRequired)
	}
	if acl.Id > 0 {
		if err := a.columnAclRepo.UpdateById(ctx, acl, "table_name", "column_name", "role_ids", "remark"); err != nil {
			return err
		}
		a.invalidate(acl.DbId)
		return nil
	}
	rules, err := a.columnAclRepo.ListByDb(acl.DbId)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		if aclTableKey(rule.Table) != aclTableKey(acl.Table) || aclTableKey(rule.Column) != aclTableKey(acl.Column) {
			continue
		}
		rule.RoleIds = acl.RoleIds
		rule.Remark = acl.Remark
		if err := a.columnAclRepo.UpdateById(ctx, rule, "role_ids", "remark"); err != nil {
			return err
		}
		a.invalidate(acl.DbId)
		return nil
	}
	if err := a.columnAclRepo.Insert(ctx, acl); err != nil {
		return err
	}
	a.invalidate(acl.DbId)
	return nil
}

// DeleteColumn 删除列级规则
func (a *tableAclAppImpl) DeleteColumn(ctx context.Context, id uint64) error {
	acl, err := a.columnAclRepo.GetById(id)
	if err != nil {
		return err
	}
	if err := a.columnAclRepo.DeleteById(ctx, id); err != nil {
		return err
	}
	a.invalidate(acl.DbId)
	return nil
}

// statementColumnRefs 语句里显式出现的列引用（用于列级拒绝判定）。
//
// 表达式里的列不会被结构化拆出来（COUNT(secret)、CONCAT(a, b)），因此对表达式文本
// 再做一次整词比对：这一侧宁可误拦（把同名列都算上），也不能让「外头包一层函数」成为旁路。
func statementColumnRefs(stmt sqlstmt.Stmt) []ColumnRef {
	switch typed := stmt.(type) {
	case *sqlstmt.SelectStmt:
		refs := selectColumnRefs(typed)
		// 条件里引用无权列同样要拦：`WHERE phone LIKE '138%'` 能把值一点点试出来，
		// 「不返回这一列」并不等于「不能用这一列做条件」
		refs = append(refs, exprColumnRefs(typed.Where)...)
		// ORDER BY / GROUP BY / HAVING 同属「用这一列做条件」：
		// 按无权列排序就能二分出它的值，按它分组计数更直接暴露取值分布
		for _, group := range typed.GroupBy {
			refs = append(refs, ColumnRef{Column: group})
		}
		refs = append(refs, exprColumnRefs(typed.Having)...)
		for _, order := range typed.OrderBy {
			refs = append(refs, exprColumnRefs(&sqlstmt.Expr{Text: order.Text})...)
		}
		return refs
	case *sqlstmt.InsertStmt:
		// VALUES 里出现的列引用（如 `INSERT INTO t (a) SELECT phone FROM u` 由 Select 分支覆盖）
		refs := make([]ColumnRef, 0, len(typed.Columns))
		for _, column := range typed.Columns {
			refs = append(refs, ColumnRef{Table: typed.Table.Name, Column: column})
		}
		return refs
	case *sqlstmt.UpdateStmt:
		refs := make([]ColumnRef, 0, len(typed.Set)+8)
		for _, assignment := range typed.Set {
			refs = append(refs, ColumnRef{Column: assignment.Column})
		}
		return append(refs, exprColumnRefs(typed.Where)...)
	case *sqlstmt.DeleteStmt:
		// 删除条件引用无权列：删除行为本身会暴露「该列等于某值时有多少行」
		return exprColumnRefs(typed.Where)
	}
	return nil
}

// exprColumnRefs 从条件表达式里取列引用（按文本整词，保守取全）
func exprColumnRefs(expr *sqlstmt.Expr) []ColumnRef {
	if expr == nil {
		return nil
	}
	words := identifierWords(expr.Text)
	refs := make([]ColumnRef, 0, len(words))
	for _, word := range words {
		refs = append(refs, ColumnRef{Column: word})
	}
	return refs
}

// selectColumnRefs 查询列表里的列引用（含 UNION 各分支）
func selectColumnRefs(stmt *sqlstmt.SelectStmt) []ColumnRef {
	if stmt == nil {
		return nil
	}
	refs := make([]ColumnRef, 0, len(stmt.Items))
	for _, item := range stmt.Items {
		if item.IsStar() {
			// 星号代表「全部列」：整表导出场景由 AssertWholeTableAllowed 拒，
			// 交互查询场景由结果集裁剪兜住，这里不产生具体列引用
			continue
		}
		if item.ColumnName != "" {
			refs = append(refs, ColumnRef{Table: item.TableAlias, Column: item.ColumnName})
		}
		for _, word := range identifierWords(item.Text) {
			refs = append(refs, ColumnRef{Table: item.TableAlias, Column: word})
		}
	}
	for _, union := range stmt.Unions {
		refs = append(refs, selectColumnRefs(union.Select)...)
	}
	return refs
}

// identifierWords 从表达式文本里取标识符（字母/下划线开头）。
//
// 不做关键字过滤：这里只用于与「无权列名」比对，关键字永远配不上具体列，
// 多留几个词不会带来误判，却能避免维护一份与解析器不同步的关键字表。
func identifierWords(text string) []string {
	words := make([]string, 0, 8)
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	for _, r := range text {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			current.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return words
}

// TrimDeniedColumns 从结果集里裁掉无权列（`SELECT *` 的兜底）。
//
// 只动返回给调用方的列与行数据，不改 SQL、不重查：裁剪发生在结果已经取回之后，
// 越权的值从未离开服务端的这一层。
func (a *tableAclAppImpl) TrimDeniedColumns(ctx context.Context, dbId uint64, tables []string, cols []*dbi.QueryColumn, rows []map[string]any) ([]*dbi.QueryColumn, []map[string]any, error) {
	// 没有表来源（如 `SELECT 1`）就没有「这张表的列」可裁；而表引用无法解析的语句
	// 已在执行入口被 fail-closed 拒绝，不会走到这里。此处若沿用「空表=整库」的兜底，
	// 会按列名与任意保护表撞名而静默删列——用户看到的是一份少了列却「成功」的结果集。
	if len(tables) == 0 {
		return cols, rows, nil
	}
	if dbId == 0 || len(cols) == 0 {
		return cols, rows, nil
	}
	denied, err := a.DeniedColumns(ctx, dbId, tables)
	if err != nil {
		return nil, nil, err
	}
	if len(denied) == 0 {
		return cols, rows, nil
	}

	kept := make([]*dbi.QueryColumn, 0, len(cols))
	for _, col := range cols {
		if col == nil {
			continue
		}
		if _, ok := denied[aclTableKey(col.Name)]; ok {
			continue
		}
		kept = append(kept, col)
	}
	if len(kept) == len(cols) {
		return cols, rows, nil
	}

	// 行数据按列 key 存放（见 dbi.scanRowMap），直接删键：
	// 重建 map 会让「同名列去重后带后缀」的 key 映射错位
	removed := make([]string, 0, len(cols)-len(kept))
	for _, col := range cols {
		if col == nil {
			continue
		}
		if _, ok := denied[aclTableKey(col.Name)]; !ok {
			continue
		}
		removed = append(removed, col.Key)
	}
	for _, row := range rows {
		for _, key := range removed {
			delete(row, key)
		}
	}
	return kept, rows, nil
}
