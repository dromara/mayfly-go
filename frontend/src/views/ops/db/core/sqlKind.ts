/**
 * SQL 语句种类判定（与 DbInst 解耦的纯函数，供执行链路与缓存失效共用同一把尺子）。
 */

/**
 * 改变库结构的语句关键字：建/改/删表、清空、重命名，以及各方言的注释语句（comment on）
 * 与授权语句（grant/revoke）——注释与权限同样会改变补全要展示的内容（表/列注释来源）。
 */
const STRUCTURE_KEYWORDS = ['create', 'alter', 'drop', 'truncate', 'rename', 'comment', 'grant', 'revoke'];

/**
 * 剥离语句前导的空白与注释（`--`、`#` 行注释与块注释），返回首个真实语句起始处。
 *
 * 语句切割（无论是后端还是编辑器本地）会保留注释原文，而「按前缀字符判类型」在注释前会命中
 * 注释里的单词（如 `-- 说明\nDELETE FROM t`），故必须先跳过注释区域。迭代而非递归：
 * 粘贴的 dump 脚本可能带成百上千行注释头，递归会按注释行数加深调用栈。
 */
function stripLeadingComments(sql: string): string {
    let rest = sql.trimStart();
    for (;;) {
        if (rest.startsWith('--') || rest.startsWith('#')) {
            const end = rest.indexOf('\n');
            if (end === -1) {
                return '';
            }
            rest = rest.slice(end + 1).trimStart();
            continue;
        }
        if (rest.startsWith('/*')) {
            const end = rest.indexOf('*/');
            if (end === -1) {
                return '';
            }
            rest = rest.slice(end + 2).trimStart();
            continue;
        }
        return rest;
    }
}

/**
 * 是否为结构变更语句。
 *
 * 消费方是「执行成功后要不要失效本地元数据缓存」：结构变了却不失效，SQL 补全会继续给旧表清单/旧列。
 * 与 `DbInst.isQuerySql` 的「非查询」不是一回事——DML 也非查询但不改结构，故两个判定各自演进。
 *
 * 只看语句首词（整词匹配，非子串），因此认不出被 `EXEC`/存储过程包住的 DDL（如 mssql 的
 * `EXECUTE sp_addextendedproperty`）——这类方言特殊写法由其调用方（表编辑弹框）无条件失效兜底。
 */
export function isDdlSql(sql: string): boolean {
    const leading = stripLeadingComments(sql).toLowerCase();
    return STRUCTURE_KEYWORDS.some((keyword) => {
        if (!leading.startsWith(keyword)) {
            return false;
        }
        // 关键字后必须是边界，避免 create_x 之类的标识符前缀被误判
        const next = leading.charAt(keyword.length);
        return next === '' || !/[a-z0-9_$]/.test(next);
    });
}
