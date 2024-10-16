package mysql

import (
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"testing"
)

func TestParserSimpleSelect(t *testing.T) {
	parser := new(MysqlParser)

	// sql := "select sum(t.age), t.`id` tid, t1.id2, t1.* from T_DB t left join t_db_ins as t1 on t.id = t1.id2 where t.id = 1 AND t1.status=0 and t.id2='9' and t.name in ('name2', 'name3') order by t.id desc limit 0, 100"
	sql := "SELECT t.* FROM `t_sys_resource` t WHERE t.`id` > 0"
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}

func TestParserUnionSelect(t *testing.T) {
	parser := new(MysqlParser)

	sql := "(select sum(t.age), t.id tid, t1.id2, t1.* from T_DB t join t_db_ins as t1 on t.id = t1.id2 where t.id = 1 AND t1.status=0 and t.id2='9' and t.name in ('name2', 'name3') order by t.id desc limit 0, 100) union all (select * from t_db2) limit 10"
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}

func TestParserSingleUpdate(t *testing.T) {
	parser := new(MysqlParser)

	sql := `UPDATE t_sys_msg t
			SET
			t.recipient_id = 13,
			t.creator = 'admin4'
			WHERE
			t.id = 1;`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}

func TestParserInsert(t *testing.T) {
	parser := new(MysqlParser)

	sql := `INSERT INTO
  mayfly_go.t_sys_msg (
    type,
    msg,
    recipient_id,
    creator_id,
    create_time,
    is_deleted
  )
VALUES
  (1, 'hahaha', 2, 1, '2024-08-26 15:36:27', 0);`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}

func TestParserSql(t *testing.T) {
	parser := new(MysqlParser)

	// 	sql := `INSERT INTO
	//   mayfly_go.t_sys_msg (
	//     type,
	//     msg,
	//     recipient_id,
	//     creator_id,
	//     create_time,
	//     is_deleted
	//   )
	// VALUES
	//   (1, 'hahaha', 2, 1, '2024-08-26 15:36:27', 0);`

	sql := `UPDATE t_sys_msg
			SET
			recipient_id = 13,
			creator = 'admin4'
			WHERE
			id = 1;`

	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}

	switch stmt := stmts[0].(type) {
	case *sqlstmt.InsertStmt:
		t.Log("insert")
		t.Log(stmt.TableName.Identifier.Value)
	case *sqlstmt.UpdateStmt:
		t.Log("update")
	case *sqlstmt.DeleteStmt:
		t.Log("delete")
	case *sqlstmt.SelectStmt:
		t.Log("select")
	default:
		t.Log("other")
	}
	t.Log(stmts)
}

func TestParserDelete(t *testing.T) {
	parser := new(MysqlParser)

	sql := `DELETE FROM t_sys_log
WHERE
  id IN (59);`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}
