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
--PGSQL_TABLE_SEARCH 表名服务端搜索（ILIKE 下推，$1 为绑定模式；可选 LIMIT/OFFSET 由代码追加；schema 取 current_schema()，与 PGSQL_TABLE_INFO 同源）
-- ILIKE 而非 LIKE：pg 的 LIKE 区分大小写，会与 mysql 下推/无下推回退路径的「大小写不敏感」口径分叉（同输入两种结果）
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
  AND c.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = current_schema())
  AND c.relname ILIKE $1
ORDER BY
  c.relname
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
  -- 时间类列的小数秒精度存于datetime_precision（numeric_precision为NULL），必须一并取出：
  -- 否则timestamp(3)转异构库时丢失小数秒，目标库按自身默认精度建表会静默截断时间值
  COALESCE(a.numeric_precision, a.datetime_precision) AS "numPrecision",
  CASE
    WHEN a.column_default LIKE 'nextval%%' THEN NULL
    ELSE a.column_default
  END AS "columnDefault",
  a.numeric_scale AS "numScale",
  -- 注：pg 10+的GENERATED ALWAYS AS IDENTITY列的column_default为空，仅凭nextval无法识别；
  -- 但is_identity/attidentity在国产pg兼容库（GaussDB/Vastbase/Kingbase等基于pg 9.x）中不存在，
  -- 在此引用会使整个列元数据查询报错，影响远大于收益，故不在此区分identity列
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
---------------------------------------
--PGSQL_VIEWS 视图信息
SELECT
  n.nspname AS "schemaName",
  c.relname AS "viewName",
  pg_get_viewdef(c.oid) AS "viewDefinition",
  COALESCE(obj_description(c.oid), '') AS "viewComment"
FROM pg_class c
  JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'v'
  AND n.nspname = COALESCE(NULLIF('%s', ''), current_schema())
ORDER BY c.relname
---------------------------------------
--PGSQL_SEQUENCES 序列信息（含定义属性 + 当前值，供前端「属性面板」直接展示，免二次查询）
SELECT
  n.nspname AS "schemaName",
  c.relname AS "seqName",
  COALESCE(obj_description(c.oid), '') AS "seqComment",
  format_type(s.seqtypid, NULL) AS "dataType",
  s.seqstart AS "startValue",
  s.seqincrement AS "incrementBy",
  s.seqmin AS "minValue",
  s.seqmax AS "maxValue",
  s.seqcache AS "cacheSize",
  s.seqcycle AS "isCycle",
  COALESCE(ps.last_value::text, '') AS "lastValue"
FROM pg_class c
  JOIN pg_namespace n ON n.oid = c.relnamespace
  JOIN pg_sequence s ON s.seqrelid = c.oid
  LEFT JOIN pg_sequences ps ON ps.schemaname = n.nspname AND ps.sequencename = c.relname
WHERE c.relkind = 'S'
  AND n.nspname = COALESCE(NULLIF('%s', ''), current_schema())
ORDER BY c.relname
---------------------------------------
--PGSQL_VIEW_DDL 视图定义
SELECT pg_get_viewdef(c.oid) AS "viewDefinition"
FROM pg_class c
  JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'v'
  AND n.nspname = COALESCE(NULLIF('%s', ''), current_schema())
  AND c.relname = '%s'
---------------------------------------
--PGSQL_SEQUENCE_DDL 序列定义（pg 无内建序列DDL函数，由 pg_sequence 目录列重建 CREATE SEQUENCE）
SELECT
  'CREATE SEQUENCE ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname)
  || ' AS ' || format_type(s.seqtypid, NULL)
  || ' INCREMENT BY ' || s.seqincrement
  || ' MINVALUE ' || s.seqmin
  || ' MAXVALUE ' || s.seqmax
  || ' START WITH ' || s.seqstart
  || ' CACHE ' || s.seqcache
  || CASE WHEN s.seqcycle THEN ' CYCLE' ELSE ' NO CYCLE' END AS "sequenceDdl"
FROM pg_class c
  JOIN pg_namespace n ON n.oid = c.relnamespace
  JOIN pg_sequence s ON s.seqrelid = c.oid
WHERE c.relkind = 'S'
  AND n.nspname = COALESCE(NULLIF('%s', ''), current_schema())
  AND c.relname = '%s'

---------------------------------------
--PGSQL_TABLE_KEYS 主键与唯一键约束（成员列按键内序号有序）
-- 注意：information_schema.key_column_usage.ordinal_position 是「列在表中的位置」而非「键内序」（PG 知名坑），
-- 故用 pg_constraint.conkey 的 unnest WITH ORDINALITY 取真实键内顺序；contype p=主键 u=唯一键。
SELECT
  con.conname AS "keyName",
  CASE con.contype WHEN 'p' THEN 'PRIMARY KEY' WHEN 'u' THEN 'UNIQUE' END AS "keyType",
  a.attname AS "columnName",
  ck.ord AS "ordinal"
FROM pg_constraint con
JOIN LATERAL unnest(con.conkey) WITH ORDINALITY AS ck(attnum, ord) ON true
JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = ck.attnum
JOIN pg_class c ON c.oid = con.conrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = COALESCE(NULLIF('%s', ''), current_schema())
  AND c.relname = '%s'
  AND con.contype IN ('p', 'u')
ORDER BY con.conname, ck.ord
