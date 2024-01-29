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
SELECT a.table_name                                                                           AS "tableName",
       a.column_name                                                                          AS "columnName",
       a.is_nullable                                                                          AS "nullable",
    case when character_maximum_length > 0 then concat(udt_name, '(',character_maximum_length,')') else udt_name end  AS "columnType",
       a.column_default                                                                       as "columnDefault",
       a.numeric_scale                                                                        AS "numScale",
       case when a.column_default like 'nextval%%' then 1 else 0 end                             "isIdentity",
       case when b.column_name is not null then 1 else 0 end                                     "isPrimaryKey",
       col_description((a.table_schema || '.' || a.table_name)::regclass, a.ordinal_position) AS "columnComment"
FROM information_schema.columns a
         left join information_schema.key_column_usage b
                   on a.table_schema = b.table_schema and b.table_name = a.table_name and b.column_name = a.column_name
WHERE a.table_schema = (select current_schema())
  and a.table_name in (%s)
order by a.table_name, a.ordinal_position
---------------------------------------
--PGSQL_TABLE_DDL_FUNC 表ddl函数
 CREATE OR REPLACE FUNCTION showcreatetable(namespace character varying, tablename character varying)
        RETURNS character varying AS
        $BODY$
        declare
        tableScript character varying default '';
        begin
        -- columns
        tableScript:=tableScript || ' CREATE TABLE '|| tablename|| ' ( '|| chr(13)||chr(10) || array_to_string(
        array(
        select ' ' || concat_ws(' ',fieldName, fieldType, isNullStr ) as column_line
        from (
        select a.attname as fieldName,format_type(a.atttypid,a.atttypmod) as fieldType,(case when atttypmod-4>0 then
        atttypmod-4 else 0 end) as fieldLen,
        (case when (select count(*) from pg_constraint where conrelid = a.attrelid and conkey[1]=attnum and
        contype='p')>0 then 'PRI'
        when (select count(*) from pg_constraint where conrelid = a.attrelid and conkey[1]=attnum and contype='u')>0
        then 'UNI'
        when (select count(*) from pg_constraint where conrelid = a.attrelid and conkey[1]=attnum and contype='f')>0
        then 'FRI'
        else '' end) as indexType,
        (case when a.attnotnull=true then 'not null' else 'null' end) as isNullStr,
        ' comment ' || col_description(a.attrelid,a.attnum) as fieldComment
        from pg_attribute a where attstattarget=-1 and attrelid = (select c.oid from pg_class c,pg_namespace n where
        c.relnamespace=n.oid and n.nspname =namespace and relname =tablename)
        ) as string_columns
        ),','||chr(13)||chr(10));
        -- 约束
        tableScript:= tableScript || array_to_string(
        array(
        select '' union all
        select concat(' CONSTRAINT ',conname ,c ,u,p,f) from (
        select conname,
        case when contype='c' then ' CHECK('|| ( select findattname(namespace,tablename,'c') ) ||')' end as c ,
        case when contype='u' then ' UNIQUE('|| ( select findattname(namespace,tablename,'u') ) ||')' end as u ,
        case when contype='p' then ' PRIMARY KEY ('|| ( select findattname(namespace,tablename,'p') ) ||')' end as p ,
        case when contype='f' then ' FOREIGN KEY('|| ( select findattname(namespace,tablename,'u') ) ||') REFERENCES '||
        (select p.relname from pg_class p where p.oid=c.confrelid ) || '('|| ( select
        findattname(namespace,tablename,'u') ) ||')' end as f
        from pg_constraint c
        where contype in('u','c','f','p') and conrelid=(
        select oid from pg_class where relname=tablename and relnamespace =(
        select oid from pg_namespace where nspname = namespace
        )
        )
        ) as t
        ) ,',' || chr(13)||chr(10) ) || chr(13)||chr(10) ||' ); ';
        -- indexs
        -- CREATE UNIQUE INDEX pg_language_oid_index ON pg_language USING btree (oid); -- table pg_language
        --
        /** **/
        --- 获取非约束索引 column
        -- CREATE UNIQUE INDEX pg_language_oid_index ON pg_language USING btree (oid); -- table pg_language
        tableScript:= tableScript || chr(13)||chr(10) || array_to_string(
        array(
        select 'CREATE INDEX ' || indexrelname || ' ON ' || tablename || ' USING btree '|| '(' || attname || ');' from (
        SELECT
        i.relname AS indexrelname , x.indkey,
        ( select array_to_string (
        array(
        select a.attname from pg_attribute a where attrelid=c.oid and a.attnum in ( select unnest(x.indkey) )
        )
        ,',' ) )as attname
        FROM pg_class c
        JOIN pg_index x ON c.oid = x.indrelid
        JOIN pg_class i ON i.oid = x.indexrelid
        LEFT JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname=tablename and i.relname not in
        ( select constraint_name from information_schema.key_column_usage where table_name=tablename )
        )as t
        ) , chr(13)||chr(10));
        -- COMMENT ON COLUMN sys_activity.id IS '主键';
        tableScript:= tableScript || chr(13)||chr(10) || array_to_string(
        array(
        SELECT 'COMMENT ON COLUMN ' || tablename || '.' || a.attname ||' IS '|| ''''|| d.description ||''';'
        FROM pg_class c
        JOIN pg_description d ON c.oid=d.objoid
        JOIN pg_attribute a ON c.oid = a.attrelid
        WHERE c.relname=tablename
        AND a.attnum = d.objsubid), chr(13)||chr(10)) ;
        return tableScript;
        end
        $BODY$ LANGUAGE plpgsql;
        CREATE OR REPLACE FUNCTION findattname(namespace character varying, tablename character varying, ctype character
        varying)
        RETURNS character varying as $BODY$
        declare
        tt oid ;
        aname character varying default '';
        begin
        tt := oid from pg_class where relname= tablename and relnamespace =(select oid from pg_namespace where
        nspname=namespace) ;
        aname:= array_to_string(
        array(
        select a.attname from pg_attribute a
        where a.attrelid=tt and a.attnum in (
        select unnest(conkey) from pg_constraint c where contype=ctype
        and conrelid=tt and array_to_string(conkey,',') is not null
        )
        ),',');
        return aname;
        end
        $BODY$ LANGUAGE plpgsql