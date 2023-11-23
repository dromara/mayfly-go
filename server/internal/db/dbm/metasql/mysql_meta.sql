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
  AND table_schema = (
    SELECT
      database ()
  )
---------------------------------------
--MYSQL_INDEX_INFO 索引信息
SELECT
  index_name indexName,
  column_name columnName,
  index_type indexType,
  non_unique nonUnique,
  SEQ_IN_INDEX seqInIndex,
  INDEX_COMMENT indexComment
FROM
  information_schema.STATISTICS
WHERE
  table_schema = (
    SELECT
      database ()
  )
  AND table_name = '%s'
ORDER BY
  index_name asc,
  SEQ_IN_INDEX asc
---------------------------------------
--MYSQL_COLUMN_MA 列信息元数据
SELECT
  table_name tableName,
  column_name columnName,
  column_type columnType,
  column_default columnDefault,
  column_comment columnComment,
  column_key columnKey,
  extra extra,
  is_nullable nullable,
  NUMERIC_SCALE numScale
from
  information_schema.columns
WHERE
  table_schema = (
    SELECT
      database ()
  )
  AND table_name in (%s)
ORDER BY
  tableName,
  ordinal_position