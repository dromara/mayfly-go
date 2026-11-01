package application

// 敏感数据发现：扫描库里所有列，按「列名 + 类型」给出可能敏感的判断，
// 并标注是否已被列保护覆盖。
//
// 定位是**发现**而不是判定：规则命中只说明「这列看起来像手机号」，最终要不要保护由人决定。
// 因此结论里带上命中依据（类别），而不是直接自动建规则——自动建规则会在一张宽表上
// 一次挡住几十列，运维第二天就把它整体关掉。
//
// 与既有能力的关系：
//   - 列保护（t_db_column_acl）负责「这一列不给看」，本功能标出「建议纳入」；
//   - 脱敏（mask-rules）负责「看得到但值被掩码」，本功能不重复判断已有脱敏规则，
//     因为脱敏规则是全局列名模式匹配，判断结果会随策略变化而漂移，误标比不标更糟。

import (
	"context"
	"regexp"
	"strings"

	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/pkg/logx"
)

// 扫描的表数上限：一次扫描要按表取列元数据，超大库（数千表）会把一次点击变成
// 几分钟的抖动。截断并如实回报「已扫描表数」，比让用户等一个不确定的时长更好。
const sensitiveScanMaxTables = 200

// SensitiveColumnFinding 一条敏感列发现
type SensitiveColumnFinding struct {
	Table  string `json:"tableName"`
	Column string `json:"columnName"`
	Type   string `json:"columnType"`
	// Level high / medium
	Level string `json:"level"`
	// Category 命中类别（phone/idcard/credential/bankcard/email/name/address/other），
	// 前端按类别出文案：后端不下发中文，避免同一句话两份真源
	Category string `json:"category"`
	// Protected 该列是否已在列保护名单中
	Protected bool `json:"protected"`
}

// SensitiveScanResult 扫描结果
type SensitiveScanResult struct {
	Findings []*SensitiveColumnFinding `json:"findings"`
	// ScannedTables 本次实际扫描的表数
	ScannedTables int `json:"scannedTables"`
	// TotalTables 库里的表数（> ScannedTables 表示被上限截断）
	TotalTables int `json:"totalTables"`
	// Truncated 是否因表数上限被截断
	Truncated bool `json:"truncated"`
}

// sensitivePattern 列名 → 敏感类别与级别
type sensitivePattern struct {
	category string
	level    string
	// tokens 该类别的敏感词（列名分段），用于「数值类型列只认末段」的收紧判据
	tokens []string
	re     *regexp.Regexp
}

// 判据只用列名：注释与类型不足以区分「金额」与「手机号」，
// 而列名在各库里都是同一批命名习惯（phone/mobile/id_card/...）。
var sensitivePatterns = []sensitivePattern{
	{"credential", "high", []string{"password", "passwd", "pwd", "secret", "token", "private_key", "access_key", "salt"},
		regexp.MustCompile(`(?i)(^|_)(password|passwd|pwd|secret|token|private_?key|access_?key|salt)(_|$)`)},
	{"idcard", "high", []string{"id_card", "id_no", "identity", "id_number", "passport"},
		regexp.MustCompile(`(?i)(^|_)(id_?card|id_?no|identity|id_?number|passport)(_|$)`)},
	{"bankcard", "high", []string{"bank_card", "card_no", "credit_card", "iban", "account_no"},
		regexp.MustCompile(`(?i)(^|_)(bank_?card|card_?no|credit_?card|iban|account_?no)(_|$)`)},
	{"phone", "high", []string{"phone", "mobile", "telephone", "tel"},
		regexp.MustCompile(`(?i)(^|_)(phone|mobile|telephone|tel)(_|$)`)},
	{"email", "medium", []string{"email", "mail"},
		regexp.MustCompile(`(?i)(^|_)(email|mail)(_|$)`)},
	{"name", "medium", []string{"real_name", "full_name", "customer_name", "user_name", "contact_name", "username"},
		regexp.MustCompile(`(?i)(^|_)(real_?name|full_?name|customer_?name|user_?name|contact_?name)(_|$)|(^|_)username(_|$)`)},
	{"address", "medium", []string{"address", "addr", "street", "zip_code", "postcode"},
		regexp.MustCompile(`(?i)(^|_)(address|addr|street|zip_?code|postcode)(_|$)`)},
	{"other", "medium", []string{"wechat", "weixin", "qq", "birthday", "birth_date", "ssn", "ip_addr"},
		regexp.MustCompile(`(?i)(^|_)(wechat|weixin|qq|birthday|birth_?date|ssn|ip_?addr)(_|$)`)},
}

// isNumericType 报告列类型是否为数值类型（用于收紧判据）
func isNumericType(columnType string) bool {
	lower := strings.ToLower(columnType)
	for _, token := range []string{"int", "decimal", "numeric", "float", "double", "real", "serial", "number", "money"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

// classifySensitiveColumn 命中第一个匹配的类别；未命中返回空。
//
// 数值列收紧：只有「末段就是敏感词」才算（电话存成 bigint、卡号存成 bigint 都能认出来），
// 而 `token_budget`、`secret_count` 这类**以敏感词开头**的计数/金额/预算字段不算——
// 它们是 A 加密、A 计数这类业务列，混进结论会让人不再相信整份扫描。
func classifySensitiveColumn(column string, columnType string) (category string, level string) {
	name := strings.ToLower(strings.TrimSpace(column))
	for _, pattern := range sensitivePatterns {
		if !pattern.re.MatchString(name) {
			continue
		}
		if isNumericType(columnType) && !endsWithToken(pattern.tokens, name) {
			continue
		}
		return pattern.category, pattern.level
	}
	return "", ""
}

// endsWithToken 报告列名是否「就是该敏感词」或以「_敏感词」结尾。
//
// 用后缀而不是末段：card_no、id_no、access_key 这类复合敏感词本身带下划线，
// 取末段会退化成 no/key，把它们全漏掉。
func endsWithToken(tokens []string, column string) bool {
	for _, token := range tokens {
		if column == token || strings.HasSuffix(column, "_"+token) {
			return true
		}
	}
	return false
}

// ScanSensitiveColumns 扫描指定库的敏感列候选
func (d *dbAppImpl) ScanSensitiveColumns(ctx context.Context, dbId uint64, dbName string) (*SensitiveScanResult, error) {
	conn, err := d.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}

	tables, err := conn.Metadata().GetTables()
	if err != nil {
		return nil, err
	}
	result := &SensitiveScanResult{Findings: make([]*SensitiveColumnFinding, 0, 32), TotalTables: len(tables)}
	if len(tables) > sensitiveScanMaxTables {
		tables = tables[:sensitiveScanMaxTables]
		result.Truncated = true
	}
	result.ScannedTables = len(tables)

	// 已受保护的列：一次取库级规则，避免逐列回表
	protected := make(map[string]struct{})
	columnRules, err := sqlexec.GetTableAclApp().ListColumns(dbId)
	if err != nil {
		// 规则读不出来不影响「发现」本身，但要在日志里留痕：
		// 否则界面上所有列都显示「未保护」，运维会重复配一遍
		logx.WarnfContext(ctx, "sensitive scan: failed to load column acl rules, all findings marked unprotected: %s", err.Error())
	}
	for _, rule := range columnRules {
		protected[strings.ToLower(rule.Table)+"."+strings.ToLower(rule.Column)] = struct{}{}
	}

	for _, table := range tables {
		columns, colErr := conn.Metadata().GetColumns(table.TableName)
		if colErr != nil {
			// 单表取列失败不终止整次扫描：权限不足或视图定义无效的表会一直失败，
			// 让整次扫描连带失败等于「一张坏表挡住整个库的发现」
			logx.WarnfContext(ctx, "sensitive scan: failed to read columns of table %s: %s", table.TableName, colErr.Error())
			continue
		}
		for _, column := range columns {
			category, level := classifySensitiveColumn(column.ColumnName, column.ColumnType)
			if category == "" {
				continue
			}
			_, isProtected := protected[strings.ToLower(column.TableName)+"."+strings.ToLower(column.ColumnName)]
			result.Findings = append(result.Findings, &SensitiveColumnFinding{
				Table:     column.TableName,
				Column:    column.ColumnName,
				Type:      column.ColumnType,
				Level:     level,
				Category:  category,
				Protected: isProtected,
			})
		}
	}
	return result, nil
}
