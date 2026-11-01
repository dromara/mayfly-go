package application

// DDL 预检：在把变更交出去之前，说清「这条语句会怎么疼」。
//
// 与触发策略的分工是刻意的：策略回答「要不要人批」，预检回答「批了之后大概会发生什么」
// （会不会丢数据、要不要重建整表、能不能回滚）。两者互补，预检**不参与放行判定**，
// 所以它是只读的、可以随便多点几次的。
//
// 语句切分由调用方完成（执行前本来就切好了），这里不重复造切割器；分类用关键字粗判，
// 因为预检是提示性输出，权威分类仍在 sqlparser 那条治理链路上。

import (
	"context"
	"regexp"
	"strings"

	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
)

// DdlRiskLevel 风险等级：danger 会造成不可逆的数据损失，warn 是耗时/兼容性风险
type DdlRiskLevel string

const (
	DdlRiskDanger DdlRiskLevel = "danger"
	DdlRiskWarn   DdlRiskLevel = "warn"
	DdlRiskInfo   DdlRiskLevel = "info"
)

// ddlLargeTableRows 大表阈值：超过它，任何需要重建表的变更都不该在业务高峰做。
// 与审核规则里的「大表 DDL」阈值取同一数量级，避免两处口径打架。
const ddlLargeTableRows = 1_000_000

// DdlFinding 一条预检发现；Args 交给前端插值（不把拼好的中文写到后端，便于 i18n）
type DdlFinding struct {
	Level DdlRiskLevel `json:"level"`
	Code  string       `json:"code"`
	Table string       `json:"table,omitempty"`
	Args  collx.M      `json:"args,omitempty"`
}

// DdlStatement 单条语句的预检视图
type DdlStatement struct {
	SQL    string   `json:"sql"`
	Kind   string   `json:"kind"`
	Tables []string `json:"tables"`
}

// DdlTableStat 参与本次变更的表规模（前端要能看见「你说它大，多大」）
type DdlTableStat struct {
	Name  string `json:"name"`
	Rows  int    `json:"rows"`
	Bytes int64  `json:"bytes"`
	HasPk bool   `json:"hasPk"`
}

// DdlPrecheckResult 预检结果
type DdlPrecheckResult struct {
	Statements []DdlStatement `json:"statements"`
	Tables     []DdlTableStat `json:"tables"`
	Findings   []DdlFinding   `json:"findings"`
}

// DdlPrecheckApp DDL 预检
type DdlPrecheckApp interface {
	// Precheck 对一组已切分的语句做只读预检（会连库读表规模与主键，不执行任何变更）
	Precheck(ctx context.Context, dbId uint64, dbName string, script string) (*DdlPrecheckResult, error)
}

func GetDdlPrecheckApp() DdlPrecheckApp {
	return ioc.Get[DdlPrecheckApp]()
}

type ddlPrecheckAppImpl struct {
	dbApp Db `inject:"T"`
}

var _ DdlPrecheckApp = (*ddlPrecheckAppImpl)(nil)

// ---------------------------------- 语句分类与目标表 ----------------------------------

var (
	reDropTable     = regexp.MustCompile(`(?is)^\s*drop\s+(?:temporary\s+)?table\s+(?:if\s+exists\s+)?([^\s,;]+)`)
	reTruncate      = regexp.MustCompile(`(?is)^\s*truncate\s+(?:table\s+)?([^\s,;]+)`)
	reAlterTable    = regexp.MustCompile(`(?is)^\s*alter\s+table\s+(?:if\s+exists\s+)?(?:only\s+)?([^\s,;]+)`)
	reCreateTable   = regexp.MustCompile(`(?is)^\s*create\s+(?:temporary\s+)?table\s+(?:if\s+not\s+exists\s+)?([^\s(,;]+)`)
	reCreateIndex   = regexp.MustCompile(`(?is)^\s*create\s+(?:unique\s+|fulltext\s+|spatial\s+)?index\s+[^\s]+\s+on\s+([^\s(,;]+)`)
	reRenameTable   = regexp.MustCompile(`(?is)^\s*rename\s+tables?\s+([^\s,;]+)`)
	reDropColumn    = regexp.MustCompile(`(?is)\bdrop\s+column\b`)
	reTypeChange    = regexp.MustCompile(`(?is)\b(modify|change|alter\s+column)\b`)
	reAddIndex      = regexp.MustCompile(`(?is)\badd\s+(?:unique\s+|fulltext\s+|spatial\s+|constraint\s+\S+\s+)?(?:index|key|primary\s+key)\b`)
	reDropIndexKind = regexp.MustCompile(`(?is)^\s*(drop|create|alter|truncate|rename)\b`)
)

// normalizeTableName 去掉引号与 schema 前缀引号：统计与展示都按裸表名比对
func normalizeTableName(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.Trim(name, "`\"[]")
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.Trim(name, "`\"[]")
}

// ddlKindOf 粗判语句类别；返回空串表示不是 DDL（预检只关心 DDL）
func ddlKindOf(sql string) (kind string, tables []string) {
	text := strings.TrimSpace(sql)
	switch {
	case reDropTable.MatchString(text):
		return "drop_table", []string{normalizeTableName(reDropTable.FindStringSubmatch(text)[1])}
	case reTruncate.MatchString(text):
		return "truncate_table", []string{normalizeTableName(reTruncate.FindStringSubmatch(text)[1])}
	case reRenameTable.MatchString(text):
		return "rename_table", []string{normalizeTableName(reRenameTable.FindStringSubmatch(text)[1])}
	case reAlterTable.MatchString(text):
		return "alter_table", []string{normalizeTableName(reAlterTable.FindStringSubmatch(text)[1])}
	case reCreateIndex.MatchString(text):
		return "create_index", []string{normalizeTableName(reCreateIndex.FindStringSubmatch(text)[1])}
	case reCreateTable.MatchString(text):
		return "create_table", []string{normalizeTableName(reCreateTable.FindStringSubmatch(text)[1])}
	case reDropIndexKind.MatchString(text):
		return "other_ddl", nil
	}
	return "", nil
}

// ddlPrecheck 纯函数：语句 + 表规模事实 → 发现清单（不连库，便于用例覆盖每条规则）
func ddlPrecheck(statements []DdlStatement, stats map[string]*DdlTableStat) []DdlFinding {
	findings := make([]DdlFinding, 0, len(statements))
	appendFinding := func(level DdlRiskLevel, code, table string, args collx.M) {
		findings = append(findings, DdlFinding{Level: level, Code: code, Table: table, Args: args})
	}

	for _, statement := range statements {
		switch statement.Kind {
		case "drop_table":
			appendFinding(DdlRiskDanger, "drop_table", firstTable(statement.Tables), nil)
		case "truncate_table":
			appendFinding(DdlRiskDanger, "truncate_table", firstTable(statement.Tables), nil)
		case "rename_table":
			appendFinding(DdlRiskWarn, "rename_table", firstTable(statement.Tables), nil)
		case "alter_table":
			if reDropColumn.MatchString(statement.SQL) {
				appendFinding(DdlRiskDanger, "drop_column", firstTable(statement.Tables), nil)
			}
			if reTypeChange.MatchString(statement.SQL) {
				appendFinding(DdlRiskWarn, "type_change", firstTable(statement.Tables), nil)
			}
			if reAddIndex.MatchString(statement.SQL) {
				appendFinding(DdlRiskWarn, "add_index", firstTable(statement.Tables), nil)
			}
		}

		// 规模类判定对所有 DDL 生效：大表上任何结构性变更都要有心理准备
		for _, table := range statement.Tables {
			stat := stats[table]
			if stat == nil {
				// 表不存在：可能是新建，也可能是名字写错——后者比前者常见，提一句
				if statement.Kind != "create_table" {
					appendFinding(DdlRiskInfo, "table_unknown", table, nil)
				}
				continue
			}
			if stat.Rows >= ddlLargeTableRows {
				appendFinding(DdlRiskWarn, "large_table", table, collx.M{"rows": stat.Rows})
			}
			if !stat.HasPk && statement.Kind == "alter_table" {
				appendFinding(DdlRiskWarn, "no_primary_key", table, nil)
			}
		}
	}
	return findings
}

func firstTable(tables []string) string {
	if len(tables) == 0 {
		return ""
	}
	return tables[0]
}

// ---------------------------------- 应用层 ----------------------------------

// Precheck 读表规模后给出预检结论
func (a *ddlPrecheckAppImpl) Precheck(ctx context.Context, dbId uint64, dbName string, script string) (*DdlPrecheckResult, error) {
	biz.IsTrue(strings.TrimSpace(script) != "", "no sql to precheck")
	sqls := splitTopLevelSql(script)
	biz.IsTrue(len(sqls) > 0, "no sql to precheck")

	statements := make([]DdlStatement, 0, len(sqls))
	tableNames := make([]string, 0, len(sqls))
	seen := make(map[string]bool, len(sqls))
	for _, sql := range sqls {
		kind, tables := ddlKindOf(sql)
		statements = append(statements, DdlStatement{SQL: sql, Kind: kind, Tables: tables})
		for _, table := range tables {
			if table != "" && !seen[table] {
				seen[table] = true
				tableNames = append(tableNames, table)
			}
		}
	}

	stats := make(map[string]*DdlTableStat, len(tableNames))
	tableStats := make([]DdlTableStat, 0, len(tableNames))
	if len(tableNames) > 0 {
		conn, err := a.dbApp.GetDbConn(ctx, dbId, dbName)
		if err != nil {
			return nil, err
		}
		tables, err := conn.Metadata().GetTables(tableNames...)
		if err != nil {
			// 读不到规模不等于没有风险：明确告诉调用方预检不完整，而不是给出「一切正常」
			logx.WarnfContext(ctx, "ddl precheck read tables of db [%d/%s] failed: %s", dbId, dbName, err.Error())
			return nil, errorx.NewBizI(ctx, imsg.ErrDdlPrecheckTables, "err", err.Error())
		}
		for _, table := range tables {
			stat := &DdlTableStat{Name: table.TableName, Rows: table.TableRows, Bytes: table.DataLength}
			if indexes, idxErr := conn.Metadata().GetTableIndex(table.TableName); idxErr == nil {
				for _, index := range indexes {
					if index.IsPrimaryKey {
						stat.HasPk = true
						break
					}
				}
			}
			stats[stat.Name] = stat
			tableStats = append(tableStats, *stat)
		}
	}

	return &DdlPrecheckResult{
		Statements: statements,
		Tables:     tableStats,
		Findings:   ddlPrecheck(statements, stats),
	}, nil
}

// splitTopLevelSql 按顶层分号切分脚本：引号与注释内的分号不切。
//
// 为什么不用治理链路里的 sqlparser：那条链路带着方言配置与大量边界处理，
// 而预检只需要「大致切成几条」；但切歪会导致目标表提取错乱，所以这里对
// 未闭合引号采取保守策略——整段当作一条，宁可少报也不误报。
func splitTopLevelSql(script string) []string {
	var (
		statements  []string
		current     strings.Builder
		quote       rune
		lineComment bool
		blockDepth  int
	)
	runes := []rune(script)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if lineComment {
			current.WriteRune(ch)
			if ch == '\n' {
				lineComment = false
			}
			continue
		}
		if blockDepth > 0 {
			if ch == '*' && i+1 < len(runes) && runes[i+1] == '/' {
				current.WriteRune(ch)
				current.WriteRune('/')
				i++
				blockDepth--
				continue
			}
			current.WriteRune(ch)
			continue
		}
		if quote != 0 {
			current.WriteRune(ch)
			// 识别 doubled quote（'' / "" / ``）作为转义
			if ch == quote {
				if i+1 < len(runes) && runes[i+1] == quote {
					current.WriteRune(runes[i+1])
					i++
					continue
				}
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
			current.WriteRune(ch)
		case '-':
			if i+1 < len(runes) && runes[i+1] == '-' {
				lineComment = true
				current.WriteRune(ch)
			} else {
				current.WriteRune(ch)
			}
		case '/':
			if i+1 < len(runes) && runes[i+1] == '*' {
				blockDepth = 1
				current.WriteRune(ch)
				current.WriteRune('*')
				i++
			} else {
				current.WriteRune(ch)
			}
		case ';':
			if statement := strings.TrimSpace(current.String()); statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}
	if quote != 0 || blockDepth > 0 {
		// 未闭合：整段当作一条，交回原文，避免切出半截语句
		if trimmed := strings.TrimSpace(script); trimmed != "" {
			return []string{trimmed}
		}
		return nil
	}
	if statement := strings.TrimSpace(current.String()); statement != "" {
		statements = append(statements, statement)
	}
	return statements
}
