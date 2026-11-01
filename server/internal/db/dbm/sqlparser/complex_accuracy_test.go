package sqlparser_test

import (
	"strconv"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/clickhouse"
	"mayfly-go/internal/db/dbm/sqlparser/dm"
	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlite"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// accuracyCase 一条「复杂 SQL 结构准确性」断言：表清单精确集合 + 语句分类 + 分页值。
//
// wantTables == nil 表示不校验表清单（派生表/含函数/已知边界）；空切片表示期望空。
type accuracyCase struct {
	sql        string
	wantTables []string
	wantType   sqlstmt.StmtType
	wantLimit  string // "" 不校验；否则 "offset/count"
}

var accPg = []accuracyCase{
	{"SELECT * FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a, b FROM t WHERE a > 1", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 LEFT OUTER JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 RIGHT JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 FULL OUTER JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 CROSS JOIN t2", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 NATURAL JOIN t2", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM a JOIN b USING (id) LEFT JOIN c ON c.id = a.id", []string{"a", "b", "c"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM a JOIN b ON b.id = a.id JOIN c ON c.id = b.id JOIN d ON d.id = c.id JOIN e ON e.id = d.id", []string{"a", "b", "c", "d", "e"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM (SELECT id FROM t) x", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 JOIN LATERAL (SELECT * FROM t2 WHERE t2.id = t1.id) s ON true", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT sum(x) OVER (PARTITION BY y ORDER BY z ROWS BETWEEN 1 PRECEDING AND CURRENT ROW) FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT k, sum(v) FILTER (WHERE v > 0) FROM m GROUP BY GROUPING SETS ((k), ())", []string{"m"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT k, sum(v) FROM m GROUP BY ROLLUP (k)", []string{"m"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT DISTINCT ON (a) a, b FROM t ORDER BY a, b", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM ONLY t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t TABLESAMPLE SYSTEM (1)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t TABLESAMPLE BERNOULLI (10) REPEATABLE (42)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t AS x (a, b) WHERE x.a > 0", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t FOR UPDATE OF t SKIP LOCKED", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t FOR SHARE", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1, t2, t3 WHERE t1.a = t2.a", []string{"t1", "t2", "t3"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE x = ANY(ARRAY[1,2])", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT payload->>'name' FROM j WHERE payload @> '{\"a\":1}'", []string{"j"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t LIMIT 10 OFFSET 20", []string{"t"}, sqlstmt.StmtTypeSelect, "20/10"},
	{"SELECT * FROM t OFFSET 5 ROWS FETCH NEXT 10 ROWS ONLY", []string{"t"}, sqlstmt.StmtTypeSelect, "5/10"},
	{"UPDATE t SET a = s.a FROM s", []string{"t", "s"}, sqlstmt.StmtTypeUpdate, ""},
	{"UPDATE t SET a = s.a FROM s JOIN u ON u.id = s.id", []string{"t", "s", "u"}, sqlstmt.StmtTypeUpdate, ""},
	{"UPDATE t SET a = 1 WHERE id = 2", []string{"t"}, sqlstmt.StmtTypeUpdate, ""},
	{"DELETE FROM t WHERE id = 1", []string{"t"}, sqlstmt.StmtTypeDelete, ""},
	{"DELETE FROM t USING s", []string{"t", "s"}, sqlstmt.StmtTypeDelete, ""},
	{"DELETE FROM t USING s JOIN u ON u.id = s.id", []string{"t", "s", "u"}, sqlstmt.StmtTypeDelete, ""},
	{"INSERT INTO t (id, v) VALUES (1, 2)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (id, v) SELECT id, v FROM s", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (id) VALUES (1) ON CONFLICT DO NOTHING", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (id, v) VALUES (1, 2) ON CONFLICT (id) DO UPDATE SET v = excluded.v", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (id) VALUES (1) RETURNING id", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.v = s.v", []string{"t"}, sqlstmt.StmtTypeMerge, ""},
	{"WITH c AS (SELECT 1) SELECT * FROM c", nil, sqlstmt.StmtTypeSelect, ""},
	{"WITH RECURSIVE r AS (SELECT 1 UNION ALL SELECT n + 1 FROM r) SELECT * FROM r", nil, sqlstmt.StmtTypeSelect, ""},
	{"WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d", nil, sqlstmt.StmtTypeDelete, ""},
	{"SELECT a FROM t1 UNION SELECT b FROM t2", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a FROM t1 INTERSECT SELECT a FROM t2", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a FROM t1 EXCEPT SELECT a FROM t2", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE EXISTS (SELECT 1 FROM s)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE id IN (SELECT id FROM s)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT (SELECT max(x) FROM s) FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"CREATE TABLE t (id int PRIMARY KEY, v varchar(20) NOT NULL)", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE INDEX i ON t (a)", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE ONLY t ADD COLUMN c int", nil, sqlstmt.StmtTypeDDL, ""},
	{"DROP TABLE IF EXISTS t CASCADE", nil, sqlstmt.StmtTypeDDL, ""},
	{"TRUNCATE t", nil, sqlstmt.StmtTypeDDL, ""},
	{"VACUUM t", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"COPY t FROM '/tmp/a.csv'", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"CALL p_proc(1)", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"EXPLAIN SELECT * FROM t", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"EXPLAIN ANALYZE SELECT * FROM t", nil, sqlstmt.StmtTypeOtherWrite, ""},
}

var accMy = []accuracyCase{
	{"SELECT * FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT `a`, `b` FROM `t` WHERE `a` > 1", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 LEFT JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 STRAIGHT_JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t PARTITION (p0) WHERE id > 0", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t USE INDEX (i) JOIN u FORCE INDEX (j) ON u.id = t.id", []string{"t", "u"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t IGNORE INDEX (i) WHERE a = 1", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t FORCE INDEX FOR ORDER BY (i) ORDER BY a", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM (SELECT id FROM t) x", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1, t2, t3 WHERE t1.a = t2.a", []string{"t1", "t2", "t3"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE MATCH(a) AGAINST ('x')", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE a REGEXP '^x'", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT JSON_EXTRACT(j, '$.a') FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t LIMIT 10", []string{"t"}, sqlstmt.StmtTypeSelect, "0/10"},
	{"SELECT * FROM t LIMIT 10, 20", []string{"t"}, sqlstmt.StmtTypeSelect, "10/20"},
	{"SELECT * FROM t ORDER BY a DESC LIMIT 5", []string{"t"}, sqlstmt.StmtTypeSelect, "0/5"},
	{"SELECT * FROM t WHERE id IN (1,2,3) AND (a = 1 OR b BETWEEN 1 AND 2)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE id IN (SELECT id FROM s)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE EXISTS (SELECT 1 FROM s)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a FROM t1 UNION ALL SELECT b FROM t2 LIMIT 10", []string{"t1"}, sqlstmt.StmtTypeSelect, "0/10"},
	{"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id JOIN t3 ON t3.id = t2.id", []string{"t1", "t2", "t3"}, sqlstmt.StmtTypeSelect, ""},
	{"UPDATE t SET a = 1 WHERE id = 2", []string{"t"}, sqlstmt.StmtTypeUpdate, ""},
	{"UPDATE t SET a = 1 WHERE id = 2 LIMIT 10", []string{"t"}, sqlstmt.StmtTypeUpdate, "0/10"},
	{"UPDATE t1 JOIN t2 ON t1.id = t2.id SET t1.v = t2.v", []string{"t1", "t2"}, sqlstmt.StmtTypeUpdate, ""},
	{"UPDATE t1, t2 SET t1.a = t2.a WHERE t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeUpdate, ""},
	{"DELETE FROM t WHERE id = 1", []string{"t"}, sqlstmt.StmtTypeDelete, ""},
	{"DELETE FROM t WHERE id = 1 LIMIT 5", []string{"t"}, sqlstmt.StmtTypeDelete, "0/5"},
	{"DELETE t1 FROM t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeDelete, ""},
	{"DELETE FROM t1, t2 USING t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeDelete, ""},
	{"INSERT INTO t (a, b) VALUES (1, 2)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t SET a = 1, b = 2", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (a) SELECT a FROM s", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT IGNORE INTO t (a) VALUES (1)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"REPLACE INTO t (a) VALUES (1)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (a) VALUES (1) ON DUPLICATE KEY UPDATE a = 2", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"WITH c AS (SELECT 1) SELECT * FROM c", nil, sqlstmt.StmtTypeSelect, ""},
	{"CREATE TABLE t (id int PRIMARY KEY, v varchar(20)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE TABLE t LIKE u", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE TABLE t AS SELECT * FROM u", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE t ADD INDEX idx (a), ADD COLUMN b int", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE t RENAME TO t2", nil, sqlstmt.StmtTypeDDL, ""},
	{"RENAME TABLE a TO b, c TO d", nil, sqlstmt.StmtTypeDDL, ""},
	{"DROP TABLE IF EXISTS t", nil, sqlstmt.StmtTypeDDL, ""},
	{"TRUNCATE TABLE t", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE PROCEDURE p() BEGIN UPDATE t SET v = 1; DELETE FROM s; END", nil, sqlstmt.StmtTypeDDL, ""},
	{"SHOW TABLES", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"SHOW CREATE TABLE t", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"DESC t", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"EXPLAIN SELECT * FROM t", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"EXPLAIN ANALYZE DELETE FROM t", nil, sqlstmt.StmtTypeDelete, ""},
	{"SET @a = 1", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"BEGIN", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"COMMIT", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"CALL p_proc(1)", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"LOAD DATA INFILE '/tmp/a.csv' INTO TABLE t", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"PREPARE s FROM 'SELECT 1'", nil, sqlstmt.StmtTypeOtherWrite, ""},
}

var accCommon = []accuracyCase{
	{"SELECT * FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a FROM t WHERE a = 1", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 LEFT JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t LIMIT 5", []string{"t"}, sqlstmt.StmtTypeSelect, "0/5"},
	{"SELECT * FROM t ORDER BY a LIMIT 10", []string{"t"}, sqlstmt.StmtTypeSelect, "0/10"},
	{"SELECT COUNT(*) FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a, b FROM t GROUP BY a, b HAVING COUNT(*) > 1", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 UNION SELECT * FROM t2", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"INSERT INTO t (a) VALUES (1)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (a) SELECT a FROM s", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"UPDATE t SET a = 1 WHERE id = 2", []string{"t"}, sqlstmt.StmtTypeUpdate, ""},
	{"DELETE FROM t WHERE id = 1", []string{"t"}, sqlstmt.StmtTypeDelete, ""},
	{"SELECT * FROM (SELECT id FROM t) x", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE id IN (SELECT id FROM s)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE EXISTS (SELECT 1 FROM s)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"WITH c AS (SELECT 1) SELECT * FROM c", nil, sqlstmt.StmtTypeSelect, ""},
	{"CREATE TABLE t (id int PRIMARY KEY)", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE t ADD COLUMN c int", nil, sqlstmt.StmtTypeDDL, ""},
	{"DROP TABLE t", nil, sqlstmt.StmtTypeDDL, ""},
	{"TRUNCATE TABLE t", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE INDEX i ON t (a)", nil, sqlstmt.StmtTypeDDL, ""},
	{"SELECT * FROM t1, t2", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE a > 1 AND b < 2 OR c = 3", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE a IS NULL", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT DISTINCT a FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a AS x FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"UPDATE t SET a = a + 1", []string{"t"}, sqlstmt.StmtTypeUpdate, ""},
	{"DELETE FROM t", []string{"t"}, sqlstmt.StmtTypeDelete, ""},
	{"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id WHERE t1.a > 0 GROUP BY t2.b HAVING COUNT(*) > 1 ORDER BY t2.b LIMIT 5", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, "0/5"},
}

var accOr = []accuracyCase{
	{"SELECT * FROM e, d WHERE e.id = d.id(+) AND e.st = 1", []string{"e", "d"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT id, LEVEL FROM t START WITH pid IS NULL CONNECT BY PRIOR id = pid ORDER SIBLINGS BY name", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE ROWNUM <= 10", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t SAMPLE (1)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t PIVOT (SUM(x) FOR y IN ('a','b'))", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM (SELECT a.*, ROWNUM rn FROM (SELECT * FROM t ORDER BY id) a WHERE ROWNUM <= 20) WHERE rn > 10", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM dual", []string{"dual"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t OFFSET 5 ROWS FETCH NEXT 10 ROWS ONLY", []string{"t"}, sqlstmt.StmtTypeSelect, "5/10"},
	{"SELECT * FROM t FETCH FIRST 10 ROWS ONLY", []string{"t"}, sqlstmt.StmtTypeSelect, "0/10"},
	{"SELECT a FROM t1 MINUS SELECT a FROM t2", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT DECODE(a, 1, 'x', 'y'), NVL(b, 0), CASE WHEN c > 0 THEN 1 ELSE 0 END FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT LISTAGG(name, ',') WITHIN GROUP (ORDER BY name) FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT `ID` FROM `T` WHERE `ID` > 0", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"UPDATE t SET a = (SELECT MAX(x) FROM s)", []string{"t"}, sqlstmt.StmtTypeUpdate, ""},
	{"DELETE FROM t WHERE ROWNUM <= 10", []string{"t"}, sqlstmt.StmtTypeDelete, ""},
	{"INSERT INTO t (a) SELECT a FROM s WHERE b > 0", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.v = s.v WHEN NOT MATCHED THEN INSERT (id, v) VALUES (s.id, s.v)", []string{"t"}, sqlstmt.StmtTypeMerge, ""},
	{"SELECT * FROM t1 WHERE a IN (SELECT a FROM t2 WHERE b IN (SELECT b FROM t3))", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE a LIKE 'x%' AND b IS NOT NULL", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"COMMIT", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"ROLLBACK", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"EXPLAIN PLAN FOR SELECT * FROM t", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"CREATE TABLE t (id NUMBER(10) PRIMARY KEY, v VARCHAR2(20))", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE t ADD (c NUMBER(5))", nil, sqlstmt.StmtTypeDDL, ""},
	{"DROP TABLE t PURGE", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE OR REPLACE VIEW v AS SELECT * FROM t", nil, sqlstmt.StmtTypeDDL, ""},
}

var accMs = []accuracyCase{
	{"SELECT * FROM t WITH (NOLOCK) WHERE id > 0", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT TOP 10 * FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, "0/10"},
	{"SELECT TOP (10) * FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, "0/10"},
	{"SELECT TOP (10) PERCENT * FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, "0/0"},
	{"SELECT * FROM t ORDER BY id OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY", []string{"t"}, sqlstmt.StmtTypeSelect, "20/10"},
	{"SELECT [a].[id] FROM [dbo].[t] AS [a] WHERE [a].[v] > 0", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM [t]]x]", []string{"t]x"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 CROSS APPLY (SELECT TOP 1 * FROM t2 WHERE t2.id = t1.id) x", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 OUTER APPLY (SELECT * FROM t2) x", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 CROSS APPLY fn(t1.id) f", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"UPDATE t SET a = s.a FROM s WHERE s.id = t.id", []string{"t", "s"}, sqlstmt.StmtTypeUpdate, ""},
	{"DELETE t1 FROM t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeDelete, ""},
	{"SELECT * FROM t1 JOIN t2 WITH (NOLOCK) ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t PIVOT (SUM(x) FOR y IN ([a],[b])) p", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM (SELECT ROW_NUMBER() OVER (ORDER BY id) rn, t.* FROM t) x WHERE rn BETWEEN 11 AND 20", nil, sqlstmt.StmtTypeSelect, ""},
	{"MERGE INTO t AS tgt USING s AS src ON tgt.id = src.id WHEN MATCHED THEN UPDATE SET tgt.v = src.v OUTPUT inserted.id", []string{"t"}, sqlstmt.StmtTypeMerge, ""},
	{"INSERT INTO t (a) VALUES (1)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"CREATE TABLE t (id int IDENTITY(1,1) PRIMARY KEY, v nvarchar(20))", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE NONCLUSTERED INDEX i ON t (a)", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE t ADD c int", nil, sqlstmt.StmtTypeDDL, ""},
	{"EXEC p_proc 1", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"DECLARE @a int", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"SET @a = 1", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"SELECT * FROM t1 UNION ALL SELECT * FROM t2 ORDER BY 1 OFFSET 0 ROWS FETCH NEXT 5 ROWS ONLY", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE a IN (1, 2, 3)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
}

var accSl = []accuracyCase{
	{"PRAGMA table_info(t)", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"INSERT OR REPLACE INTO t (a) VALUES (1)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT OR IGNORE INTO t (a) VALUES (1)", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"INSERT INTO t (a) VALUES (1) ON CONFLICT(a) DO UPDATE SET a = excluded.a", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"SELECT * FROM t LIMIT -1 OFFSET 10", []string{"t"}, sqlstmt.StmtTypeSelect, "10/-1"},
	{"VACUUM", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"ANALYZE", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"BEGIN", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT) WITHOUT ROWID", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE INDEX IF NOT EXISTS i ON t (a)", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE t RENAME TO t2", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE VIEW v AS SELECT * FROM t", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE TRIGGER trg AFTER INSERT ON t BEGIN UPDATE t SET v = 1; END", nil, sqlstmt.StmtTypeDDL, ""},
	{"SELECT * FROM t WHERE a GLOB 'x*'", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"WITH RECURSIVE r AS (SELECT 1 UNION ALL SELECT n + 1 FROM r) SELECT * FROM r", nil, sqlstmt.StmtTypeSelect, ""},
	{"SELECT ROW_NUMBER() OVER (PARTITION BY a ORDER BY b) FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"UPDATE t SET a = s.a FROM s WHERE s.id = t.id", []string{"t", "s"}, sqlstmt.StmtTypeUpdate, ""},
	{"DELETE FROM t RETURNING id", []string{"t"}, sqlstmt.StmtTypeDelete, ""},
	{"EXPLAIN QUERY PLAN SELECT * FROM t", nil, sqlstmt.StmtTypeOtherRead, ""},
	{"ATTACH DATABASE '/tmp/a.db' AS a", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"SELECT * FROM t1 JOIN t2 USING (id)", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT a, SUM(b) FROM t GROUP BY a", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE a LIKE '%x%' ESCAPE '\\'", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t LIMIT 20", []string{"t"}, sqlstmt.StmtTypeSelect, "0/20"},
	{"SELECT * FROM t1, t2 WHERE t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
}

var accCh = []accuracyCase{
	{"SELECT * FROM t FINAL PREWHERE a > 0 WHERE b = 1 ARRAY JOIN arr AS x LIMIT 10", []string{"t"}, sqlstmt.StmtTypeSelect, "0/10"},
	{"SELECT * FROM t SETTINGS max_threads = 1", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t SAMPLE 0.1", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 ASOF JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 ANY LEFT JOIN t2 USING (id)", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT k, count() FROM t GROUP BY k WITH TOTALS", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT k, count() FROM t GROUP BY k WITH ROLLUP", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t LIMIT 5 BY k", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM `db`.`t` LIMIT 5", []string{"t"}, sqlstmt.StmtTypeSelect, "0/5"},
	{"INSERT INTO t SELECT * FROM s", []string{"t"}, sqlstmt.StmtTypeInsert, ""},
	{"ALTER TABLE t UPDATE v = 1 WHERE id = 2", nil, sqlstmt.StmtTypeDDL, ""},
	{"ALTER TABLE t DELETE WHERE id = 2", nil, sqlstmt.StmtTypeDDL, ""},
	{"OPTIMIZE TABLE t FINAL", nil, sqlstmt.StmtTypeDDL, ""},
	{"CREATE MATERIALIZED VIEW mv TO t AS SELECT * FROM s", nil, sqlstmt.StmtTypeDDL, ""},
	{"SELECT arrayMap(x -> x + 1, arr) FROM t", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 GLOBAL JOIN t2 USING (id)", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t1 LEFT ARRAY JOIN arr AS x", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE a = 1 AND b IN (1, 2)", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t LIMIT 10 OFFSET 5", []string{"t"}, sqlstmt.StmtTypeSelect, "5/10"},
	{"SELECT * FROM t1 UNION ALL SELECT * FROM t2", []string{"t1"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT toDate(ts) d, count() FROM t GROUP BY d ORDER BY d", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"SELECT * FROM t WHERE ts >= now() - INTERVAL 1 DAY", []string{"t"}, sqlstmt.StmtTypeSelect, ""},
	{"TRUNCATE TABLE t", nil, sqlstmt.StmtTypeDDL, ""},
	{"SYSTEM FLUSH LOGS", nil, sqlstmt.StmtTypeOtherWrite, ""},
	{"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}, sqlstmt.StmtTypeSelect, ""},
}

// TestComplexSQLAccuracy 复杂 SQL 的结构准确性回归库（每方言 50+ 条）。
//
// 断言的是**被消费的字段**：表清单（表级 ACL / 「指定表」策略的唯一依据）、语句分类
// （读写通道与审批）、分页（采集层行限制守卫）。已知边界（派生表内层/条件子查询/
// 表值函数）不进外层表清单，由 aclTableScope 判为不可信并 fail-closed，不在此断言。
func TestComplexSQLAccuracy(t *testing.T) {
	withCommon := func(cases []accuracyCase) []accuracyCase {
		return append(append([]accuracyCase{}, cases...), accCommon...)
	}

	run := func(name string, parser sqlparser.SQLParser, cfg tokenizer.DialectConfig, cases []accuracyCase) {
		splitter := sqlparser.NewSplitter(cfg)
		for _, c := range cases {
			stmt, err := parser.Parse(c.sql)
			if err != nil {
				t.Errorf("%s 解析失败: %q -> %v", name, c.sql, err)
				continue
			}
			typ := sqlparser.Classify(parser, splitter, c.sql, stmt, err)
			if typ != c.wantType {
				t.Errorf("%s 分类错误: %q -> %q，期望 %q", name, c.sql, typ, c.wantType)
			}
			if c.wantTables != nil {
				if got := tablesOf(stmt); !sameSet(got, c.wantTables) {
					t.Errorf("%s 表清单错误: %q -> %v，期望 %v", name, c.sql, got, c.wantTables)
				}
			}
			if c.wantLimit != "" {
				if got := limitOf(stmt); got != c.wantLimit {
					t.Errorf("%s 分页错误: %q -> %q，期望 %q", name, c.sql, got, c.wantLimit)
				}
			}
		}
	}

	run("pgsql", new(pgsql.PgsqlParser), tokenizer.PgConfig, withCommon(accPg))
	run("mysql", new(mysql.MysqlParser), tokenizer.MysqlConfig, withCommon(accMy))
	run("oracle", new(oracle.OracleParser), tokenizer.OracleConfig, withCommon(accOr))
	run("dm", new(dm.DmParser), tokenizer.DmConfig, withCommon(accOr))
	run("mssql", new(mssql.MssqlParser), tokenizer.MssqlConfig, withCommon(accMs))
	run("sqlite", new(sqlite.SqliteParser), tokenizer.SQLiteConfig, withCommon(accSl))
	run("clickhouse", new(clickhouse.ClickhouseParser), tokenizer.ClickhouseConfig, withCommon(accCh))
}

func tablesOf(stmt sqlstmt.Stmt) []string {
	var refs []sqlstmt.TableRef
	switch v := stmt.(type) {
	case *sqlstmt.SelectStmt:
		refs = append(refs, v.From...)
		for _, j := range v.Joins {
			refs = append(refs, j.Table)
		}
	case *sqlstmt.UpdateStmt:
		refs = append(refs, v.Tables...)
	case *sqlstmt.DeleteStmt:
		refs = append(refs, v.Tables...)
	case *sqlstmt.InsertStmt:
		refs = append(refs, v.Table)
	case *sqlstmt.MergeStmt:
		if v.Target != "" {
			names := strings.SplitN(v.Target, ".", 2)
			refs = append(refs, sqlstmt.TableRef{Name: names[len(names)-1]})
		}
	}
	names := make([]string, 0, len(refs))
	for _, r := range refs {
		if n := strings.TrimSpace(r.Name); n != "" && !strings.HasPrefix(n, "(") {
			names = append(names, strings.ToLower(n))
		}
	}
	return names
}

func limitOf(stmt sqlstmt.Stmt) string {
	type limiter interface{ GetLimit() *sqlstmt.Limit }
	switch v := stmt.(type) {
	case *sqlstmt.SelectStmt:
		if v.Limit != nil {
			return itoa(v.Limit.Offset) + "/" + itoa(v.Limit.Count)
		}
	case *sqlstmt.UpdateStmt:
		if v.Limit != nil {
			return itoa(v.Limit.Offset) + "/" + itoa(v.Limit.Count)
		}
	case *sqlstmt.DeleteStmt:
		if v.Limit != nil {
			return itoa(v.Limit.Offset) + "/" + itoa(v.Limit.Count)
		}
	}
	return ""
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		seen[v]--
		if seen[v] < 0 {
			return false
		}
	}
	return true
}
