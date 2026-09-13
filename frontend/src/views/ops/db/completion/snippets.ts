/**
 * SQL 片段模板（对齐国际同类产品的 snippet 能力）。
 *
 * body 使用 monaco snippet 语法：`${1:placeholder}` 占位、Tab 逐项跳转、`$0` 最终光标位。
 *
 * 职责边界：本文件只维护**方言无关**的通用模板（JOIN / INSERT 等）。
 * 与限行语法相关的分页模板属方言事实，由 `DbDialect.getPageSnippet()` 自描述
 * （预设见 `dialect/shared/snippets.ts`），故此处不存在 dbType → 模板 的映射表：
 * 新增方言时无需回到本文件登记，也不会因漏登记而静默退化为错误语法。
 */
import type { DbDialect, SqlSnippetTemplate } from '../dialect/types';

/** JOIN 查询模板（方言无关） */
const joinSnippet: SqlSnippetTemplate = {
    label: 'JOIN',
    description: 'join query snippet',
    body: 'SELECT ${1:t1.*}\nFROM ${2:table1} t1\nJOIN ${3:table2} t2 ON ${4:t1.id = t2.id}\nWHERE ${5:condition}\n$0',
};

/** INSERT 模板（方言无关） */
const insertSnippet: SqlSnippetTemplate = {
    label: 'INSERT',
    description: 'insert snippet',
    body: 'INSERT INTO ${1:table} (${2:columns})\nVALUES (${3:values})\n$0',
};

/** 方言无关的通用模板，任何方言都适用 */
const commonSnippets: SqlSnippetTemplate[] = [joinSnippet, insertSnippet];

/**
 * 获取指定方言的片段模板：通用模板 + 该方言的分页模板。
 *
 * @param dialect 数据库方言，由其自描述分页写法
 */
export function getSnippets(dialect: DbDialect): SqlSnippetTemplate[] {
    return [...commonSnippets, dialect.getPageSnippet()];
}
