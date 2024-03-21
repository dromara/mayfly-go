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
select
	c.relname as "tableName",
	obj_description (c.oid) as "tableComment",
	pg_table_size ('"' || n.nspname || '"."' || c.relname || '"') as "dataLength",
	pg_indexes_size ('"' || n.nspname || '"."' || c.relname || '"') as "indexLength",
	psut.n_live_tup as "tableRows"
from
	pg_class c
join pg_namespace n on
	c.relnamespace = n.oid
join pg_stat_user_tables psut on
	psut.relid = c.oid
where
    has_table_privilege(CAST(c.oid AS regclass), 'SELECT')
	and n.nspname = current_schema()
	and c.reltype > 0
    {{if .tableNames}}
        and c.relname in ({{.tableNames}})
    {{end}}
order by c.relname
---------------------------------------
--PGSQL_INDEX_INFO 表索引信息
SELECT
    indexname AS "indexName",
    'BTREE' AS "IndexType",
    case when indexdef like 'CREATE UNIQUE INDEX%%' then 1 else 0 end as "isUnique",
    obj_description(b.oid, 'pg_class') AS "indexComment",
    indexdef AS "indexDef",
    c.attname AS "columnName",
    c.attnum AS "seqInIndex"
FROM pg_indexes a
     join pg_class b on a.indexname = b.relname
     join pg_attribute c on b.oid = c.attrelid
WHERE a.schemaname = (select current_schema())
  AND a.tablename = '%s';
---------------------------------------
--PGSQL_COLUMN_MA 表列信息
SELECT a.*,
       a.table_name                                                                            AS "tableName",
       a.column_name                                                                           AS "columnName",
       a.is_nullable                                                                           AS "nullable",
       a.udt_name                                                                              AS "dataType",
       a.character_maximum_length                                                              AS "charMaxLength",
       a.numeric_precision                                                                     AS "numPrecision",
       a.column_default                                                                        AS "columnDefault",
       a.numeric_scale                                                                         AS "numScale",
       case when a.column_default like 'nextval%%' then 1 else 0 end                           AS "isIdentity",
       case when b.column_name is not null then 1 else 0 end                                   AS "isPrimaryKey",
       col_description((a.table_schema || '.' || a.table_name) ::regclass, a.ordinal_position) AS "columnComment"
FROM information_schema.columns a
         left join information_schema.key_column_usage b
                   on a.table_schema = b.table_schema and b.table_name = a.table_name and b.column_name = a.column_name
WHERE a.table_schema = (select current_schema())
  and a.table_name in (%s)
order by a.table_name, a.ordinal_position
