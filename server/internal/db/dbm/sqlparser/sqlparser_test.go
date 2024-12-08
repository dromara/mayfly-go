package sqlparser

import (
	"fmt"
	"strings"
	"testing"
)

func TestSQLSplit(t *testing.T) {

	allsql := `
	/*删除*/
	DELETE FROM t_sys_log
WHERE
  id IN (59) and name='哈哈哈' and name2="hahah;呵呵呵;";delete form tsyslog;
  -- alter sql语句
  ALTER TABLE mayfly_go.t_sys_log
DROP COLUMN delete_time;
--插入sql语句
  INSERT INTO t_sys_log VALUES(15, 2, '用户登录', '{"ip":"::1 ",\\n"username":"test_user"}', 'errCode: 401, errMsg: --您的密码安全等级较低，请修改后重新登录', '-', 0, '2024-04-23 20:00:35', 0, NULL, '');
  --插入sql语句
  INSERT INTO t_sys_log VALUES(15, 2, '用户登录', NULL, '⑴ 成孔；⑵ 砼浇筑；⑶ 桩头掏渣；⑷ 桩基检测配合;
  ⑸ 其他工作；⑹ 甲方现场要求乙方完成的其它临时工作。', '{"ip":"::1 ","username":"test_user"}', 'errCode: 401, errMsg: 您的密码安全等级较低，请修改后重新登录;\n信息嘻嘻嘻', '-', 0, '2024-04-23 20:00:35', 0, NULL, '');
  `

	err := SQLSplit(strings.NewReader(allsql), func(sql string) error {
		// if strings.Contains(sql, "INSERT") {
		// 	return fmt.Errorf("不能存在INSERT语句")
		// }
		fmt.Println(sql)
		fmt.Println()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
