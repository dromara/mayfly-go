package sqlexec

// 表访问控制（表级权限）。
//
// 语义：某张表被列入黑名单后，命中该名单的账号在**任何读取/写入入口**都拿不到它——
// 编辑器、数据网格、SQL 文件执行、数据导入、即席导出、查询导出、AI 工具、值补全采样、
// 复制表。这是「表级权限」而不是「列级脱敏」：脱敏保留列、只改值，这里是不许碰。
//
// 为什么不用白名单：白名单会让「以后新建的表」默认对所有人不可见，升级即断供；
// 企业最常见的诉求是「把这几张敏感表挡住」，黑名单最贴合。
//
// 判定口径：
//   - 表名忽略大小写比较（方言与建表方式会让大小写漂移，区分大小写等于给了绕过手法）；
//   - 规则 RoleIds 为空 = 对所有账号生效；非空则与账号角色求交集；
//   - 取不到账号角色时按命中处理（fail-closed）：判不出来就当作无权。

import (
	"context"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
)

// aclSnapshotTtl 规则快照的存活时间。
//
// 每次语句执行都查一遍元数据库在「长脚本/批量执行」下会把往返次数放大一个量级，
// 而表级权限的增删本身是低频管理动作：10 秒的生效延迟可以换掉整条热路径上的查询。
// 保存与删除时本实例立即失效，多实例部署下的最坏情况就是这个 TTL。
const aclSnapshotTtl = 10 * time.Second

// TableAcl 表访问控制应用
type TableAcl interface {
	base.App[*entity.DbTableAcl]

	// ListByDb 某库的全部规则（界面展示）
	ListByDb(dbId uint64) ([]*entity.DbTableAcl, error)

	// SaveAcl 保存一条规则：同一库同一张表只允许一条（按表名忽略大小写去重）
	SaveAcl(ctx context.Context, acl *entity.DbTableAcl) error

	// DeleteAcl 删除规则并立即失效快照
	DeleteAcl(ctx context.Context, id uint64) error

	// ListColumns 某库的列级规则（界面展示）
	ListColumns(dbId uint64) ([]*entity.DbColumnAcl, error)

	// SaveColumn 保存列级规则（同一库同一张表同一列只保留一条）
	SaveColumn(ctx context.Context, acl *entity.DbColumnAcl) error

	// DeleteColumn 删除列级规则
	DeleteColumn(ctx context.Context, id uint64) error

	// AssertColumnsAllowed 语句里显式引用了无权列即拒绝（别名、函数包裹都拦得住）
	AssertColumnsAllowed(ctx context.Context, dbId uint64, tables []string, refs []ColumnRef) error

	// AssertExportAllowed 导出场景：显式列按列判；星号（会带上全部列）要求整表可读
	AssertExportAllowed(ctx context.Context, dbId uint64, tables []string, stmt sqlstmt.Stmt) error

	// AssertWholeTableAllowed 产物级场景（整表导出/复制/整列采样）：
	// 表上存在当前账号无权的列即拒绝——这类产物不能静默裁剪，少一列比报错更危险
	AssertWholeTableAllowed(ctx context.Context, dbId uint64, tables []string) error

	// DeniedColumns 当前账号在这些表上无权看到的列名集合（小写），供结果集裁剪使用
	DeniedColumns(ctx context.Context, dbId uint64, tables []string) (map[string]struct{}, error)

	// TrimDeniedColumns 从结果集裁掉无权列（`SELECT *` 的兜底）
	TrimDeniedColumns(ctx context.Context, dbId uint64, tables []string, cols []*dbi.QueryColumn, rows []map[string]any) ([]*dbi.QueryColumn, []map[string]any, error)

	// HasRules 该库是否配置了保护规则。
	// 调用方据此决定「要不要为这一步付解析开销」，避免没配规则时也把语句全解析一遍
	HasRules(ctx context.Context, dbId uint64) (bool, error)

	// AssertTablesAllowed 判定这些表当前账号是否可访问，命中黑名单返回业务错误
	AssertTablesAllowed(ctx context.Context, dbId uint64, tables []string) error

	// VisibleTables 过滤出当前账号「可见」的表（用于表清单 / 资源树 / SQL 补全）。
	//
	// 表级保护是「拒绝访问」：名字继续出现在列表与补全里，用户会写出一条必然被拒的语句，
	// 与「这张表对我不可见」的心智不一致。列级早就这么做了（c-metadata 过滤无权列），
	// 表级此前漏了这一步。
	VisibleTables(ctx context.Context, dbId uint64, tables []string) ([]string, error)

	// AssertStatementAllowed 语句级入口：在表判定之外多一个「表引用是否可信」的维度。
	//
	// resolvable=false 表示这条语句的真实表引用无法静态确定（解析失败、派生表、
	// INSERT…SELECT、条件子查询）。此时只要该库配了规则就拒绝执行——
	// **空表清单不能等于放行**：那正是 `SELECT * FROM (SELECT * FROM t_secret) x`
	// 绕过保护的那条路。
	AssertStatementAllowed(ctx context.Context, dbId uint64, tables []string, resolvable bool, refs []ColumnRef) error
}

// AssertDumpTablesAllowed 导出前的表/列权限判定（**唯一入口**）。
//
// 抽成显式函数而不是在调用点直接 acl.AssertWholeTableAllowed，原因有两个：
//  1. 它是"任何导出形态（含仅结构）都必须先过"的硬边界，写成单点后源码顺序守卫才有明确锚点；
//  2. acl 可作为参数注入，行为测试不需要污染全局 IOC 容器。
//
// acl == nil 表示该库未配置权限（放行），这与既有 GetTableAclApp 可能在未装配时返回 nil 的
// 现状一致；配置了规则时由实现给出拒绝原因（整表导出会把无权列一起带出，只能整条拒绝）。
func AssertDumpTablesAllowed(ctx context.Context, acl TableAcl, dbId uint64, tables []string) error {
	if acl == nil {
		return nil
	}
	// 表级（deny 名单 / 白名单）与列级是两道独立闸门：AssertWholeTableAllowed 只查列规则
	// （见 column_acl.go 的 DeniedColumns），不碰 rules/allows。只调后者会让「只配了表级保护
	// 的表」在导出/下载通道上被放行 —— 复制表路径一直是这样两个都调的。
	if err := acl.AssertTablesAllowed(ctx, dbId, tables); err != nil {
		return err
	}
	return acl.AssertWholeTableAllowed(ctx, dbId, tables)
}

func GetTableAclApp() TableAcl {
	return ioc.Get[TableAcl]()
}

// NewTableAclApp 供父包 IOC 注册（构造与注册分离：本子包不反向依赖父包）
func NewTableAclApp() TableAcl {
	return &tableAclAppImpl{}
}

type tableAclAppImpl struct {
	base.AppImpl[*entity.DbTableAcl, repository.DbTableAcl]

	roleApp sysapp.Role `inject:"T"`

	columnAclRepo repository.DbColumnAcl `inject:"T"`

	mu      sync.RWMutex
	cache   map[uint64]*tableAclSnapshot
	expires map[uint64]time.Time
}

var _ TableAcl = (*tableAclAppImpl)(nil)

// tableAclSnapshot 某库的规则快照：
//
//	rules   表名 → 生效角色 id
//	columns 表名 → 列名 → 生效角色 id
type tableAclSnapshot struct {
	rules map[string][]uint64
	// allows 白名单：该库出现任何一条 allow 规则就切到「默认拒绝」模式
	allows  map[string][]uint64
	columns map[string]map[string][]uint64
}

// empty 报告该库是否一条规则都没有（此时判定直接放行，不做角色查询）
func (s *tableAclSnapshot) empty() bool {
	return s == nil || (len(s.rules) == 0 && len(s.allows) == 0 && len(s.columns) == 0)
}

// whitelistMode 该库是否处于白名单模式（存在 allow 规则）
func (s *tableAclSnapshot) whitelistMode() bool {
	return s != nil && len(s.allows) > 0
}

// allowedBy 该表是否在允许清单中且当前账号命中
func (s *tableAclSnapshot) allowedBy(accountRoles []uint64, rolesUnresolved bool, table string) bool {
	roleIds, ok := s.allows[aclTableKey(table)]
	if !ok {
		return false
	}
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

// hasTableRules 该库是否配了表级规则（列级与表级是两套合并判定，不能相互替代）
func (s *tableAclSnapshot) hasTableRules() bool {
	return s != nil && (len(s.rules) > 0 || len(s.allows) > 0)
}

// aclTableKey 把表名归一成判定用的 key：去引号、取最后一段、转小写。
//
// 三处差异都会让「配了却拦不住」：
//   - DDL 的表名由正则从语句文本取，可能带反引号或双引号（ALTER TABLE `t_order`）；
//   - 带 schema 的语句（SELECT * FROM public.t_order）与只写表名的配置对不上；
//   - 大小写随方言与建表方式漂移。
//
// 因此两侧都归一到「最后一段 + 小写」：宁可把不同 schema 的同名表一起挡住（更严），
// 也不要出现「换个写法就绕过」。
func aclTableKey(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.Trim(name, " \t`\"'[]")
	return strings.ToLower(name)
}

// deniedBy 返回第一张被挡住的表名；空串表示放行。
//
// accountRoles 为空切片表示「账号没有任何角色」：此时 RoleIds 为空的规则仍然命中
// （它是「所有人」），非空规则不命中。
//
// rolesUnresolved 表示「连账号是谁都没拿到」（例如调用链上没有登录态）。此时角色维度
// 的规则按命中处理：证明不了有权，就不放行——否则「绕过登录态调用」等于一键摘掉保护。
func (s *tableAclSnapshot) deniedBy(accountRoles []uint64, rolesUnresolved bool, tables []string) string {
	if s.empty() {
		return ""
	}
	for _, table := range tables {
		roleIds, ok := s.rules[aclTableKey(table)]
		if !ok {
			continue
		}
		if len(roleIds) == 0 {
			return table
		}
		if rolesUnresolved {
			return table
		}
		for _, roleId := range roleIds {
			for _, owned := range accountRoles {
				if roleId == owned {
					return table
				}
			}
		}
	}
	return ""
}

// snapshot 取某库的规则快照（带 TTL 缓存）
func (a *tableAclAppImpl) snapshot(ctx context.Context, dbId uint64) (*tableAclSnapshot, error) {
	a.mu.RLock()
	if snap, ok := a.cache[dbId]; ok && time.Now().Before(a.expires[dbId]) {
		a.mu.RUnlock()
		return snap, nil
	}
	a.mu.RUnlock()

	rules, err := a.Repo.ListByDb(dbId)
	if err != nil {
		return nil, err
	}
	columnRules, err := a.columnAclRepo.ListByDb(dbId)
	if err != nil {
		return nil, err
	}
	snap := &tableAclSnapshot{
		rules:   make(map[string][]uint64, len(rules)),
		allows:  make(map[string][]uint64),
		columns: make(map[string]map[string][]uint64, len(columnRules)),
	}
	for _, rule := range columnRules {
		tableKey, columnKey := aclTableKey(rule.Table), aclTableKey(rule.Column)
		if tableKey == "" || columnKey == "" {
			continue
		}
		if snap.columns[tableKey] == nil {
			snap.columns[tableKey] = make(map[string][]uint64)
		}
		snap.columns[tableKey][columnKey] = append(snap.columns[tableKey][columnKey], rule.RoleIds...)
	}
	for _, rule := range rules {
		name := aclTableKey(rule.Table)
		if name == "" {
			continue
		}
		// 同一张表配了多条（历史数据/并发写入）时合并角色：任一规则命中即生效
		if rule.Effect == entity.DbTableAclEffectAllow {
			snap.allows[name] = append(snap.allows[name], rule.RoleIds...)
			continue
		}
		snap.rules[name] = append(snap.rules[name], rule.RoleIds...)
	}

	a.mu.Lock()
	if a.cache == nil {
		a.cache = make(map[uint64]*tableAclSnapshot)
		a.expires = make(map[uint64]time.Time)
	}
	a.cache[dbId] = snap
	a.expires[dbId] = time.Now().Add(aclSnapshotTtl)
	a.mu.Unlock()
	return snap, nil
}

// invalidate 立即失效某库的快照（保存/删除后调用，避免本实例上还要等 TTL）
func (a *tableAclAppImpl) invalidate(dbId uint64) {
	a.mu.Lock()
	delete(a.cache, dbId)
	delete(a.expires, dbId)
	a.mu.Unlock()
}

// accountRoleIds 当前账号的角色 id 列表
func (a *tableAclAppImpl) accountRoleIds(accountId uint64) ([]uint64, error) {
	if accountId == 0 {
		return nil, nil
	}
	roles, err := a.roleApp.GetAccountRoles(accountId)
	if err != nil {
		return nil, err
	}
	roleIds := make([]uint64, 0, len(roles))
	for _, role := range roles {
		roleIds = append(roleIds, role.RoleId)
	}
	return roleIds, nil
}

// HasRules 该库是否配置了保护规则
func (a *tableAclAppImpl) HasRules(ctx context.Context, dbId uint64) (bool, error) {
	if dbId == 0 {
		return false, nil
	}
	snap, err := a.snapshot(ctx, dbId)
	if err != nil {
		return false, err
	}
	return !snap.empty(), nil
}

// AssertTablesAllowed 表级权限判定：命中黑名单即拒绝。
//
// 这是所有数据面入口的公共闸门，任何入口都不要自己再判一套：出现第二套口径时
// 「编辑器被拦、导出放行」这类偏差从日志里都看不出来。
func (a *tableAclAppImpl) AssertTablesAllowed(ctx context.Context, dbId uint64, tables []string) error {
	if dbId == 0 || len(tables) == 0 {
		return nil
	}
	snap, err := a.snapshot(ctx, dbId)
	if err != nil {
		// 规则读不出来时不能当成「没有规则」：那正是权限被摘掉的路径
		return err
	}
	if !snap.hasTableRules() {
		return nil
	}
	la := contextx.GetLoginAccount(ctx)
	var accountId uint64
	if la != nil {
		accountId = la.Id
	}
	roleIds, err := a.accountRoleIds(accountId)
	if err != nil {
		logx.ErrorfContext(ctx, "table acl: failed to load account roles, denying by default: %s", err.Error())
		return errorx.NewBizI(ctx, imsg.ErrTableAclRolesUnavailable)
	}
	if table := snap.deniedBy(roleIds, accountId == 0, tables); table != "" {
		return errorx.NewBizI(ctx, imsg.ErrTableAccessDenied, "table", table)
	}
	// 白名单模式：deny 规则优先，其次「不在这份清单里就拒绝」。
	// 只有配了 allow 规则才进入这一支，避免「没配时把全库挡住」
	if snap.whitelistMode() {
		for _, table := range tables {
			if !snap.allowedBy(roleIds, accountId == 0, table) {
				return errorx.NewBizI(ctx, imsg.ErrTableNotAllowed, "table", table)
			}
		}
	}
	return nil
}

// ListByDb 某库的全部规则
func (a *tableAclAppImpl) ListByDb(dbId uint64) ([]*entity.DbTableAcl, error) {
	return a.Repo.ListByDb(dbId)
}

// SaveAcl 保存（按「库 + 表名忽略大小写」去重后更新，等价于 upsert）
func (a *tableAclAppImpl) SaveAcl(ctx context.Context, acl *entity.DbTableAcl) error {
	acl.Table = strings.TrimSpace(acl.Table)
	if acl.Effect != entity.DbTableAclEffectAllow {
		// 只接受两种取值；零值/非法值一律归为「禁止」，避免因脏数据意外切到白名单
		acl.Effect = entity.DbTableAclEffectDeny
	}
	if acl.DbId == 0 || acl.Table == "" {
		return errorx.NewBizI(ctx, imsg.ErrTableAclTableRequired)
	}
	if acl.Id > 0 {
		// 带 id 即「编辑已有规则」：按 id 更新，改名也该跟着改这一行，
		// 而不是新增一行把旧表名继续留在名单里（那会变成两条谁也说不清关系的规则）
		if err := a.Repo.UpdateById(ctx, acl, "table_name", "effect", "role_ids", "remark"); err != nil {
			return err
		}
		a.invalidate(acl.DbId)
		return nil
	}
	rules, err := a.Repo.ListByDb(acl.DbId)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		if !strings.EqualFold(rule.Table, acl.Table) {
			continue
		}
		// 已存在则更新角色与备注，而不是再插一行：同一张表两条规则会让「谁生效」
		// 取决于查询顺序，界面上也看不出哪条管用
		rule.Effect = acl.Effect
		rule.RoleIds = acl.RoleIds
		rule.Remark = acl.Remark
		if err := a.Repo.UpdateById(ctx, rule, "effect", "role_ids", "remark"); err != nil {
			return err
		}
		a.invalidate(acl.DbId)
		return nil
	}
	if err := a.Repo.Insert(ctx, acl); err != nil {
		return err
	}
	a.invalidate(acl.DbId)
	return nil
}

// DeleteAcl 删除规则
func (a *tableAclAppImpl) DeleteAcl(ctx context.Context, id uint64) error {
	acl, err := a.Repo.GetById(id)
	if err != nil {
		return err
	}
	if err := a.Repo.DeleteById(ctx, id); err != nil {
		return err
	}
	a.invalidate(acl.DbId)
	return nil
}

// VisibleTables 过滤不可见的表：deny 命中，或白名单模式下未命中 allow
func (a *tableAclAppImpl) VisibleTables(ctx context.Context, dbId uint64, tables []string) ([]string, error) {
	if dbId == 0 || len(tables) == 0 {
		return tables, nil
	}
	snap, err := a.snapshot(ctx, dbId)
	if err != nil {
		// 规则读不出来不能当成「没有规则」：那正是保护被摘掉的路径
		return nil, err
	}
	if snap.empty() {
		return tables, nil
	}
	roleIds, resolved, err := a.accountRoles(ctx)
	if err != nil {
		return nil, err
	}
	kept := make([]string, 0, len(tables))
	for _, table := range tables {
		if snap.deniedBy(roleIds, !resolved, []string{table}) != "" {
			continue
		}
		if snap.whitelistMode() && !snap.allowedBy(roleIds, !resolved, table) {
			continue
		}
		kept = append(kept, table)
	}
	return kept, nil
}

// AssertStatementAllowed 语句级判定：先看「表引用可信度」，再走既有的表级与列级判定。
//
// 为什么必须在语句级而不是各入口自己判：绕过面恰恰藏在解析结果里——
// 派生表在 AST 里只是一个名为 "(SELECT ...)" 的表引用，INSERT…SELECT 只有写入目标，
// 条件子查询根本不在表清单里。各入口拿到的 tables 看起来「合法且为空」，直接放行就是漏洞。
func (a *tableAclAppImpl) AssertStatementAllowed(ctx context.Context, dbId uint64, tables []string, resolvable bool, refs []ColumnRef) error {
	if dbId == 0 {
		return nil
	}
	if !resolvable {
		snap, err := a.snapshot(ctx, dbId)
		if err != nil {
			// 规则读不出来时不能当成「没有规则」：那正是权限被摘掉的路径
			return err
		}
		// 只在真的配了规则时才拒绝：没配规则的库保持既有行为，
		// 否则解析失败会把整库查询挡死，而不是挡住越权
		if snap.hasTableRules() || len(snap.columns) > 0 {
			return errorx.NewBizI(ctx, imsg.ErrAclUnresolvable)
		}
		return nil
	}
	if err := a.AssertTablesAllowed(ctx, dbId, tables); err != nil {
		return err
	}
	return a.AssertColumnsAllowed(ctx, dbId, tables, refs)
}
