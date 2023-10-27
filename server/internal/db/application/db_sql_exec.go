package application

import (
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"strconv"
	"strings"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"
)

type DbSqlExecReq struct {
	DbId         uint64
	Db           string
	Sql          string
	Remark       string
	LoginAccount *model.LoginAccount
	DbConn       *dbm.DbConn
}

type DbSqlExecRes struct {
	ColNames []string
	Res      []map[string]any
}

// 合并执行结果，主要用于执行多条sql使用
func (d *DbSqlExecRes) Merge(execRes *DbSqlExecRes) {
	canMerge := len(d.ColNames) == len(execRes.ColNames)
	if !canMerge {
		return
	}
	// 列名不一致，则不合并
	for i, colName := range d.ColNames {
		if execRes.ColNames[i] != colName {
			return
		}
	}
	d.Res = append(d.Res, execRes.Res...)
}

type DbSqlExec interface {
	// 执行sql
	Exec(execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error)

	// 根据条件删除sql执行记录
	DeleteBy(condition *entity.DbSqlExec)

	// 分页获取
	GetPageList(condition *entity.DbSqlExecQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

func newDbSqlExecApp(dbExecSqlRepo repository.DbSqlExec) DbSqlExec {
	return &dbSqlExecAppImpl{
		dbSqlExecRepo: dbExecSqlRepo,
	}
}

type dbSqlExecAppImpl struct {
	dbSqlExecRepo repository.DbSqlExec
}

func createSqlExecRecord(execSqlReq *DbSqlExecReq) *entity.DbSqlExec {
	dbSqlExecRecord := new(entity.DbSqlExec)
	dbSqlExecRecord.DbId = execSqlReq.DbId
	dbSqlExecRecord.Db = execSqlReq.Db
	dbSqlExecRecord.Sql = execSqlReq.Sql
	dbSqlExecRecord.Remark = execSqlReq.Remark
	dbSqlExecRecord.SetBaseInfo(execSqlReq.LoginAccount)
	return dbSqlExecRecord
}

func (d *dbSqlExecAppImpl) Exec(execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error) {
	sql := execSqlReq.Sql
	dbSqlExecRecord := createSqlExecRecord(execSqlReq)
	dbSqlExecRecord.Type = entity.DbSqlExecTypeOther
	var execRes *DbSqlExecRes
	isSelect := false

	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		// 就算解析失败也执行sql，让数据库来判断错误。如果是查询sql则简单判断是否有limit分页参数信息（兼容pgsql）
		// logx.Warnf("sqlparse解析sql[%s]失败: %s", sql, err.Error())
		lowerSql := strings.ToLower(execSqlReq.Sql)
		isSelect := strings.HasPrefix(lowerSql, "select")
		if isSelect {
			// 如果配置为0，则不校验分页参数
			maxCount := config.GetDbQueryMaxCount()
			if maxCount != 0 {
				if !strings.Contains(lowerSql, "limit") {
					return nil, errorx.NewBiz("请完善分页信息后执行")
				}
			}
		}
		var execErr error
		if isSelect || strings.HasPrefix(lowerSql, "show") {
			execRes, execErr = doRead(execSqlReq)
		} else {
			execRes, execErr = doExec(execSqlReq.Sql, execSqlReq.DbConn)
		}
		if execErr != nil {
			return nil, execErr
		}
		d.saveSqlExecLog(isSelect, dbSqlExecRecord)
		return execRes, nil
	}

	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		isSelect = true
		execRes, err = doSelect(stmt, execSqlReq)
	case *sqlparser.Show:
		isSelect = true
		execRes, err = doRead(execSqlReq)
	case *sqlparser.OtherRead:
		isSelect = true
		execRes, err = doRead(execSqlReq)
	case *sqlparser.Update:
		execRes, err = doUpdate(stmt, execSqlReq, dbSqlExecRecord)
	case *sqlparser.Delete:
		execRes, err = doDelete(stmt, execSqlReq, dbSqlExecRecord)
	case *sqlparser.Insert:
		execRes, err = doInsert(stmt, execSqlReq, dbSqlExecRecord)
	default:
		execRes, err = doExec(execSqlReq.Sql, execSqlReq.DbConn)
	}
	if err != nil {
		return nil, err
	}
	d.saveSqlExecLog(isSelect, dbSqlExecRecord)
	return execRes, nil
}

// 保存sql执行记录，如果是查询类则根据系统配置判断是否保存
func (d *dbSqlExecAppImpl) saveSqlExecLog(isQuery bool, dbSqlExecRecord *entity.DbSqlExec) {
	if !isQuery {
		d.dbSqlExecRepo.Insert(dbSqlExecRecord)
		return
	}
	if config.GetDbSaveQuerySql() {
		dbSqlExecRecord.Table = "-"
		dbSqlExecRecord.OldValue = "-"
		dbSqlExecRecord.Type = entity.DbSqlExecTypeQuery
		d.dbSqlExecRepo.Insert(dbSqlExecRecord)
	}
}

func (d *dbSqlExecAppImpl) DeleteBy(condition *entity.DbSqlExec) {
	d.dbSqlExecRepo.DeleteByCond(condition)
}

func (d *dbSqlExecAppImpl) GetPageList(condition *entity.DbSqlExecQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.dbSqlExecRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func doSelect(selectStmt *sqlparser.Select, execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error) {
	selectExprsStr := sqlparser.String(selectStmt.SelectExprs)
	if selectExprsStr == "*" || strings.Contains(selectExprsStr, ".*") ||
		len(strings.Split(selectExprsStr, ",")) > 1 {
		// 如果配置为0，则不校验分页参数
		maxCount := config.GetDbQueryMaxCount()
		if maxCount != 0 {
			limit := selectStmt.Limit
			if limit == nil {
				return nil, errorx.NewBiz("请完善分页信息后执行")
			}

			count, err := strconv.Atoi(sqlparser.String(limit.Rowcount))
			if err != nil {
				return nil, errorx.NewBiz("分页参数有误")
			}

			if count > maxCount {
				return nil, errorx.NewBiz("查询结果集数需小于系统配置的%d条", maxCount)
			}
		}
	}

	return doRead(execSqlReq)
}

func doRead(execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error) {
	dbConn := execSqlReq.DbConn
	sql := execSqlReq.Sql
	colNames, res, err := dbConn.SelectData(sql)
	if err != nil {
		return nil, err
	}
	return &DbSqlExecRes{
		ColNames: colNames,
		Res:      res,
	}, nil
}

func doUpdate(update *sqlparser.Update, execSqlReq *DbSqlExecReq, dbSqlExec *entity.DbSqlExec) (*DbSqlExecRes, error) {
	dbConn := execSqlReq.DbConn

	tableStr := sqlparser.String(update.TableExprs)
	// 可能使用别名，故空格切割
	tableName := strings.Split(tableStr, " ")[0]
	where := sqlparser.String(update.Where)
	if len(where) == 0 {
		return nil, errorx.NewBiz("SQL[%s]未执行. 请完善 where 条件后再执行", execSqlReq.Sql)
	}

	updateExprs := update.Exprs
	updateColumns := make([]string, 0)
	for _, v := range updateExprs {
		updateColumns = append(updateColumns, v.Name.Name.String())
	}

	// 获取表主键列名,排除使用别名
	primaryKey, err := dbConn.GetMeta().GetPrimaryKey(tableName)
	if err != nil {
		return nil, errorx.NewBiz("获取表主键信息失败")
	}

	updateColumnsAndPrimaryKey := strings.Join(updateColumns, ",") + "," + primaryKey
	// 查询要更新字段数据的旧值，以及主键值
	selectSql := fmt.Sprintf("SELECT %s FROM %s %s LIMIT 200", updateColumnsAndPrimaryKey, tableStr, where)
	_, res, err := dbConn.SelectData(selectSql)
	if err == nil {
		bytes, _ := json.Marshal(res)
		dbSqlExec.OldValue = string(bytes)
	} else {
		dbSqlExec.OldValue = err.Error()
	}

	dbSqlExec.Table = tableName
	dbSqlExec.Type = entity.DbSqlExecTypeUpdate

	return doExec(execSqlReq.Sql, dbConn)
}

func doDelete(delete *sqlparser.Delete, execSqlReq *DbSqlExecReq, dbSqlExec *entity.DbSqlExec) (*DbSqlExecRes, error) {
	dbConn := execSqlReq.DbConn

	tableStr := sqlparser.String(delete.TableExprs)
	// 可能使用别名，故空格切割
	table := strings.Split(tableStr, " ")[0]
	where := sqlparser.String(delete.Where)
	if len(where) == 0 {
		return nil, errorx.NewBiz("SQL[%s]未执行. 请完善 where 条件后再执行", execSqlReq.Sql)
	}

	// 查询删除数据
	selectSql := fmt.Sprintf("SELECT * FROM %s %s LIMIT 200", tableStr, where)
	_, res, _ := dbConn.SelectData(selectSql)

	bytes, _ := json.Marshal(res)
	dbSqlExec.OldValue = string(bytes)
	dbSqlExec.Table = table
	dbSqlExec.Type = entity.DbSqlExecTypeDelete

	return doExec(execSqlReq.Sql, dbConn)
}

func doInsert(insert *sqlparser.Insert, execSqlReq *DbSqlExecReq, dbSqlExec *entity.DbSqlExec) (*DbSqlExecRes, error) {
	tableStr := sqlparser.String(insert.Table)
	// 可能使用别名，故空格切割
	table := strings.Split(tableStr, " ")[0]
	dbSqlExec.Table = table
	dbSqlExec.Type = entity.DbSqlExecTypeInsert

	return doExec(execSqlReq.Sql, execSqlReq.DbConn)
}

func doExec(sql string, dbConn *dbm.DbConn) (*DbSqlExecRes, error) {
	rowsAffected, err := dbConn.Exec(sql)
	execRes := "success"
	if err != nil {
		execRes = err.Error()
	}
	res := make([]map[string]any, 0)
	resData := make(map[string]any)
	resData["rowsAffected"] = rowsAffected
	resData["sql"] = sql
	resData["result"] = execRes
	res = append(res, resData)

	return &DbSqlExecRes{
		ColNames: []string{"sql", "rowsAffected", "result"},
		Res:      res,
	}, err
}
