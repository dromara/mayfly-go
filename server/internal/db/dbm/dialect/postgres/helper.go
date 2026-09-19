package postgres

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"regexp"
	"strings"
)

var (
	// 提取pg默认值的字面量部分，如：'id'::varchar  提取 'id'  ；  '-1'::integer  提取 '-1'
	// 默认值本身可能含单引号（内部双写），需支持成对双写引号
	defaultValueRegexp = regexp.MustCompile(`'((?:[^']|'')*)'`)
)

// FixColumnDefault 列默认值只剥去 ::type 的cast后缀，保留字面量的书写引号：
//
// 旧实现把引号一并还原为原始值，导致两类静默丢失：
//   - 空串默认值经还原后为空串，与“无默认值”完全不可区分，迁建表时默认值丢失；
//   - '(0)'、'unknown (pending)' 这类内容含括号的默认值，与表达式默认值不可区分，
//     被“含左括号即视为函数”的规则丢弃。
//
// 保留字面量形态后，是否为字符串默认值可由后续SQLGenerator精确判定（首尾成对单引号）
func FixColumnDefault(column *dbi.Column) {
	if column.ColumnDefault == "" {
		return
	}
	// 自增序列取值表达式：目标库靠AutoIncrement建自增列，不能将其当默认值写出（既非法也无意义）
	if strings.HasPrefix(column.ColumnDefault, "nextval") {
		column.IsExprDefault = true
		return
	}
	// 显式DEFAULT NULL在pg元数据中被呈现为 NULL::type（varchar/numeric等类型），
	// 其语义是无默认值；若不归一为空，残留的cast文本会被当作字符串默认值写入目标表，
	// 使省略该列的插入静默得到 "NULL::character varying" 这类垃圾值
	if strings.EqualFold(column.ColumnDefault, "NULL") || strings.HasPrefix(strings.ToUpper(column.ColumnDefault), "NULL::") {
		column.ColumnDefault = ""
		return
	}
	if !strings.Contains(column.ColumnDefault, "::") {
		// 无cast后缀不等于无默认值：gen_random_uuid()这类裸函数调用就是表达式默认值，
		// 必须继续标记，否则会被当字面量写入目标列
		dbi.MarkExprDefault(column)
		return
	}
	// 提取最左且完整的带引号字面量（内部单引号双写），丢弃其后的::cast及其余包装
	if match := defaultValueRegexp.FindString(column.ColumnDefault); match != "" {
		column.ColumnDefault = match
	}
	// pg的字面量默认值恒带引号，不带引号又呈函数/运算形态的必为表达式默认值（如uuid列的gen_random_uuid()）：
	// 标记后统一不输出，旧逻辑会写成 DEFAULT 'gen_random_uuid()'，uuid/jsonb列建表即报非法输入，
	// text列则静默把函数名当成默认值内容
	dbi.MarkExprDefault(column)
}

// pg的函数默认值外层cast无需在此处理：实测pg会自行消除无意义cast（now()::timestamp回显为now()），
// 仍残留cast文本的只能是无法跨库还原的表达式

var _ dbi.DumpHelper = (*DumpHelper)(nil)

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

// pg导入方（transfer2Db/ExecReader）已在自身事务内逐条执行，脚本内的BEGIN/COMMIT语句
// 会提交/破坏外层事务（报 unexpected transaction status idle），故不输出，与sqlite/mssql保持一致
func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	// 设置自增序列当前值
	for _, column := range columns {
		if column.AutoIncrement {
			// 表名/列名作为标识符必须引用（并双写内部引用符），直接拼进 "%s" 会使含特殊字符的表名语法错误；
			// 序列名处于字符串字面量内，但其内容会被pg再当对象名解析一次，故先按标识符引用再按字符串转义
			seqIdent := dbi.DefaultQuoter.QuoteIdent(fmt.Sprintf("%s_%s_seq", tableName, column.ColumnName))
			// 校正只增不减：setval立即生效且不随事务回滚，故序列终值等于**最后一次执行**的校正写入值。
			// 大表按主键分片并行迁移时每段尾部都带一句校正，其max子查询只能读到本分片+他人已提交的数据，
			// 直接写入读到的值会让“后完成但id更小”的分片把序列下调（实测300行/8分片并发下终值228而全表最大300），
			// 迁移后首条业务插入即撞主键；与序列当前值取GREATEST后，终值恒不低于全表最大id
			maxExpr := fmt.Sprintf("(SELECT max(%s) FROM %s)", dbi.DefaultQuoter.QuoteIdent(column.ColumnName), dbi.DefaultQuoter.QuoteIdent(tableName))
			// 空表时max为NULL，而GREATEST会忽略NULL取到序列当前值并把is_called置true（首条插入平白跳号），故用WHERE整体跳过
			seq := fmt.Sprintf("SELECT setval('%s', GREATEST(%s, (SELECT last_value FROM %s)), true) WHERE %s IS NOT NULL;\n",
				dbi.QuoteEscape(seqIdent), maxExpr, seqIdent, maxExpr)
			if _, err := writer.Write([]byte(seq)); err != nil {
				return err
			}
		}
	}

	return nil
}
