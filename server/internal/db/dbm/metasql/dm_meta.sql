--DM_DB_SCHEMAS 库schemas
select
    distinct owner as SCHEMA_NAME
from dba_objects
---------------------------------------
--DM_TABLE_INFO 表详细信息
select
    a.object_name as TABLE_NAME,
    b.comments as TABLE_COMMENT,
    a.created as CREATE_TIME,
    TABLE_USED_SPACE((SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)), a.object_name)*page() as DATA_LENGTH
from dba_objects a
         LEFT JOIN DBA_TAB_COMMENTS b ON b.TABLE_TYPE='TABLE' and a.object_name = b.TABLE_NAME and b.owner = a.owner
where a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  and a.object_type = 'TABLE' and a.status = 'VALID'
---------------------------------------
--DM_INDEX_INFO 表索引信息
select
    a.index_name as INDEX_NAME,
    a.index_type as INDEX_TYPE,
    case when a.uniqueness = 'UNIQUE' then 1 else 0 end as NON_UNIQUE,
    indexdef(b.object_id,1) as INDEX_DEF,
    c.column_name as COLUMN_NAME,
    c.column_position as SEQ_IN_INDEX,
    '无' as INDEX_COMMENT
FROM DBA_INDEXES  a
         LEFT JOIN dba_objects b on a.owner = b.owner and b.object_name = a.index_name and b.object_type = 'INDEX'
         LEFT JOIN DBA_IND_COLUMNS c on a.owner = c.table_owner and a.index_name = c.index_name and a.TABLE_NAME = c.table_name

WHERE a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  and  a.TABLE_NAME = '%s'
  and indexdef(b.object_id,1) != '禁止查看系统定义的索引信息'
order by  a.TABLE_NAME, a.index_name, c.column_position asc
---------------------------------------
--DM_COLUMN_MA 表列信息
select a.table_name                                                                        as TABLE_NAME,
       a.column_name                                                                       as COLUMN_NAME,
       case when a.NULLABLE = 'Y' then 'YES' when a.NULLABLE = 'N' then 'NO' else 'NO' end as NULLABLE,
       case
           when a.char_col_decl_length > 0 then concat(a.data_type, '(', a.char_col_decl_length, ')')
           when a.data_precision > 0 and a.data_scale > 0
               then concat(a.data_type, '(', a.data_precision, ',', a.data_scale, ')')
           else a.data_type end
                                                                                           as COLUMN_TYPE,
       b.comments                                                                          as COLUMN_COMMENT,
       a.data_default                                                                      as COLUMN_DEFAULT,
       a.data_scale                                                                        as NUM_SCALE,
       case when t.COL_NAME = a.column_name then 'PRI' else '' end                         as COLUMN_KEY
from dba_tab_columns a
         left join user_col_comments b
                   on b.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)) and b.table_name = a.table_name and
                      a.column_name = b.column_name
         left join (select b.owner, b.table_name, a.name COL_NAME
                    from SYS.SYSCOLUMNS a,
                         dba_tables b,
                         sys.sysobjects c,
                         sys.sysobjects d
                    where a.INFO2 & 0x01 = 0x01
   and a.id=c.id and d.type$ = 'SCH' and d.id = c.schid
   and b.owner =  (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
   and c.schid = ( select id from sys.sysobjects where type$ = 'SCH' and name = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)))
   and c.name = b.table_name) t
                   on t.table_name = a.table_name
where a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  and a.table_name in (%s)
order by a.table_name, a.column_id