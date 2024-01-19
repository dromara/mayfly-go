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
ORDER BY tbl_name
---------------------------------------
--SQLITE_INDEX_INFO 表索引信息
select name     as indexName,
       `sql`    as indexSql,
       'normal' as indexType,
       ''       as indexComment
FROM sqlite_master
WHERE type = 'index'
  and tbl_name = '%s'
ORDER BY name