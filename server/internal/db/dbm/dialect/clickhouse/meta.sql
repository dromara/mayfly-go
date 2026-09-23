-- ClickHouse metadata SQL queries

-- 获取数据库名称
SELECT name FROM system.databases 
WHERE name NOT IN ('system', 'information_schema', 'INFORMATION_SCHEMA') 
ORDER BY name;

-- 获取表名称
SELECT name, engine, comment 
FROM system.tables 
WHERE database = '{{.Database}}' 
ORDER BY name;

-- 获取表列
SELECT 
    name,
    type,
    '' as column_comment,
    0 as is_nullable,
    0 as is_primary_key,
    '' as column_default,
    0 as is_auto_increment
FROM system.columns 
WHERE database = '{{.Database}}' AND table = '{{.Table}}' 
ORDER BY position;

-- 获取表索引（简化了Clickhouse）
SELECT 
    name,
    type,
    '' as comment
FROM system.indexes 
WHERE database = '{{.Database}}' AND table = '{{.Table}}';

-- 获取主要密钥信息
SELECT primary_key 
FROM system.tables 
WHERE database = '{{.Database}}' AND name = '{{.Table}}';

-- 获取创建表语句
SELECT create_table_query 
FROM system.tables 
WHERE database = '{{.Database}}' AND name = '{{.Table}}';

-- GET数据库服务器信息
SELECT 
    version() as version,
    'ClickHouse' as database,
    '' as uptime;

-- 获取数据库大小
SELECT sum(bytes) as size 
FROM system.parts 
WHERE database = '{{.Database}}';

-- 获取表行计数
SELECT sum(rows) as row_count 
FROM system.parts 
WHERE database = '{{.Database}}' AND table = '{{.Table}}';

---------------------------------------
--CLICKHOUSE_VIEWS 视图信息（system.tables 引擎为视图类；create_table_query 即完整 DDL）
SELECT name AS "viewName", create_table_query AS "viewDefinition", comment AS "viewComment"
FROM system.tables WHERE database = ? AND engine IN ('View', 'MaterializedView', 'LiveView', 'WindowView')
ORDER BY name
---------------------------------------
--CLICKHOUSE_VIEW_DDL 视图定义（create_table_query 存完整 CREATE VIEW / MATERIALIZED VIEW 语句）
SELECT create_table_query AS "viewDefinition" FROM system.tables WHERE database = ? AND name = ?
