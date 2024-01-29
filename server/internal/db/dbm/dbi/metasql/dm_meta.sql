--DM_DB_SCHEMAS 库schemas
select
    distinct owner as SCHEMA_NAME
from all_objects
order by owner
---------------------------------------
--DM_TABLE_INFO 表详细信息
SELECT a.object_name                                      as TABLE_NAME,
       b.comments                                         as TABLE_COMMENT,
       a.created                                          as CREATE_TIME,
       TABLE_USED_SPACE(
               (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)),
               a.object_name
       ) * page()                                         as DATA_LENGTH,
       (SELECT sum(INDEX_USED_PAGES(id))* page()
        FROM SYSOBJECTS
        WHERE NAME IN (SELECT INDEX_NAME
                       FROM ALL_INDEXES
                       WHERE OWNER = 'wxb'
                         AND TABLE_NAME = a.object_name)) as INDEX_LENGTH,
       c.num_rows                                         as TABLE_ROWS

FROM all_objects a
         LEFT JOIN ALL_TAB_COMMENTS b ON b.TABLE_TYPE = 'TABLE'
    AND a.object_name = b.TABLE_NAME
    AND b.owner = a.owner
         LEFT JOIN (SELECT a.owner, a.table_name, a.num_rows FROM all_tables a) c
                   ON c.owner = a.owner AND c.table_name = a.object_name

WHERE a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  AND a.object_type = 'TABLE'
  AND a.status = 'VALID'
ORDER BY a.object_name
---------------------------------------
--DM_INDEX_INFO 表索引信息
select
    a.index_name as INDEX_NAME,
    a.index_type as INDEX_TYPE,
    case when a.uniqueness = 'UNIQUE' then 1 else 0 end as IS_UNIQUE,
    indexdef(b.object_id,1) as INDEX_DEF,
    c.column_name as COLUMN_NAME,
    c.column_position as SEQ_IN_INDEX,
    '无' as INDEX_COMMENT
FROM ALL_INDEXES a
         LEFT JOIN all_objects b on a.owner = b.owner and b.object_name = a.index_name and b.object_type = 'INDEX'
         LEFT JOIN ALL_IND_COLUMNS c
                   on a.owner = c.table_owner and a.index_name = c.index_name and a.TABLE_NAME = c.table_name

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
       case when t.COL_NAME = a.column_name then 1 else 0 end as IS_IDENTITY,
       case when t2.constraint_type = 'P' then 1 else 0 end   as IS_PRIMARY_KEY
from all_tab_columns a
         left join user_col_comments b
                   on b.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
                       and b.table_name = a.table_name
                       and a.column_name = b.column_name
         left join (select b.owner, b.TABLE_NAME, a.NAME as COL_NAME
                    from SYS.SYSCOLUMNS a,
                         SYS.all_tables b,
                         SYS.SYSOBJECTS c
                    where a.INFO2 & 0x01 = 0x01
                      and a.ID = c.ID
                      and c.NAME = b.TABLE_NAME) t
                   on t.table_name = a.table_name and t.owner = a.owner
         left join (select uc.OWNER, uic.column_name, uic.table_name, uc.constraint_type
                    from user_ind_columns uic
                             left join user_constraints uc on uic.index_name = uc.index_name) t2
                   on t2.table_name = t.table_name and a.column_name = t2.column_name
where a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
  and a.table_name in (%s)
order by a.table_name,
         a.column_id