package sync

import (
	"context"
	"testing"

	_ "mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
)

// TestValidateDataSyncSql DataSql与增量字段合法性校验矩阵
func TestValidateDataSyncSql(t *testing.T) {
	mysql := dbi.GetDialect("mysql")
	pg := dbi.GetDialect("postgres")
	if mysql == nil || pg == nil {
		t.Fatal("mysql/postgres dialect should not be nil")
	}

	newTask := func(dataSql string) *entity.DataSyncTask {
		return &entity.DataSyncTask{DataSql: dataSql}
	}

	cases := []struct {
		name    string
		dialect dbi.Dialect
		task    *entity.DataSyncTask
		wantErr bool
	}{
		{name: "plain select", dialect: pg, task: newTask("select * from t_order")},
		{name: "select with where", dialect: pg, task: newTask("select id, name from t_order where status = 1")},
		{name: "mysql backquote", dialect: mysql, task: newTask("select `id`, `name` from `t_order` where `id` > 1")},
		{name: "with cte", dialect: pg, task: newTask("with cte as (select id from t_order) select * from cte")},
		{name: "insert rejected", dialect: pg, task: newTask("insert into t_order(id) values (1)"), wantErr: true},
		{name: "update rejected", dialect: pg, task: newTask("update t_order set status = 2"), wantErr: true},
		{name: "delete rejected", dialect: pg, task: newTask("delete from t_order"), wantErr: true},
		{name: "drop rejected", dialect: pg, task: newTask("drop table t_order"), wantErr: true},
		{name: "multi statement rejected", dialect: pg, task: newTask("select 1; select 2"), wantErr: true},
		{name: "multi statement with trailing drop rejected", dialect: pg, task: newTask("select * from t_order; drop table t_order"), wantErr: true},
		{name: "empty rejected", dialect: pg, task: newTask("   "), wantErr: true},
		{name: "invalid updField semicolon", dialect: pg, task: &entity.DataSyncTask{DataSql: "select * from t", UpdField: "id;drop table t"}, wantErr: true},
		{name: "invalid updField leading digit", dialect: pg, task: &entity.DataSyncTask{DataSql: "select * from t", UpdField: "1abc"}, wantErr: true},
		{name: "invalid updField space", dialect: pg, task: &entity.DataSyncTask{DataSql: "select * from t", UpdField: "id desc"}, wantErr: true},
		{name: "invalid updFieldSrc", dialect: pg, task: &entity.DataSyncTask{DataSql: "select * from t", UpdFieldSrc: "a.id;delete"}, wantErr: true},
		{name: "valid updField qualified", dialect: pg, task: &entity.DataSyncTask{DataSql: "select * from t", UpdField: "a.id"}, wantErr: false},
		{name: "valid updField underscore", dialect: pg, task: &entity.DataSyncTask{DataSql: "select * from t", UpdField: "_created_time"}, wantErr: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateDataSyncSql(c.dialect, c.task)
			if (err != nil) != c.wantErr {
				t.Fatalf("validateDataSyncSql(%q) error = %v, wantErr %v", c.task.DataSql, err, c.wantErr)
			}
		})
	}
}

// TestDataSqlHasWhere where判定：解析器优先，替代旧(?i)where正则的误匹配
func TestDataSqlHasWhere(t *testing.T) {
	pg := dbi.GetDialect("postgres")
	if pg == nil {
		t.Fatal("postgres dialect should not be nil")
	}

	cases := []struct {
		dataSql string
		want    bool
	}{
		{"select * from t where id > 1", true},
		// 字符串字面量含where：旧正则会误判
		{"select * from t where remark = 'where'", true},
		// 字段名含where子串但无where条件：旧正则会误判为true，解析器应正确返回false
		{"select nowhere_col from t", false},
		{"select * from somewhere_table", false},
		{"select * from t", false},
	}

	for _, c := range cases {
		if got := dataSqlHasWhere(pg, &entity.DataSyncTask{DataSql: c.dataSql}); got != c.want {
			t.Errorf("dataSqlHasWhere(%q) = %v, want %v", c.dataSql, got, c.want)
		}
	}

	// WITH语句解析结果无法内省where，降级正则兜底
	withHasWhere := "with cte as (select id from t) select * from cte where id > 1"
	if !dataSqlHasWhere(pg, &entity.DataSyncTask{DataSql: withHasWhere}) {
		t.Errorf("with statement with where should fallback to regex and return true")
	}
	withNoWhere := "with cte as (select id from t) select * from cte"
	if dataSqlHasWhere(pg, &entity.DataSyncTask{DataSql: withNoWhere}) {
		t.Errorf("with statement without where should return false")
	}

	// 方言为nil时降级正则
	if !dataSqlHasWhere(nil, &entity.DataSyncTask{DataSql: "select * from t where id = 1"}) {
		t.Errorf("nil dialect should fallback to regex and return true")
	}
}

// fakeDataSyncRepo 嵌入接口仅覆写UpdateById：未覆写方法被调用时panic即暴露问题
type fakeDataSyncRepo struct {
	repository.DataSyncTask

	updated []*entity.DataSyncTask
	err     error
}

func (f *fakeDataSyncRepo) UpdateById(ctx context.Context, e *entity.DataSyncTask, columns ...string) error {
	if f.err != nil {
		return f.err
	}
	f.updated = append(f.updated, e)
	return nil
}

// TestValidateDataSyncSqlComplexLiterals 真实业务复杂SQL：字符串字面量含分号/注释样式、
// pg专属操作符（->> @> :: ILIKE 数组下标 双引号标识符）、mysql反引号标识符含分号、
// sqlite函数内多参数字面量等均不得误拒（误拒会阻断合法存量任务），
// 而借字面量伪装的注入/多语句必须拦截
func TestValidateDataSyncSqlComplexLiterals(t *testing.T) {
	pg := dbi.GetDialect("postgres")
	mysql := dbi.GetDialect("mysql")
	sqlite := dbi.GetDialect("sqlite")
	if pg == nil || mysql == nil || sqlite == nil {
		t.Fatal("postgres/mysql/sqlite dialect should not be nil")
	}

	cases := []struct {
		name     string
		dialect  dbi.Dialect
		dataSql  string
		wantErr  bool
		hasWhere bool // 期望的where判定（仅校验通过时有意义）
	}{
		{name: "pg值含分号", dialect: pg, dataSql: `SELECT * FROM t WHERE a = 'x;y'`, hasWhere: true},
		{name: "pg值含行注释样式", dialect: pg, dataSql: `SELECT * FROM t WHERE a = '--b'`, hasWhere: true},
		{name: "pg值含块注释样式", dialect: pg, dataSql: `SELECT * FROM t WHERE a = '/*x*/'`, hasWhere: true},
		{name: "pg jsonb取值", dialect: pg, dataSql: `SELECT id, data->>'name' AS n FROM t`, hasWhere: false},
		{name: "pg jsonb包含判断含分号值", dialect: pg, dataSql: `SELECT id FROM t WHERE data @> '{"k":"v;"}'`, hasWhere: true},
		{name: "pg类型转换", dialect: pg, dataSql: `SELECT id, meta::jsonb AS m FROM t WHERE id > 1`, hasWhere: true},
		{name: "pg ILIKE通配", dialect: pg, dataSql: `SELECT id FROM t WHERE name ILIKE '%a;b%'`, hasWhere: true},
		{name: "pg双引号标识符+JOIN", dialect: pg, dataSql: `SELECT a.id FROM "MyTable" a JOIN t b ON a.id = b.id WHERE a.x = 1`, hasWhere: true},
		{name: "pg数组下标", dialect: pg, dataSql: `SELECT tags[1] FROM t WHERE id = 1`, hasWhere: true},
		{name: "pg多行WITH含分号值", dialect: pg, dataSql: "WITH c AS (SELECT id FROM t WHERE a='x;y')\nSELECT * FROM c", hasWhere: true},
		{name: "pg尾注释含分号不计为多语句", dialect: pg, dataSql: `SELECT id FROM t; -- tail; comment`, hasWhere: false},
		{name: "pg多行缩进+前导注释", dialect: pg, dataSql: "-- c\nSELECT id\n  FROM t\n WHERE a = 1;", hasWhere: true},

		{name: "mysql反引号表名含分号", dialect: mysql, dataSql: "SELECT `id` FROM `t;x` WHERE `a` = 'b;c'", hasWhere: true},
		{name: "mysql保留字列名", dialect: mysql, dataSql: "SELECT `id`, `desc` FROM `t` WHERE `group` = 1", hasWhere: true},
		// 无where且顶层有group by/limit：运行器会追加"where 1=1"落在尾部子句之后，必须拒绝
		{name: "mysql GROUP BY HAVING无where拒绝", dialect: mysql, dataSql: "SELECT a, COUNT(*) c FROM t GROUP BY a HAVING COUNT(*) > 1", wantErr: true},
		{name: "mysql JSON路径含点与引号", dialect: mysql, dataSql: "SELECT JSON_EXTRACT(data, '$.k') FROM t WHERE id = 1", hasWhere: true},
		{name: "mysql LIMIT OFFSET无where拒绝", dialect: mysql, dataSql: "SELECT id FROM t ORDER BY id LIMIT 10 OFFSET 20", wantErr: true},

		{name: "sqlite值含分号", dialect: sqlite, dataSql: `SELECT id FROM t WHERE name = 'a;b' LIMIT 10`, hasWhere: true},
		{name: "sqlite多参数字面量函数", dialect: sqlite, dataSql: `SELECT id FROM t WHERE created > datetime('now','-1 day')`, hasWhere: true},

		// 借字面量伪装的破坏性语句：必须拦截
		{name: "真多语句-分号后drop", dialect: pg, dataSql: `SELECT id FROM t; DROP TABLE t`, wantErr: true},
		{name: "真多语句-分号后insert", dialect: mysql, dataSql: "SELECT `id` FROM `t`; INSERT INTO t VALUES (1)", wantErr: true},
		{name: "多语句且首条为delete", dialect: sqlite, dataSql: "DELETE FROM t; SELECT 1", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			task := &entity.DataSyncTask{DataSql: c.dataSql}
			err := validateDataSyncSql(c.dialect, task)
			if (err != nil) != c.wantErr {
				t.Fatalf("validateDataSyncSql(%q) error = %v, wantErr %v", c.dataSql, err, c.wantErr)
			}
			if !c.wantErr && dataSqlHasWhere(c.dialect, task) != c.hasWhere {
				t.Fatalf("dataSqlHasWhere(%q) = %v, want %v", c.dataSql, !c.hasWhere, c.hasWhere)
			}
		})
	}
}

// TestValidateDataSyncSqlAppendable 尾部拼接合法性校验：
// 运行器按 `dataSql [where 1=1] [and upd > val] [order by upd asc]` 尾部追加，
// 顶层GROUP BY/HAVING/ORDER BY/LIMIT会使追加内容落在其后产生非法SQL
func TestValidateDataSyncSqlAppendable(t *testing.T) {
	pg := dbi.GetDialect("postgres")
	if pg == nil {
		t.Fatal("postgres dialect should not be nil")
	}

	newTask := func(dataSql, updField string) *entity.DataSyncTask {
		return &entity.DataSyncTask{DataSql: dataSql, UpdField: updField}
	}

	cases := []struct {
		name    string
		dataSql string
		upd     string
		wantErr bool
	}{
		// 未配置增量字段：仅在无where时追加where 1=1
		{name: "无增量有where直接可用", dataSql: "select id from t where id > 1", upd: "", wantErr: false},
		{name: "无增量无where无尾部子句", dataSql: "select id from t", upd: "", wantErr: false},
		{name: "无增量无where顶层group by拒绝", dataSql: "select a, count(*) from t group by a", upd: "", wantErr: true},
		{name: "无增量无where顶层order by拒绝", dataSql: "select id from t order by id", upd: "", wantErr: true},
		{name: "无增量无where顶层limit拒绝", dataSql: "select id from t limit 10", upd: "", wantErr: true},
		// 子查询内的group by/order by不影响（仅检查顶层）
		{name: "无增量无where子查询含order by可用", dataSql: "select id from (select id from t order by id limit 5) sub", upd: "", wantErr: false},

		// 配置增量字段：追加and/order by，顶层尾部子句一律拒绝
		{name: "增量有where无尾部子句", dataSql: "select id, updated from t where updated > '2026-01-01'", upd: "updated", wantErr: false},
		{name: "增量无where无尾部子句", dataSql: "select id, updated from t", upd: "updated", wantErr: false},
		{name: "增量有where顶层group by拒绝", dataSql: "select a, max(updated) from t where id > 1 group by a", upd: "updated", wantErr: true},
		{name: "增量有where顶层order by拒绝", dataSql: "select id from t where id > 1 order by id", upd: "id", wantErr: true},
		{name: "增量有where顶层limit拒绝", dataSql: "select id from t where id > 1 limit 10", upd: "id", wantErr: true},
		{name: "增量有where子查询含order by可用", dataSql: "select id, updated from t where id in (select id from log order by id desc limit 100)", upd: "updated", wantErr: false},

		// WITH语句：无法内省顶层形态，配置增量字段一律拒绝；无增量时无where且有尾部子句拒绝
		{name: "WITH无增量无尾部子句", dataSql: "with cte as (select id from t) select * from cte", upd: "", wantErr: false},
		{name: "WITH无增量有where不追加", dataSql: "with cte as (select id from t where id > 1) select * from cte", upd: "", wantErr: false},
		{name: "WITH无增量无where尾部group by拒绝", dataSql: "with cte as (select a from t) select a from cte group by a", upd: "", wantErr: true},
		{name: "WITH增量拒绝", dataSql: "with cte as (select id, updated from t) select * from cte", upd: "updated", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateDataSyncSql(pg, newTask(c.dataSql, c.upd))
			if (err != nil) != c.wantErr {
				t.Fatalf("validateDataSyncSql(%q, upd=%q) error = %v, wantErr %v", c.dataSql, c.upd, err, c.wantErr)
			}
		})
	}
}

// TestPersistUpdFieldVal 批级水位持久化触发逻辑
func TestPersistUpdFieldVal(t *testing.T) {
	fake := &fakeDataSyncRepo{}
	app := &DataSyncAppImpl{
		AppImpl: base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]{Repo: fake},
	}

	// 无增量字段时不触发持久化
	app.persistUpdFieldVal(context.Background(), &entity.DataSyncTask{Id: 1, UpdField: "", UpdFieldVal: "100"})
	if len(fake.updated) != 0 {
		t.Fatalf("no updField should not persist watermark, got %d updates", len(fake.updated))
	}

	// 有增量字段时持久化当前水位
	task := &entity.DataSyncTask{Id: 2, UpdField: "id", UpdFieldVal: "200"}
	app.persistUpdFieldVal(context.Background(), task)
	if len(fake.updated) != 1 {
		t.Fatalf("updField set should persist watermark once, got %d updates", len(fake.updated))
	}
	if got := fake.updated[0]; got.Id != 2 || got.UpdFieldVal != "200" {
		t.Fatalf("persisted watermark = {id:%d, val:%s}, want {id:2, val:200}", got.Id, got.UpdFieldVal)
	}

	// 持久化失败仅记录日志不panic不中断
	fake.err = context.DeadlineExceeded
	app.persistUpdFieldVal(context.Background(), &entity.DataSyncTask{Id: 3, UpdField: "id", UpdFieldVal: "300"})
}
