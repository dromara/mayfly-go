--DM_DB_SCHEMAS 库schemas
select
    distinct owner as schemaName
from dba_objects
---------------------------------------
--DM_TABLE_INFO 表详细信息
select
    a.object_name as tableName,
    b.comments as tableComment,
    a.created as createTime,
    TABLE_USED_SPACE((SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)), a.object_name)*page() as dataLength
from dba_objects a
         JOIN USER_TAB_COMMENTS b ON b.TABLE_TYPE='TABLE' and a.object_name = b.TABLE_NAME
where a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  and a.object_type = 'TABLE'
---------------------------------------
--DM_INDEX_INFO 表索引信息
select
    a.index_name as indexName,
    a.index_type as indexType,
    case when a.uniqueness = 'UNIQUE' then 1 else 0 end as nonUnique,
    indexdef(b.object_id,1) as indexDef,
    c.column_name as columnName,
    c.column_position as seqInIndex,
    '无' as indexComment
FROM DBA_INDEXES  a
         JOIN dba_objects b on a.owner = b.owner and b.object_name = a.index_name and b.object_type = 'INDEX'
         JOIN DBA_IND_COLUMNS c on a.owner = c.table_owner and a.index_name = c.index_name and a.TABLE_NAME = c.table_name

WHERE a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  and  a.TABLE_NAME = '%s'
  and indexdef(b.object_id,1) != '禁止查看系统定义的索引信息'
order by  a.TABLE_NAME, a.index_name, c.column_position asc
---------------------------------------
--DM_COLUMN_MA 表列信息
select a.table_name                                                                        as tableName,
       a.column_name                                                                       as columnName,
       case when a.NULLABLE = 'Y' then 'YES' when a.NULLABLE = 'N' then 'NO' else 'NO' end as nullable,
       case
           when a.char_col_decl_length > 0 then concat(a.data_type, '(', a.char_col_decl_length, ')')
           when a.data_precision > 0 and a.data_scale > 0
               then concat(a.data_type, '(', a.data_precision, ',', a.data_scale, ')')
           else a.data_type end
                                                                                           as columnType,
       b.comments                                                                          as columnComment,
       a.data_default                                                                      as columnDefault,
       a.data_scale                                                                        as numScale,
       case when t.COL_NAME = a.column_name then 'PRI' else '' end                         as columnKey
from dba_tab_columns a
         join user_col_comments b on b.owner = a.owner and b.table_name = a.table_name and a.column_name = b.column_name
         join (select b.owner, b.table_name, a.name COL_NAME
               from SYS.SYSCOLUMNS a,
                    dba_tables b,
                    sys.sysobjects c
               where a.INFO2 & 0x01 = 0x01
   and a.id=c.id and c.name = b.table_name) t on t.owner = a.owner and t.table_name = a.table_name
where a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  and a.table_name in (%s)
order by a.table_name, a.column_id