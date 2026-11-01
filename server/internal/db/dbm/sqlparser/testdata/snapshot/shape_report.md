# SQL 解析行为看板

> 由 `TestShapeReport` 生成，反映当前解析行为。列含义：type=语句类型，tables=表清单数，joins=JOIN 数，limit=分页，where=有无条件。

| SQL | 方言 | type | tables | joins | limit | where | cols |
|---|---|---|---|---|---|---|---|
| `SELECT a, b FROM t1 JOIN t2 ON t1.id = t2.id WHERE t1.x > 0 OR…` | mysql | select | 1 | 1 | 0/10 | true | 0 |
| `SELECT a, b FROM t1 JOIN t2 ON t1.id = t2.id WHERE t1.x > 0 OR…` | pgsql | select | 1 | 1 | 0/10 | true | 0 |
| `SELECT a, b FROM t1 JOIN t2 ON t1.id = t2.id WHERE t1.x > 0 OR…` | oracle | select | 1 | 1 | 0/10 | true | 0 |
| `SELECT a, b FROM t1 JOIN t2 ON t1.id = t2.id WHERE t1.x > 0 OR…` | mssql | select | 1 | 1 | 0/10 | true | 0 |
| `SELECT * FROM (SELECT id FROM t) x WHERE id IN (SELECT id FROM…` | mysql | select | 1 | 0 |  | true | 0 |
| `SELECT * FROM (SELECT id FROM t) x WHERE id IN (SELECT id FROM…` | pgsql | select | 1 | 0 |  | true | 0 |
| `SELECT * FROM (SELECT id FROM t) x WHERE id IN (SELECT id FROM…` | oracle | select | 1 | 0 |  | true | 0 |
| `SELECT * FROM (SELECT id FROM t) x WHERE id IN (SELECT id FROM…` | mssql | select | 1 | 0 |  | true | 0 |
| `WITH c AS (SELECT 1) SELECT * FROM c` | mysql | select | 0 | 0 |  | false | 0 |
| `WITH c AS (SELECT 1) SELECT * FROM c` | pgsql | select | 0 | 0 |  | false | 0 |
| `WITH c AS (SELECT 1) SELECT * FROM c` | oracle | select | 0 | 0 |  | false | 0 |
| `WITH c AS (SELECT 1) SELECT * FROM c` | mssql | select | 0 | 0 |  | false | 0 |
| `WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d` | mysql | delete | 0 | 0 |  | false | 0 |
| `WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d` | pgsql | delete | 0 | 0 |  | false | 0 |
| `WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d` | oracle | delete | 0 | 0 |  | false | 0 |
| `WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d` | mssql | delete | 0 | 0 |  | false | 0 |
| `SELECT COUNT(*) FROM t u GROUP BY u.k HAVING COUNT(*) > 1` | mysql | select | 1 | 0 |  | false | 0 |
| `SELECT COUNT(*) FROM t u GROUP BY u.k HAVING COUNT(*) > 1` | pgsql | select | 1 | 0 |  | false | 0 |
| `SELECT COUNT(*) FROM t u GROUP BY u.k HAVING COUNT(*) > 1` | oracle | select | 1 | 0 |  | false | 0 |
| `SELECT COUNT(*) FROM t u GROUP BY u.k HAVING COUNT(*) > 1` | mssql | select | 1 | 0 |  | false | 0 |
| `SELECT * FROM t1 UNION ALL SELECT * FROM t2` | mysql | select | 1 | 0 |  | false | 0 |
| `SELECT * FROM t1 UNION ALL SELECT * FROM t2` | pgsql | select | 1 | 0 |  | false | 0 |
| `SELECT * FROM t1 UNION ALL SELECT * FROM t2` | oracle | select | 1 | 0 |  | false | 0 |
| `SELECT * FROM t1 UNION ALL SELECT * FROM t2` | mssql | select | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) VALUES (1, 2)` | mysql | insert | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) VALUES (1, 2)` | pgsql | insert | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) VALUES (1, 2)` | oracle | insert | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) VALUES (1, 2)` | mssql | insert | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) SELECT id, v FROM s` | mysql | insert | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) SELECT id, v FROM s` | pgsql | insert | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) SELECT id, v FROM s` | oracle | insert | 1 | 0 |  | false | 0 |
| `INSERT INTO t (id, v) SELECT id, v FROM s` | mssql | insert | 1 | 0 |  | false | 0 |
| `UPDATE t SET v = 1 WHERE id = 2` | mysql | update | 1 | 0 |  | true | 0 |
| `UPDATE t SET v = 1 WHERE id = 2` | pgsql | update | 1 | 0 |  | true | 0 |
| `UPDATE t SET v = 1 WHERE id = 2` | oracle | update | 1 | 0 |  | true | 0 |
| `UPDATE t SET v = 1 WHERE id = 2` | mssql | update | 1 | 0 |  | true | 0 |
| `UPDATE t1 JOIN t2 ON t1.id = t2.id SET t1.v = t2.v` | mysql | update | 2 | 0 |  | false | 0 |
| `UPDATE t1 JOIN t2 ON t1.id = t2.id SET t1.v = t2.v` | pgsql | update | 2 | 0 |  | false | 0 |
| `UPDATE t1 JOIN t2 ON t1.id = t2.id SET t1.v = t2.v` | oracle | update | 2 | 0 |  | false | 0 |
| `UPDATE t1 JOIN t2 ON t1.id = t2.id SET t1.v = t2.v` | mssql | update | 2 | 0 |  | false | 0 |
| `DELETE FROM t WHERE id = 1` | mysql | delete | 1 | 0 |  | true | 0 |
| `DELETE FROM t WHERE id = 1` | pgsql | delete | 1 | 0 |  | true | 0 |
| `DELETE FROM t WHERE id = 1` | oracle | delete | 1 | 0 |  | true | 0 |
| `DELETE FROM t WHERE id = 1` | mssql | delete | 1 | 0 |  | true | 0 |
| `MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE…` | mysql | merge | 0 | 0 |  | false | 0 |
| `MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE…` | pgsql | merge | 0 | 0 |  | false | 0 |
| `MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE…` | oracle | merge | 0 | 0 |  | false | 0 |
| `MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE…` | mssql | merge | 0 | 0 |  | false | 0 |
| `EXPLAIN SELECT * FROM t` | mysql | other-read | 0 | 0 |  | false | 0 |
| `EXPLAIN SELECT * FROM t` | pgsql | other-read | 0 | 0 |  | false | 0 |
| `EXPLAIN SELECT * FROM t` | oracle | other-read | 0 | 0 |  | false | 0 |
| `EXPLAIN SELECT * FROM t` | mssql | other-read | 0 | 0 |  | false | 0 |
| `CREATE TABLE IF NOT EXISTS s1.t1 (id int, v varchar(20))` | mysql | ddl | 0 | 0 |  | false | 2 |
| `CREATE TABLE IF NOT EXISTS s1.t1 (id int, v varchar(20))` | pgsql | ddl | 0 | 0 |  | false | 2 |
| `CREATE TABLE IF NOT EXISTS s1.t1 (id int, v varchar(20))` | oracle | ddl | 0 | 0 |  | false | 2 |
| `CREATE TABLE IF NOT EXISTS s1.t1 (id int, v varchar(20))` | mssql | ddl | 0 | 0 |  | false | 2 |
| `ALTER TABLE t1 ADD COLUMN c int` | mysql | ddl | 0 | 0 |  | false | 0 |
| `ALTER TABLE t1 ADD COLUMN c int` | pgsql | ddl | 0 | 0 |  | false | 0 |
| `ALTER TABLE t1 ADD COLUMN c int` | oracle | ddl | 0 | 0 |  | false | 0 |
| `ALTER TABLE t1 ADD COLUMN c int` | mssql | ddl | 0 | 0 |  | false | 0 |
| `DROP TABLE IF EXISTS t1` | mysql | ddl | 0 | 0 |  | false | 0 |
| `DROP TABLE IF EXISTS t1` | pgsql | ddl | 0 | 0 |  | false | 0 |
| `DROP TABLE IF EXISTS t1` | oracle | ddl | 0 | 0 |  | false | 0 |
| `DROP TABLE IF EXISTS t1` | mssql | ddl | 0 | 0 |  | false | 0 |
| `RENAME TABLE a TO b, c TO d` | mysql | ddl | 0 | 0 |  | false | 0 |
| `RENAME TABLE a TO b, c TO d` | pgsql | ddl | 0 | 0 |  | false | 0 |
| `RENAME TABLE a TO b, c TO d` | oracle | ddl | 0 | 0 |  | false | 0 |
| `RENAME TABLE a TO b, c TO d` | mssql | ddl | 0 | 0 |  | false | 0 |
| `TRUNCATE TABLE t1` | mysql | ddl | 0 | 0 |  | false | 0 |
| `TRUNCATE TABLE t1` | pgsql | ddl | 0 | 0 |  | false | 0 |
| `TRUNCATE TABLE t1` | oracle | ddl | 0 | 0 |  | false | 0 |
| `TRUNCATE TABLE t1` | mssql | ddl | 0 | 0 |  | false | 0 |
| `CALL p_proc(1)` | mysql | other-write | 0 | 0 |  | false | 0 |
| `CALL p_proc(1)` | pgsql | other-write | 0 | 0 |  | false | 0 |
| `CALL p_proc(1)` | oracle | other-write | 0 | 0 |  | false | 0 |
| `CALL p_proc(1)` | mssql | other-write | 0 | 0 |  | false | 0 |
| `LOAD DATA INFILE '/tmp/a.csv' INTO TABLE t1` | mysql | other-write | 0 | 0 |  | false | 0 |
| `LOAD DATA INFILE '/tmp/a.csv' INTO TABLE t1` | pgsql | other-write | 0 | 0 |  | false | 0 |
| `LOAD DATA INFILE '/tmp/a.csv' INTO TABLE t1` | oracle | other-write | 0 | 0 |  | false | 0 |
| `LOAD DATA INFILE '/tmp/a.csv' INTO TABLE t1` | mssql | other-write | 0 | 0 |  | false | 0 |
