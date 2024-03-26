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
{{if .tableNames}}
    and t.name in ({{.tableNames}})
{{end}}
ORDER BY t.name ASC;
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
       idx.value                         AS indexComment,
       CASE
           WHEN LEFT(ind.name, 3) = 'PK_' THEN 1
           ELSE 0
           END                           AS isPrimaryKey
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
SELECT t.name                                               AS TABLE_NAME,
       c.name                                               AS COLUMN_NAME,
       CASE WHEN c.is_nullable = 1 THEN 'YES' ELSE 'NO' END AS NULLABLE,
       tp.name                                              AS DATA_TYPE,
       c.max_length                                         AS CHAR_MAX_LENGTH,
       c.precision                                          AS NUM_PRECISION,
       c.scale                                              AS NUM_SCALE,
       ep.value                                             AS COLUMN_COMMENT,
       COLUMN_DEFAULT =
       CASE WHEN c.default_object_id IS NOT NULL THEN object_definition(c.default_object_id) ELSE '' END,
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
