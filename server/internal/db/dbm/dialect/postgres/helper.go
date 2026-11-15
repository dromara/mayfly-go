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
// 若把引号一并还原为原始值，会导致两类静默丢失：
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
	// 标记后统一不输出，否则会写成 DEFAULT 'gen_random_uuid()'，uuid/jsonb列建表即报非法输入，
	// text列则静默把函数名当成默认值内容
	dbi.MarkExprDefault(column)
}

// pg的函数默认值外层cast无需在此处理：实测pg会自行消除无意义cast（now()::timestamp回显为now()），
// 仍残留cast文本的只能是无法跨库还原的表达式

var _ dbi.DumpTxnWrapper = (*DumpTxnWrapper)(nil)

type DumpTxnWrapper struct {
	dbi.DefaultDumpTxnWrapper
}

// pg导入方（transfer2Db/ExecReader）已在自身事务内逐条执行，脚本内的BEGIN/COMMIT语句
// 会提交/破坏外层事务（报 unexpected transaction status idle），故不输出，与sqlite/mssql保持一致
func (dh *DumpTxnWrapper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

func (dh *DumpTxnWrapper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	// 设置自增序列当前值
	for _, column := range columns {
		if column.AutoIncrement {
			// 表名/列名作为标识符必须引用（并双写内部引用符），直接拼进 "%s" 会使含特殊字符的表名语法错误；
			// 序列名处于字符串字面量内，但其内容会被pg再当对象名解析一次，故先按标识符引用再按字符串转义
			seqIdent := dbi.DefaultQuoter.QuoteIdent(fmt.Sprintf("%s_%s_seq", tableName, column.ColumnName))
			// 校正只增不减：setval立即生效且不随事务回滚，故序列终值等于**最后一次执行**的校正写入值。
			// 大表按主键分片并行迁移时每段尾部都带一句校正，其max子查询只能读到本分片+他人已提交的数据，
			// 直接写入读到的值会让“后完成但id更小”的分片把序列下调（实测300行/8分片并发下终值228而全表最大300），
			// 迁移后首条业务插入即撞主键。
			maxExpr := fmt.Sprintf("(SELECT max(%s) FROM %s)", dbi.DefaultQuoter.QuoteIdent(column.ColumnName), dbi.DefaultQuoter.QuoteIdent(tableName))
			// 与序列当前值取GREATEST还不够：读当前值与写回不是一个原子动作，并发下仍是丢更新。
			// 本机 pg16 实测：B 先读到 last_value=1，A 随即 setval(100) 并提交，B 后执行 setval 时
			// 写的是 GREATEST(50, 1)=50 → 终值 50 低于全表最大 100（旧文案认为“后执行的校正必
			// 然看到先执行推高的当前值”，忽略了它是在对方写入**之前**读的那一份）。
			// 因此用事务级咨询锁把「读当前值→写回」围成一个临界区：pg_advisory_xact_lock 持锁到本事务
			// 结束，并发的另一句只能等它提交后再读，读到的必然不低于对方刚写入的值。各句写入值又不低于
			// 自己的 max，故终值恒不低于全表最大 id。
			// 锁必须走在读之前：这里靠 LATERAL 造出真实的数据依赖（侧向子查询要求左侧先产出行），
			// 不用 CASE/WHERE 里的求值顺序——实测 pg_advisory_xact_lock 返回 void 且 `void IS NULL` 为 false，
			// 拿它做 CASE 条件会走 ELSE 分支把 setval 写成 NULL（静默不校正）。
			lockedNow := fmt.Sprintf("(SELECT _v FROM (SELECT pg_advisory_xact_lock(hashtext('%s'))) _lock, LATERAL (SELECT last_value AS _v FROM %s) _now)",
				dbi.QuoteEscape(seqIdent), seqIdent)
			// 空表时max为NULL，而GREATEST会忽略NULL取到序列当前值并把is_called置true（首条插入平白跳号），故用WHERE整体跳过
			seq := fmt.Sprintf("SELECT setval('%s', GREATEST(%s, %s), true) WHERE %s IS NOT NULL;\n",
				dbi.QuoteEscape(seqIdent), maxExpr, lockedNow, maxExpr)
			if _, err := writer.Write([]byte(seq)); err != nil {
				return err
			}
		}
	}

	return nil
}
