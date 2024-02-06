--ORACLE_DB_SCHEMAS 库schemas
select USERNAME
from sys.all_users
order by USERNAME
---------------------------------------
--ORACLE_TABLE_INFO 表详细信息
select a.TABLE_NAME,
       b.COMMENTS as TABLE_COMMENT,
       c.CREATED  as CREATE_TIME,
       d.BYTES    as DATA_LENGTH,
       0          as INDEX_LENGTH,
       a.NUM_ROWS as TABLE_ROWS
from ALL_TABLES a
         left join ALL_TAB_COMMENTS b on b.TABLE_NAME = a.TABLE_NAME AND b.OWNER = a.OWNER
         left join ALL_OBJECTS c on c.OBJECT_TYPE = 'TABLE' AND c.OWNER = a.OWNER AND c.OBJECT_NAME = a.TABLE_NAME
         left join dba_segments d on d.SEGMENT_TYPE = 'TABLE' AND d.OWNER = a.OWNER AND d.SEGMENT_NAME = a.TABLE_NAME
where a.owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual)
ORDER BY a.TABLE_NAME
---------------------------------------
--ORACLE_INDEX_INFO 表索引信息
SELECT ai.INDEX_NAME                          AS INDEX_NAME,
       ai.INDEX_TYPE                          AS INDEX_TYPE,
       CASE
           WHEN ai.uniqueness = 'UNIQUE' THEN 1
           ELSE 0
           END AS IS_UNIQUE,
       (SELECT LISTAGG(column_name, ', ') WITHIN GROUP (ORDER BY column_position)
        FROM ALL_IND_COLUMNS aic
        WHERE aic.INDEX_NAME = ai.INDEX_NAME
          AND aic.TABLE_NAME = ai.TABLE_NAME) AS COLUMN_NAME,
            1                                 AS SEQ_IN_INDEX,
       (SELECT comments
        FROM ALL_IND_COLUMNS aic
            LEFT JOIN ALL_COL_COMMENTS acc ON aic.column_name = acc.column_name
        WHERE aic.INDEX_OWNER = ai.OWNER
          AND aic.INDEX_NAME = ai.INDEX_NAME
          AND aic.TABLE_NAME = ai.TABLE_NAME
          AND ROWNUM = 1)                     AS INDEX_COMMENT
FROM ALL_INDEXES ai
WHERE ai.OWNER = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM DUAL)
  AND ai.table_name = '%s'
---------------------------------------
--ORACLE_COLUMN_MA 表列信息
SELECT a.TABLE_NAME                                              as TABLE_NAME,
       a.COLUMN_NAME                                             as COLUMN_NAME,
       case
           when a.NULLABLE = 'Y' then 'YES'
           when a.NULLABLE = 'N' then 'NO'
           else 'NO' end                                         as NULLABLE,
       case
           when a.DATA_PRECISION > 0 then a.DATA_TYPE
           else (a.DATA_TYPE || '(' || a.DATA_LENGTH || ')') end as COLUMN_TYPE,
       b.COMMENTS                                                as COLUMN_COMMENT,
       a.DATA_DEFAULT                                            as COLUMN_DEFAULT,
       a.DATA_SCALE                                              as NUM_SCALE,
       CASE WHEN d.pri IS NOT NULL THEN 1 ELSE 0 END         as IS_PRIMARY_KEY,
       CASE WHEN a.IDENTITY_COLUMN = 'YES' THEN 1 ELSE 0 END as IS_IDENTITY
FROM ALL_TAB_COLUMNS a
         LEFT JOIN ALL_COL_COMMENTS b
                   on a.OWNER = b.OWNER AND a.TABLE_NAME = b.TABLE_NAME AND a.COLUMN_NAME = b.COLUMN_NAME
         LEFT JOIN (select ac.TABLE_NAME, ac.OWNER, cc.COLUMN_NAME, 1 as pri
                    from ALL_CONSTRAINTS ac
                             join ALL_CONS_COLUMNS cc on cc.CONSTRAINT_NAME = ac.CONSTRAINT_NAME AND cc.OWNER = ac.OWNER
                    where cc.CONSTRAINT_NAME IS NOT NULL
                      AND ac.CONSTRAINT_TYPE = 'P') d
                   on d.OWNER = a.OWNER AND d.TABLE_NAME = a.TABLE_NAME AND d.COLUMN_NAME = a.COLUMN_NAME
WHERE a.OWNER = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM DUAL)
  AND a.TABLE_NAME in (%s)
order by a.COLUMN_ID
