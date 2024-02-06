--MSSQL_DBS 数据库名信息
SELECT name AS dbname
FROM sys.databases
---------------------------------------
--MSSQL_TABLE_DETAIL 查询表名和表注释
SELECT t.name   AS tableName,
       ep.value AS tableComment
FROM sys.tables t
         left OUTER JOIN sys.schemas ss on t.schema_id = ss.schema_id
         LEFT OUTER JOIN
     sys.extended_properties ep ON ep.major_id = t.object_id AND ep.minor_id = 0 AND ep.class = 1
WHERE ss.name = ?
  and t.name = ?
---------------------------------------
--MSSQL_DB_SCHEMAS 数据库下所有schema
SELECT a.SCHEMA_NAME
FROM information_schema.schemata a
where a.catalog_name = DB_NAME()
  and (a.SCHEMA_NAME in ('dbo', 'guest') or a.SCHEMA_NAME not like 'db_%')
  and a.SCHEMA_NAME not in ('sys', 'INFORMATION_SCHEMA')
---------------------------------------
--MSSQL_TABLE_INFO 表详细信息
SELECT t.name        AS tableName,
       ss.name AS tableSchema,
       c.value       AS tableComment,
       p.rows        AS tableRows,
       0             AS dataLength,
       0             AS indexLength,
       t.create_date AS createTime
FROM sys.tables t
         left OUTER JOIN sys.schemas ss on t.schema_id = ss.schema_id
         left OUTER JOIN sys.partitions p ON t.object_id = p.object_id AND p.index_id = 1
         left OUTER JOIN sys.extended_properties c ON t.object_id = c.major_id AND c.minor_id = 0 AND c.class = 1
where ss.name = ?
ORDER BY t.name DESC;
---------------------------------------
--MSSQL_INDEX_INFO 索引信息
SELECT ind.name                          AS indexName,
       col.name                          AS columnName,
       CASE
           WHEN ind.is_primary_key = 1 THEN 'CLUSTERED'
           ELSE 'NON-CLUSTERED'
           END                           AS indexType,
       IIF(ind.is_unique = 'true', 1, 0) AS isUnique,
       ic.key_ordinal                    AS seqInIndex,
       idx.value                         AS indexComment
FROM sys.indexes ind
         LEFT JOIN sys.tables t on t.object_id = ind.object_id
         LEFT JOIN sys.schemas ss on t.schema_id = ss.schema_id
         LEFT JOIN
     sys.index_columns ic ON ind.object_id = ic.object_id AND ind.index_id = ic.index_id
         LEFT JOIN
     sys.columns col ON ind.object_id = col.object_id AND ic.column_id = col.column_id
         LEFT JOIN
     sys.extended_properties idx ON ind.object_id = idx.major_id AND ind.index_id = idx.minor_id AND idx.class = 7
WHERE ss.name = ?
  and ind.name is not null
  and t.name = ?
---------------------------------------
--MSSQL_COLUMN_MA 列信息元数据
SELECT t.name   AS TABLE_NAME,
       c.name   AS COLUMN_NAME,
       CASE
           WHEN c.is_nullable = 1 THEN 'YES'
           ELSE 'NO'
           END  AS NULLABLE,
       tp.name +
       CASE
           WHEN tp.name IN ('char', 'varchar', 'nchar', 'nvarchar') THEN '(' + CASE
                                                                                   WHEN c.max_length = -1 THEN 'max'
                                                                                   ELSE CAST(c.max_length AS NVARCHAR(255)) END +
                                                                         ')'
           WHEN tp.name IN ('numeric', 'decimal') THEN '(' + CAST(c.precision AS NVARCHAR(255)) + ',' +
                                                       CAST(c.scale AS NVARCHAR(255)) + ')'
           ELSE ''
           END  AS COLUMN_TYPE,
       ep.value AS COLUMN_COMMENT,
       COLUMN_DEFAULT = CASE
                            WHEN c.default_object_id IS NOT NULL THEN object_definition(c.default_object_id)
                            ELSE ''
           END,
       c.scale  AS NUM_SCALE,
       IS_IDENTITY = COLUMNPROPERTY(c.object_id, c.name, 'IsIdentity'),
       IS_PRIMARY_KEY = CASE
                            WHEN (SELECT COUNT(*)
                                  FROM sys.index_columns ic
                                           INNER JOIN sys.indexes i
                                                      ON ic.index_id = i.index_id AND ic.object_id = i.object_id
                                  WHERE ic.object_id = c.object_id
                                    AND ic.column_id = c.column_id
                                    AND i.is_primary_key = 1) > 0 THEN 1
                            ELSE 0
           END
FROM sys.tables t
         INNER JOIN sys.schemas ss on t.schema_id = ss.schema_id
         INNER JOIN
     sys.columns c ON t.object_id = c.object_id
         INNER JOIN
     sys.types tp ON c.system_type_id = tp.system_type_id AND c.user_type_id = tp.user_type_id
         LEFT JOIN
     sys.extended_properties ep ON t.object_id = ep.major_id AND c.column_id = ep.minor_id AND ep.class = 1
WHERE ss.name = ?
  and t.name in (%s)
ORDER BY t.name, c.column_id
---------------------------------------
--MSSQL_TABLE_DDL 建表ddl
declare
@tabname varchar(50)
set @tabname= ? --表名
if ( object_id('tempdb.dbo.#t') is not null)
begin
DROP TABLE #t
end
select 'create table [' + so.name + '] (' + o.list + ')'
           + CASE
                 WHEN tc.Constraint_Name IS NULL THEN ''
                 ELSE 'ALTER TABLE ' + so.Name + ' ADD CONSTRAINT ' + tc.Constraint_Name + ' PRIMARY KEY ' +
                      ' (' + LEFT(j.List, Len(j.List)-1) + ')' END
    TABLE_DDL
into #t
from sysobjects so
    cross apply
    (SELECT
    ' \n ['+ column_name +'] ' +
    data_type + case data_type
    when 'sql_variant' then ''
    when 'text' then ''
    when 'ntext' then ''
    when 'xml' then ''
    when 'decimal' then '(' + cast (numeric_precision as varchar) + ', ' + cast (numeric_scale as varchar) + ')'
    else coalesce ('('+ case when character_maximum_length = -1 then 'MAX' else cast (character_maximum_length as varchar) end +')', '') end + ' ' +
    case when exists (
    select id from syscolumns
    where object_name(id)=so.name
    and name = column_name
    and columnproperty(id, name, 'IsIdentity') = 1
    ) then
    'IDENTITY(' +
    cast (ident_seed(so.name) as varchar) + ',' +
    cast (ident_incr(so.name) as varchar) + ')'
    else ''
    end + ' ' +
    (case when IS_NULLABLE = 'No' then 'NOT ' else '' end ) + 'NULL ' +
    case when information_schema.columns.COLUMN_DEFAULT IS NOT NULL THEN 'DEFAULT '+ information_schema.columns.COLUMN_DEFAULT ELSE '' END + ', '
    from information_schema.columns where table_name = so.name
    order by ordinal_position
    FOR XML PATH ('')) o (list)
    left join
    information_schema.table_constraints tc
on tc.Table_name = so.Name
    AND tc.Constraint_Type = 'PRIMARY KEY'
    cross apply
    (select '[' + Column_Name + '], '
    FROM information_schema.key_column_usage kcu
    WHERE kcu.Constraint_Name = tc.Constraint_Name
    ORDER BY
    ORDINAL_POSITION
    FOR XML PATH ('')) j (list)
where xtype = 'U'
  AND name =@tabname

select (
           case
               when (select count(a.constraint_type)
                     from information_schema.table_constraints a
                              inner join information_schema.constraint_column_usage b
                                         on a.constraint_name = b.constraint_name
                     where a.constraint_type = 'PRIMARY KEY'--主键
                       and a.table_name = @tabname) = 1 then replace(table_ddl
                   , ', )ALTER TABLE'
                   , ')' + CHAR (13)+'ALTER TABLE')
               else SUBSTRING(table_ddl
                        , 1
                        , len(table_ddl) - 3) + ')' end
           ) as TableDDL
from #t

drop table #t
---------------------------------------
--MSSQL_TABLE_INDEX_DDL 建索引ddl
DECLARE
@TableName NVARCHAR(255)
SET @TableName = ?;

SELECT 'CREATE ' +
       CASE
           WHEN i.is_primary_key = 1 THEN 'CLUSTERED '
           WHEN i.type_desc = 'HEAP' THEN ''
           ELSE 'NONCLUSTERED '
           END +
       'INDEX ' + i.name + ' ON ' + t.name + ' (' +
       STUFF((SELECT ',' + c.name +
                     CASE
                         WHEN ic.is_descending_key = 1 THEN ' DESC'
                         ELSE ' ASC'
                         END
              FROM sys.index_columns ic
                       INNER JOIN
                   sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
              WHERE ic.object_id = i.object_id
                AND ic.index_id = i.index_id
              ORDER BY ic.key_ordinal
                 FOR XML PATH (''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 1, '') + ');' AS IndexDDL
FROM sys.tables t
         INNER JOIN
     sys.indexes i ON t.object_id = i.object_id
WHERE t.name = @TableName;
