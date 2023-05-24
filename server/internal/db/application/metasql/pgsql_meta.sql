--PGSQL_TABLE_MA 表信息元数据
SELECT
  obj_description (c.oid) AS "tableComment",
  c.relname AS "tableName"
FROM
  pg_class c
  JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE
  n.nspname = (
    select
      current_schema ()
  )
  AND c.reltype > 0

--PGSQL_TABLE_INFO 表详细信息
SELECT
  obj_description (c.oid) AS "tableComment",
  c.relname AS "tableName",
  pg_table_size ('"' || n.nspname || '"."' || c.relname || '"') as "dataLength",
  pg_indexes_size ('"' || n.nspname || '"."' || c.relname || '"') as "indexLength",
  c.reltuples as "tableRows"
FROM
  pg_class c
  JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE
  n.nspname = (
    select
      current_schema ()
  )
  AND c.reltype > 0

--PGSQL_INDEX_INFO 表索引信息
SELECT
  indexname AS "indexName",
  indexdef AS "indexComment"
FROM
  pg_indexes
WHERE
  schemaname = (
    select
      current_schema ()
  )
  AND tablename = '%s'

--PGSQL_COLUMN_MA 表列信息
SELECT
	C.relname AS "tableName",
	A.attname AS "columnName",
	tc.is_nullable AS "nullable",
	concat_ws ( '', t.typname, SUBSTRING ( format_type ( a.atttypid, a.atttypmod ) FROM '\(.*\)' ) ) AS "columnType",
	(CASE WHEN ( SELECT COUNT(*) FROM pg_constraint WHERE conrelid = a.attrelid AND conkey[1]= attnum AND contype = 'p' ) > 0 THEN 'PRI' ELSE '' END ) AS "columnKey",
	d.description AS "columnComment" 
FROM
	pg_attribute a LEFT JOIN pg_description d ON d.objoid = a.attrelid 
	AND d.objsubid = A.attnum
	LEFT JOIN pg_class c ON A.attrelid = c.oid
	LEFT JOIN pg_namespace pn ON c.relnamespace = pn.oid
	LEFT JOIN pg_type t ON a.atttypid = t.oid 
	JOIN information_schema.columns tc ON tc.column_name = a.attname AND tc.table_name = C.relname AND tc.table_schema = pn.nspname
WHERE
	A.attnum >= 0 
	AND pn.nspname = (select current_schema())
	AND C.relname in (%s)
ORDER BY
	C.relname DESC,
	A.attnum ASC

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
        select ' ' || concat_ws(' ',fieldName, fieldType, fieldLen, indexType, isNullStr, fieldComment ) as column_line
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
        ),','||chr(13)||chr(10)) || ',';
        -- 约束
        tableScript:= tableScript || chr(13)||chr(10) || array_to_string(
        array(
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
        tableScript:= tableScript || chr(13)||chr(10) || chr(13)||chr(10) || array_to_string(
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
        ) ,','|| chr(13)||chr(10));
        -- COMMENT COMMENT ON COLUMN sys_activity.id IS '主键';
        tableScript:= tableScript || chr(13)||chr(10) || chr(13)||chr(10) || array_to_string(
        array(
        SELECT 'COMMENT ON COLUMN' || tablename || '.' || a.attname ||' IS '|| ''''|| d.description ||''''
        FROM pg_class c
        JOIN pg_description d ON c.oid=d.objoid
        JOIN pg_attribute a ON c.oid = a.attrelid
        WHERE c.relname=tablename
        AND a.attnum = d.objsubid),','|| chr(13)||chr(10)) ;
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