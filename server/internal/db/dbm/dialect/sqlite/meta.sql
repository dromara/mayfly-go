--SQLITE_TABLE_INFO 表详细信息
select tbl_name as tableName,
       ''       as tableComment,
       ''       as createTime,
       0        as dataLength,
       0        as indexLength,
       0        as tableRows
FROM sqlite_master
WHERE type = 'table'
  and name not like 'sqlite_%'
    {{if .tableNames}}
        and tbl_name in ({{.tableNames}})
    {{end}}
ORDER BY tbl_name
---------------------------------------
--SQLITE_INDEX_INFO 表索引信息
-- 走 pragma 而非 sqlite_master：约束（PRIMARY KEY/UNIQUE）自动生成的隐式索引在sqlite_master中sql为NULL，
-- 既取不到列名也无法DROP/按原名重建，旧口径会把它们报成“无列的非唯一索引”并在dump中生成非法DDL
-- origin='pk'（主键索引）不返回：它随CREATE TABLE的PRIMARY KEY重建，与mysql排除PRIMARY索引、pg排除_pkey后缀索引的口径一致
select il.name     as indexName,
       il.origin   as indexOrigin,
       il."unique" as isUnique,
       (select group_concat(ii.name, ',') from pragma_index_info(il.name) ii order by ii.seqno) as columnName,
       'normal'    as indexType,
       ''          as indexComment
from pragma_index_list('%s') il
where il.origin != 'pk'
order by il.seq