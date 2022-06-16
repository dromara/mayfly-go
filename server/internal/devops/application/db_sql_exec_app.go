package application

import (
	"encoding/json"
	"fmt"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/internal/devops/infrastructure/persistence"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"strings"

	"github.com/xwb1989/sqlparser"
)

type DbSqlExec interface {
	// 生成sql执行记录
	GenExecLog(loginAccount *model.LoginAccount, dbId uint64, db string, sql string, dbInstance *DbInstance) *entity.DbSqlExec

	// 保存sql执行记录
	Save(*entity.DbSqlExec)

	// 根据条件删除sql执行记录
	DeleteBy(condition *entity.DbSqlExec)

	// 分页获取
	GetPageList(condition *entity.DbSqlExec, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult
}

type dbSqlExecAppImpl struct {
	dbSqlExecRepo repository.DbSqlExec
}

var DbSqlExecApp DbSqlExec = &dbSqlExecAppImpl{
	dbSqlExecRepo: persistence.DbSqlExecDao,
}

func (d *dbSqlExecAppImpl) GenExecLog(loginAccount *model.LoginAccount, dbId uint64, db string, sql string, dbInstance *DbInstance) *entity.DbSqlExec {
	defer func() {
		if err := recover(); err != nil {
			global.Log.Error("生成sql执行记录失败", err)
		}
	}()
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		global.Log.Error("记录数据库sql执行记录失败", err)
	}

	dbSqlExecRecord := new(entity.DbSqlExec)
	dbSqlExecRecord.DbId = dbId
	dbSqlExecRecord.Db = db
	dbSqlExecRecord.Sql = sql
	dbSqlExecRecord.SetBaseInfo(loginAccount)

	switch stmt := stmt.(type) {
	case *sqlparser.Update:
		doUpdate(stmt, dbInstance, dbSqlExecRecord)
	case *sqlparser.Delete:
		doDelete(stmt, dbInstance, dbSqlExecRecord)
	case *sqlparser.Insert:
		doInsert(stmt, dbSqlExecRecord)
	}

	return dbSqlExecRecord
}

func (d *dbSqlExecAppImpl) Save(dse *entity.DbSqlExec) {
	d.dbSqlExecRepo.Insert(dse)
}

func (d *dbSqlExecAppImpl) DeleteBy(condition *entity.DbSqlExec) {
	d.dbSqlExecRepo.DeleteBy(condition)
}

func (d *dbSqlExecAppImpl) GetPageList(condition *entity.DbSqlExec, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return d.dbSqlExecRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func doUpdate(update *sqlparser.Update, dbInstance *DbInstance, dbSqlExec *entity.DbSqlExec) {
	tableStr := sqlparser.String(update.TableExprs)
	// 可能使用别名，故空格切割
	tableName := strings.Split(tableStr, " ")[0]
	where := sqlparser.String(update.Where)

	updateExprs := update.Exprs
	updateColumns := make([]string, 0)
	for _, v := range updateExprs {
		updateColumns = append(updateColumns, v.Name.Name.String())
	}

	// 获取表主键列名,排除使用别名
	primaryKey := dbInstance.GetPrimaryKey(tableName)

	updateColumnsAndPrimaryKey := strings.Join(updateColumns, ",") + "," + primaryKey
	// 查询要更新字段数据的旧值，以及主键值
	selectSql := fmt.Sprintf("SELECT %s FROM %s %s LIMIT 200", updateColumnsAndPrimaryKey, tableStr, where)
	_, res, _ := dbInstance.SelectData(selectSql)

	bytes, _ := json.Marshal(res)
	dbSqlExec.OldValue = string(bytes)
	dbSqlExec.Table = tableName
	dbSqlExec.Type = entity.DbSqlExecTypeUpdate
}

func doDelete(delete *sqlparser.Delete, dbInstance *DbInstance, dbSqlExec *entity.DbSqlExec) {
	tableStr := sqlparser.String(delete.TableExprs)
	// 可能使用别名，故空格切割
	table := strings.Split(tableStr, " ")[0]
	where := sqlparser.String(delete.Where)

	// 查询删除数据
	selectSql := fmt.Sprintf("SELECT * FROM %s %s LIMIT 200", tableStr, where)
	_, res, _ := dbInstance.SelectData(selectSql)

	bytes, _ := json.Marshal(res)
	dbSqlExec.OldValue = string(bytes)
	dbSqlExec.Table = table
	dbSqlExec.Type = entity.DbSqlExecTypeDelete
}

func doInsert(insert *sqlparser.Insert, dbSqlExec *entity.DbSqlExec) {
	tableStr := sqlparser.String(insert.Table)
	// 可能使用别名，故空格切割
	table := strings.Split(tableStr, " ")[0]
	dbSqlExec.Table = table
	dbSqlExec.Type = entity.DbSqlExecTypeInsert
}
