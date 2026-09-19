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
  SUB_PART subPart,
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
           END AS     autoIncrement,
       is_nullable    nullable,
       CHARACTER_MAXIMUM_LENGTH charMaxLength,
       NUMERIC_SCALE  numScale,
       -- 日期/时间类的精度存于DATETIME_PRECISION（NUMERIC_PRECISION为NULL），必须一并取出：
       -- 否则datetime(3)转异构库时丢失小数秒，目标库按自身默认精度建表会静默截断时间值
       COALESCE(NUMERIC_PRECISION, DATETIME_PRECISION) numPrecision,
       -- MySQL 8.0.13+的表达式默认值与生成列的派生表达式均存于COLUMN_DEFAULT（且会剥去/改写原始书写形态），
       -- 与字符串字面量默认值无法从文本区分，必须靠EXTRA判定：
       -- DEFAULT_GENERATED=表达式默认值；VIRTUAL/STORED GENERATED=生成列（不可显式插入）
       CASE
           WHEN EXTRA LIKE '%%DEFAULT_GENERATED%%' THEN 1
           ELSE 0
           END AS     isExprDefault,
       -- MySQL的「自动更新」子句（ON UPDATE CURRENT_TIMESTAMP[(fsp)]）仅存在于EXTRA，其他列无法取到；
       -- 不取则结构迁移会静默丢失该行为（业务依赖update_time自动刷新的列迁移后不再更新）
       CASE
           WHEN EXTRA LIKE '%%on update%%' THEN SUBSTRING(EXTRA, LOCATE('on update', EXTRA))
           ELSE ''
           END AS     onUpdate,
       CASE
           WHEN EXTRA LIKE '%%VIRTUAL GENERATED%%' OR EXTRA LIKE '%%STORED GENERATED%%' THEN 1
           ELSE 0
           END AS     isGenerated
FROM information_schema.COLUMNS
WHERE table_schema = (SELECT DATABASE())
  AND table_name IN (%s)
ORDER BY table_name,
         ordinal_position