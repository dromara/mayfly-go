/**
 * 分页查询片段模板预设
 *
 * 各家限行语法差异大，与 getPageSql / getPreviewSql 同属「分页写法」这一方言事实，
 * 故预设放在方言层，由各方言的 getPageSnippet() 自行挑选，补全层不再维护
 * dbType → 模板 的平行映射表（那种表在新增方言漏登记时只会静默退化，不报错）。
 *
 * body 为 monaco snippet 语法：`${n:default}` 为第 n 个 Tab 占位，`$0` 为最终光标位。
 * 本文件是纯数据，零依赖，可被任意方言安全导入。
 */

import type { SqlSnippetTemplate } from '../types';

/** 通用 SELECT 骨架：表名/条件/排序占 ${1..3}，${4..} 留给限行子句 */
const SELECT_SKELETON = 'SELECT *\nFROM ${1:table}\nWHERE ${2:condition}\nORDER BY ${3:id}\n';

/**
 * `LIMIT offset, count`（MySQL 系逗号形态）
 * 适用：mysql / mariadb / sqlite / clickhouse（四者均接受该写法）
 */
export const limitCommaPageSnippet: SqlSnippetTemplate = {
    label: 'SELECT PAGE (LIMIT)',
    description: 'limit pagination snippet',
    body: SELECT_SKELETON + 'LIMIT ${4:0}, ${5:25}\n$0',
};

/**
 * `LIMIT count OFFSET offset`（标准 SQL 形态）
 * 适用：postgres 及其兼容方言（gauss / kingbaseEs / vastbase）
 */
export const limitOffsetPageSnippet: SqlSnippetTemplate = {
    label: 'SELECT PAGE (OFFSET)',
    description: 'offset pagination snippet',
    body: SELECT_SKELETON + 'LIMIT ${4:25} OFFSET ${5:0}\n$0',
};

/**
 * `OFFSET n ROWS FETCH NEXT m ROWS ONLY`（T-SQL，要求语句已有 ORDER BY）
 * 适用：mssql
 */
export const offsetFetchPageSnippet: SqlSnippetTemplate = {
    label: 'SELECT PAGE (FETCH)',
    description: 'offset fetch pagination snippet',
    body: SELECT_SKELETON + 'OFFSET ${4:0} ROWS FETCH NEXT ${5:25} ROWS ONLY\n$0',
};

/**
 * ROWNUM 伪列双层子查询（Oracle 12c 以前无 OFFSET/FETCH）
 * 适用：oracle / dm。骨架与其余三者不同，故整体独立给出。
 */
export const rownumPageSnippet: SqlSnippetTemplate = {
    label: 'SELECT PAGE (ROWNUM)',
    description: 'rownum pagination snippet',
    body: 'SELECT * FROM (\n  SELECT t.*, ROWNUM AS rn\n  FROM ${1:table} t\n  WHERE ROWNUM <= ${2:25}\n)\nWHERE rn > ${3:0}\n$0',
};
