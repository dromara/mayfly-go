package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
	"strconv"
	"strings"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"
)

type DbSqlExecReq struct {
	DbId   uint64
	Db     string
	Sql    string
	Remark string
	DbConn *dbi.DbConn
}

type DbSqlExecRes struct {
	Columns []*dbi.QueryColumn
	Res     []map[string]any
}

// 合并执行结果，主要用于执行多条sql使用
func (d *DbSqlExecRes) Merge(execRes *DbSqlExecRes) {
	canMerge := len(d.Columns) == len(execRes.Columns)
	if !canMerge {
		return
	}
	// 列名不一致，则不合并
	for i, col := range d.Columns {
		if execRes.Columns[i].Name != col.Name {
			return
		}
	}
	d.Res = append(d.Res, execRes.Res...)
}

type DbSqlExec interface {
	// 执行sql
	Exec(ctx context.Context, execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error)

	// 根据条件删除sql执行记录
	DeleteBy(ctx context.Context, condition *entity.DbSqlExec)

	// 分页获取
	GetPageList(condition *entity.DbSqlExecQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type dbSqlExecAppImpl struct {
	dbSqlExecRepo repository.DbSqlExec `inject:"DbSqlExecRepo"`
}

func createSqlExecRecord(ctx context.Context, execSqlReq *DbSqlExecReq) *entity.DbSqlExec {
	dbSqlExecRecord := new(entity.DbSqlExec)
	dbSqlExecRecord.DbId = execSqlReq.DbId
	dbSqlExecRecord.Db = execSqlReq.Db
	dbSqlExecRecord.Sql = execSqlReq.Sql
	dbSqlExecRecord.Remark = execSqlReq.Remark
	dbSqlExecRecord.FillBaseInfo(model.IdGenTypeNone, contextx.GetLoginAccount(ctx))
	return dbSqlExecRecord
}

func (d *dbSqlExecAppImpl) Exec(ctx context.Context, execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error) {
	sql := execSqlReq.Sql
	dbSqlExecRecord := createSqlExecRecord(ctx, execSqlReq)
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

				if !strings.Contains(lowerSql, "limit") &&
					// 兼容oracle rownum分页
					!strings.Contains(lowerSql, "rownum") &&
					// 兼容mssql offset分页
					!strings.Contains(lowerSql, "offset") &&
					// 兼容mssql top 分页  with result as ({query sql}) select top 100 * from result
					!strings.Contains(lowerSql, " top ") {
					// 判断是不是count语句
					if !strings.Contains(lowerSql, "count(") {
						return nil, errorx.NewBiz("请完善分页信息后执行")
					}
				}
			}
		}
		var execErr error
		if isSelect || strings.HasPrefix(lowerSql, "show") {
			execRes, execErr = doRead(ctx, execSqlReq)
		} else {
			execRes, execErr = doExec(ctx, execSqlReq.Sql, execSqlReq.DbConn)
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
		execRes, err = doSelect(ctx, stmt, execSqlReq)
	case *sqlparser.Show:
		isSelect = true
		execRes, err = doRead(ctx, execSqlReq)
	case *sqlparser.OtherRead:
		isSelect = true
		execRes, err = doRead(ctx, execSqlReq)
	case *sqlparser.Update:
		execRes, err = doUpdate(ctx, stmt, execSqlReq, dbSqlExecRecord)
	case *sqlparser.Delete:
		execRes, err = doDelete(ctx, stmt, execSqlReq, dbSqlExecRecord)
	case *sqlparser.Insert:
		execRes, err = doInsert(ctx, stmt, execSqlReq, dbSqlExecRecord)
	default:
		execRes, err = doExec(ctx, execSqlReq.Sql, execSqlReq.DbConn)
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
		d.dbSqlExecRepo.Insert(context.TODO(), dbSqlExecRecord)
		return
	}
	if config.GetDbSaveQuerySql() {
		dbSqlExecRecord.Table = "-"
		dbSqlExecRecord.OldValue = "-"
		dbSqlExecRecord.Type = entity.DbSqlExecTypeQuery
		d.dbSqlExecRepo.Insert(context.TODO(), dbSqlExecRecord)
	}
}

func (d *dbSqlExecAppImpl) DeleteBy(ctx context.Context, condition *entity.DbSqlExec) {
	d.dbSqlExecRepo.DeleteByCond(ctx, condition)
}

func (d *dbSqlExecAppImpl) GetPageList(condition *entity.DbSqlExecQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.dbSqlExecRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func doSelect(ctx context.Context, selectStmt *sqlparser.Select, execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error) {
	selectExprsStr := sqlparser.String(selectStmt.SelectExprs)
	if selectExprsStr == "*" || strings.Contains(selectExprsStr, ".*") ||
		len(strings.Split(selectExprsStr, ",")) > 1 {
		// 如果配置为0，则不校验分页参数
		maxCount := config.GetDbQueryMaxCount()
		// 哪些数据库跳过校验
		skipped := dbi.DbTypeOracle == execSqlReq.DbConn.Info.Type || dbi.DbTypeMssql == execSqlReq.DbConn.Info.Type
		if maxCount != 0 && !skipped {
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

	return doRead(ctx, execSqlReq)
}

func doRead(ctx context.Context, execSqlReq *DbSqlExecReq) (*DbSqlExecRes, error) {
	dbConn := execSqlReq.DbConn
	sql := execSqlReq.Sql
	cols, res, err := dbConn.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	return &DbSqlExecRes{
		Columns: cols,
		Res:     res,
	}, nil
}

func doUpdate(ctx context.Context, update *sqlparser.Update, execSqlReq *DbSqlExecReq, dbSqlExec *entity.DbSqlExec) (*DbSqlExecRes, error) {
	dbConn := execSqlReq.DbConn

	tableStr := sqlparser.String(update.TableExprs)
	// 可能使用别名，故空格切割
	tableName := strings.Split(tableStr, " ")[0]
	if strings.Contains(tableName, ".") {
		tableName = strings.Split(tableName, ".")[1]
	}
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
	primaryKey, err := dbConn.GetDialect().GetPrimaryKey(tableName)
	if err != nil {
		return nil, errorx.NewBiz("获取表主键信息失败")
	}

	updateColumnsAndPrimaryKey := strings.Join(updateColumns, ",") + "," + primaryKey
	// 查询要更新字段数据的旧值，以及主键值
	selectSql := fmt.Sprintf("SELECT %s FROM %s %s", updateColumnsAndPrimaryKey, tableStr, where)

	// WalkQuery查出最多200条数据
	maxRec := 200
	nowRec := 0
	res := make([]map[string]any, 0)
	err = dbConn.WalkQueryRows(ctx, selectSql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		nowRec++
		res = append(res, row)
		if nowRec == maxRec {
			return errorx.NewBiz(fmt.Sprintf("超出更新最大查询条数限制: %d", maxRec))
		}
		return nil
	})

	if err != nil {
		logx.Warn(err.Error())
	}

	dbSqlExec.OldValue = jsonx.ToStr(res)
	dbSqlExec.Table = tableName
	dbSqlExec.Type = entity.DbSqlExecTypeUpdate

	return doExec(ctx, execSqlReq.Sql, dbConn)
}

func doDelete(ctx context.Context, delete *sqlparser.Delete, execSqlReq *DbSqlExecReq, dbSqlExec *entity.DbSqlExec) (*DbSqlExecRes, error) {
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
	_, res, _ := dbConn.QueryContext(ctx, selectSql)

	dbSqlExec.OldValue = jsonx.ToStr(res)
	dbSqlExec.Table = table
	dbSqlExec.Type = entity.DbSqlExecTypeDelete

	return doExec(ctx, execSqlReq.Sql, dbConn)
}

func doInsert(ctx context.Context, insert *sqlparser.Insert, execSqlReq *DbSqlExecReq, dbSqlExec *entity.DbSqlExec) (*DbSqlExecRes, error) {
	tableStr := sqlparser.String(insert.Table)
	// 可能使用别名，故空格切割
	table := strings.Split(tableStr, " ")[0]
	dbSqlExec.Table = table
	dbSqlExec.Type = entity.DbSqlExecTypeInsert

	return doExec(ctx, execSqlReq.Sql, execSqlReq.DbConn)
}

func doExec(ctx context.Context, sql string, dbConn *dbi.DbConn) (*DbSqlExecRes, error) {
	rowsAffected, err := dbConn.ExecContext(ctx, sql)
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
		Columns: []*dbi.QueryColumn{
			{Name: "sql", Type: "string"},
			{Name: "rowsAffected", Type: "number"},
			{Name: "result", Type: "string"},
		},
		Res: res,
	}, err
}
