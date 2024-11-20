package base

import (
	"fmt"
	"testing"
)

func TestParserSql(t *testing.T) {
	sql := `-- selectByCond
	Select * from tdb where id > 10;
	-- another comment
	Select * from another_table where name = 'test'
	and age = ?;
	-- multi-line comment
	-- continues here
	Select * from yet_another_table
	Where id = ?;`

	statements, err := parseSQL(sql)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, stmt := range statements {
		fmt.Printf("Comment: %s\nSQL: %s\n\n", stmt.Comment, stmt.SQL)
	}
}
