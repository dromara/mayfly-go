/**
 * SQL 片段模板（对齐国际同类产品的 snippet 能力）。
 * body 使用 monaco snippet 语法：${1:placeholder} 占位、Tab 逐项跳转、$0 最终光标位。
 * 模板为纯数据，按方言索引 + 通用兜底，新增模板只改本文件。
 */
export interface SqlSnippet {
    label: string;
    description: string;
    body: string;
}

/** JOIN 查询模板（方言无关） */
const joinSnippet: SqlSnippet = {
    label: 'JOIN',
    description: 'join query snippet',
    body: 'SELECT ${1:t1.*}\nFROM ${2:table1} t1\nJOIN ${3:table2} t2 ON ${4:t1.id = t2.id}\nWHERE ${5:condition}\n$0',
};

/** INSERT 模板（方言无关） */
const insertSnippet: SqlSnippet = {
    label: 'INSERT',
    description: 'insert snippet',
    body: 'INSERT INTO ${1:table} (${2:columns})\nVALUES (${3:values})\n$0',
};

/** LIMIT 分页（mysql 系） */
const limitPageSnippet: SqlSnippet = {
    label: 'SELECT PAGE (LIMIT)',
    description: 'limit pagination snippet',
    body: 'SELECT *\nFROM ${1:table}\nWHERE ${2:condition}\nORDER BY ${3:id}\nLIMIT ${4:0}, ${5:25}\n$0',
};

/** OFFSET/LIMIT 分页（postgres 系） */
const offsetPageSnippet: SqlSnippet = {
    label: 'SELECT PAGE (OFFSET)',
    description: 'offset pagination snippet',
    body: 'SELECT *\nFROM ${1:table}\nWHERE ${2:condition}\nORDER BY ${3:id}\nLIMIT ${4:25} OFFSET ${5:0}\n$0',
};

/** OFFSET FETCH 分页（mssql） */
const fetchPageSnippet: SqlSnippet = {
    label: 'SELECT PAGE (FETCH)',
    description: 'offset fetch pagination snippet',
    body: 'SELECT *\nFROM ${1:table}\nWHERE ${2:condition}\nORDER BY ${3:id}\nOFFSET ${4:0} ROWS FETCH NEXT ${5:25} ROWS ONLY\n$0',
};

/** ROWNUM 分页（oracle 系） */
const rownumPageSnippet: SqlSnippet = {
    label: 'SELECT PAGE (ROWNUM)',
    description: 'rownum pagination snippet',
    body: 'SELECT * FROM (\n  SELECT t.*, ROWNUM AS rn\n  FROM ${1:table} t\n  WHERE ROWNUM <= ${2:25}\n)\nWHERE rn > ${3:0}\n$0',
};

const commonSnippets: SqlSnippet[] = [joinSnippet, insertSnippet];

const dialectSnippets: Record<string, SqlSnippet[]> = {
    mysql: [...commonSnippets, limitPageSnippet],
    mariadb: [...commonSnippets, limitPageSnippet],
    clickhouse: [...commonSnippets, limitPageSnippet],
    sqlite: [...commonSnippets, limitPageSnippet],
    postgres: [...commonSnippets, offsetPageSnippet],
    gauss: [...commonSnippets, offsetPageSnippet],
    kingbaseEs: [...commonSnippets, offsetPageSnippet],
    vastbase: [...commonSnippets, offsetPageSnippet],
    mssql: [...commonSnippets, fetchPageSnippet],
    oracle: [...commonSnippets, rownumPageSnippet],
    dm: [...commonSnippets, rownumPageSnippet],
};

/**
 * 获取指定方言的片段模板，未注册方言返回通用模板。
 * @param dbType 数据库类型
 */
export function getSnippets(dbType: string): SqlSnippet[] {
    return dialectSnippets[dbType] ?? commonSnippets;
}
