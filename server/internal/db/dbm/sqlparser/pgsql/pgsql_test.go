package pgsql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"testing"
)

func TestParserSimpleSelect(t *testing.T) {
	parser := new(PgsqlParser)

	sql := `SELECT t.* FROM mayfly.sys_login_log as t  where t.id > 0  OFFSET 0 LIMIT 25;  select * from tdb where id > 1`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	stmt := stmts[1].(*sqlstmt.SimpleSelectStmt)

	t.Log(stmt.QuerySpecification.Where.GetText())
	fmt.Println(stmt.QuerySpecification.From.GetText())
	t.Log(stmts)
}

func TestParserUnionSelect(t *testing.T) {
	parser := new(PgsqlParser)

	sql := `(select sum(t.age), t.id tid, t1.id2, t1.* from T_DB t join t_db_ins as t1 on t.id = t1.id2 where t.id = 1 AND t1.status=0 and t.id2='9' and t.name in ('name2', 'name3') order by t.id desc) union all (select * from t_db2)  OFFSET 0 LIMIT 25;`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}

func TestParserSingleUpdate(t *testing.T) {
	parser := new(PgsqlParser)

	sql := `UPDATE test.t_sys_msg t
			SET
			recipient_id = 13,
			t.creator = 'admin4'
			WHERE
			t.id = 1;`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}

func TestParserDelete(t *testing.T) {
	parser := new(PgsqlParser)

	sql := `Delete from t_sys_msg t
			WHERE
			t.id = 1;`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stmts)
}

func TestParserInsert(t *testing.T) {
	parser := new(PgsqlParser)

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
