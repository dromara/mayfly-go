--PGSQL_DB_SCHEMAS 库schemas
select
	n.nspname as "schemaName"
from
	pg_namespace n
where
	has_schema_privilege(n.nspname, 'USAGE')
	and n.nspname not like 'pg_%'
    and n.nspname not like 'dbms_%'
    and n.nspname not like 'utl_%'
	and n.nspname != 'information_schema'
order by
    n.nspname
---------------------------------------
--PGSQL_TABLE_INFO 表详细信息
SELECT DISTINCT
  c.relname AS "tableName",
  COALESCE(b.description, '') AS "tableComment",
  pg_total_relation_size(c.oid) AS "dataLength",
  pg_indexes_size(c.oid) AS "indexLength",
  psut.n_live_tup AS "tableRows"
FROM
  pg_class c
  LEFT JOIN pg_description b ON c.oid = b.objoid AND b.objsubid = 0
  JOIN pg_stat_user_tables psut ON psut.relid = c.oid
WHERE
  c.relkind = 'r'
  AND c.relnamespace = (
    SELECT
      oid
    FROM
      pg_namespace
    WHERE
      nspname = current_schema()
        {{if .tableNames}}
            and c.relname in ({{.tableNames}})
        {{end}}
  )
ORDER BY
  c.relname;
---------------------------------------
--PGSQL_INDEX_INFO 表索引信息
SELECT a.indexname                                                         AS "indexName",
       'BTREE'                                                           AS "IndexType",
       case when a.indexdef like 'CREATE UNIQUE INDEX%%' then 1 else 0 end as "isUnique",
       obj_description(b.oid, 'pg_class')                                AS "indexComment",
       indexdef                                                          AS "indexDef",
       c.attname                                                         AS "columnName",
       c.attnum                                                          AS "seqInIndex",
       case when a.indexname like '%%_pkey' then 1 else 0 end             AS "isPrimaryKey"
FROM pg_indexes a
         join pg_class b on a.indexname = b.relname
         join pg_attribute c on b.oid = c.attrelid
WHERE a.schemaname = (select current_schema())
  AND a.tablename = '%s'
  AND a.indexname not like '%%_pkey'
---------------------------------------
--PGSQL_COLUMN_MA 表列信息
SELECT
  a.table_name AS "tableName",
  a.column_name AS "columnName",
  a.is_nullable AS "nullable",
  t.typname AS "dataType",
  a.character_maximum_length AS "charMaxLength",
  a.numeric_precision AS "numPrecision",
  CASE
    WHEN a.column_default LIKE 'nextval%%' THEN NULL
    ELSE a.column_default
  END AS "columnDefault",
  a.numeric_scale AS "numScale",
  CASE
    WHEN a.column_default LIKE 'nextval%%' THEN 1
    ELSE 0
  END AS "autoIncrement",
  CASE
    WHEN b.column_name IS NOT NULL THEN 1
    ELSE 0
  END AS "isPrimaryKey",
  (
    SELECT
      description
    FROM
      pg_description
    WHERE
      objoid = c.oid
      AND objsubid = a.ordinal_position
  ) AS "columnComment"
FROM
  information_schema.columns a
  LEFT JOIN information_schema.key_column_usage b ON a.table_schema = b.table_schema
  AND b.table_name = a.table_name
  AND b.column_name = a.column_name
  JOIN pg_catalog.pg_class c ON c.relname = a.table_name
  AND c.relnamespace = (
    SELECT
      oid
    FROM
      pg_catalog.pg_namespace
    WHERE
      nspname = a.table_schema
  )
  JOIN pg_catalog.pg_attribute att ON att.attrelid = c.oid
  AND att.attname = a.column_name
  JOIN pg_catalog.pg_type t ON t.oid = att.atttypid
WHERE
  a.table_schema = (
    SELECT
      current_schema()
  )
  AND a.table_name IN (%s)
ORDER BY
  a.table_name,
  a.ordinal_position;
