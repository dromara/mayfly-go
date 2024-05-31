--MYSQL_DBS 数据库名信息
SELECT
	SCHEMA_NAME AS dbname
FROM
	information_schema.SCHEMATA
WHERE
	SCHEMA_NAME NOT IN ('mysql', 'information_schema', 'performance_schema')
ORDER BY SCHEMA_NAME
---------------------------------------
--MYSQL_TABLE_INFO 表详细信息
SELECT
  table_name tableName,
  table_comment tableComment,
  table_rows tableRows,
  data_length dataLength,
  index_length indexLength,
  create_time createTime
FROM
  information_schema.tables
WHERE
  table_type = 'BASE TABLE'
    {{if .tableNames}}
        AND table_name IN ({{.tableNames}})
    {{end}}
  AND table_schema = (
    SELECT
      database ()
  )
ORDER BY table_name
---------------------------------------
--MYSQL_INDEX_INFO 索引信息
SELECT
  index_name indexName,
  column_name columnName,
  index_type indexType,
  IF(non_unique, 0, 1) isUnique,
  SEQ_IN_INDEX seqInIndex,
  INDEX_COMMENT indexComment,
  index_name = 'PRIMARY' as isPrimaryKey
FROM
  information_schema.STATISTICS
WHERE
  table_schema = (
    SELECT
      database ()
  )
  AND index_name != 'PRIMARY'
  AND table_name = ?
ORDER BY
  index_name asc,
  SEQ_IN_INDEX asc
---------------------------------------
--MYSQL_COLUMN_MA 列信息元数据
SELECT table_name     tableName,
       column_name    columnName,
       column_type    columnType,
       data_type      dataType,
       column_default columnDefault,
       column_comment columnComment,
       CASE
           WHEN column_key = 'PRI' THEN
               1
           ELSE 0
           END AS     isPrimaryKey,
       CASE
           WHEN extra LIKE '%%auto_increment%%' THEN
               1
           ELSE 0
           END AS     isIdentity,
       is_nullable    nullable,
       CHARACTER_MAXIMUM_LENGTH charMaxLength,
       NUMERIC_SCALE  numScale,
       NUMERIC_PRECISION numPrecision
FROM information_schema.COLUMNS
WHERE table_schema = (SELECT DATABASE())
  AND table_name IN (%s)
ORDER BY table_name,
         ordinal_position