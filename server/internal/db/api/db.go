package api

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
	"mayfly-go/pkg/ws"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xwb1989/sqlparser"
)

type Db struct {
	DbApp        application.Db
	DbSqlExecApp application.DbSqlExec
	MsgApp       msgapp.Msg
	TagApp       tagapp.TagTree
}

const DEFAULT_ROW_SIZE = 5000

// @router /api/dbs [get]
func (d *Db) Dbs(rc *req.Ctx) {
	condition := new(entity.DbQuery)
	condition.TagPathLike = rc.GinCtx.Query("tagPath")

	// 不存在可访问标签id，即没有可操作数据
	tagIds := d.TagApp.ListTagIdByAccountId(rc.LoginAccount.Id)
	if len(tagIds) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	condition.TagIds = tagIds
	rc.ResData = d.DbApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]vo.SelectDataDbVO))
}

func (d *Db) Save(rc *req.Ctx) {
	form := &form.DbForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	db := new(entity.Db)
	utils.Copy(db, form)

	// 密码解密，并使用解密后的赋值
	originPwd, err := utils.DefaultRsaDecrypt(form.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	db.Password = originPwd

	// 密码脱敏记录日志
	form.Password = "****"
	rc.ReqParam = form

	db.SetBaseInfo(rc.LoginAccount)
	d.DbApp.Save(db)
}

// 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
func (d *Db) GetDbPwd(rc *req.Ctx) {
	dbId := GetDbId(rc.GinCtx)
	dbEntity := d.DbApp.GetById(dbId, "Password")
	dbEntity.PwdDecrypt()
	rc.ResData = dbEntity.Password
}

// 获取数据库实例的所有数据库名
func (d *Db) GetDatabaseNames(rc *req.Ctx) {
	form := &form.DbForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	db := new(entity.Db)
	utils.Copy(db, form)

	// 密码解密，并使用解密后的赋值
	originPwd, err := utils.DefaultRsaDecrypt(form.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	db.Password = originPwd

	// 如果id不为空，并且密码为空则从数据库查询
	if form.Id != 0 && db.Password == "" {
		db = d.DbApp.GetById(form.Id)
	}
	rc.ResData = d.DbApp.GetDatabases(db)
}

func (d *Db) DeleteDb(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "dbId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		dbId := uint64(value)
		d.DbApp.Delete(dbId)
		// 删除该库的sql执行记录
		d.DbSqlExecApp.DeleteBy(&entity.DbSqlExec{DbId: dbId})
	}
}

func (d *Db) TableInfos(rc *req.Ctx) {
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta().GetTableInfos()
}

func (d *Db) TableIndex(rc *req.Ctx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta().GetTableIndex(tn)
}

func (d *Db) GetCreateTableDdl(rc *req.Ctx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta().GetCreateTableDdl(tn)
}

func (d *Db) ExecSql(rc *req.Ctx) {
	g := rc.GinCtx
	form := &form.DbSqlExecForm{}
	ginx.BindJsonAndValid(g, form)

	id := GetDbId(g)
	db := form.Db
	dbInstance := d.DbApp.GetDbInstance(id, db)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbInstance.Info.TagPath), "%s")

	rc.ReqParam = fmt.Sprintf("%s\n-> %s", dbInstance.Info.GetLogDesc(), form.Sql)
	biz.NotEmpty(form.Sql, "sql不能为空")

	// 去除前后空格及换行符
	sql := utils.StrTrimSpaceAndBr(form.Sql)

	execReq := &application.DbSqlExecReq{
		DbId:         id,
		Db:           db,
		Remark:       form.Remark,
		DbInstance:   dbInstance,
		LoginAccount: rc.LoginAccount,
	}

	sqls, err := sqlparser.SplitStatementToPieces(sql)
	biz.ErrIsNil(err, "SQL解析错误,请检查您的执行SQL")
	isMulti := len(sqls) > 1
	var execResAll *application.DbSqlExecRes
	for _, s := range sqls {
		s = utils.StrTrimSpaceAndBr(s)
		// 多条执行，如果有查询语句，则跳过
		if isMulti && strings.HasPrefix(strings.ToLower(s), "select") {
			continue
		}
		execReq.Sql = s
		execRes, err := d.DbSqlExecApp.Exec(execReq)
		if err != nil {
			biz.ErrIsNilAppendErr(err, fmt.Sprintf("[%s] -> 执行失败: ", s)+"%s")
		}

		if execResAll == nil {
			execResAll = execRes
		} else {
			execResAll.Merge(execRes)
		}
	}

	colAndRes := make(map[string]any)
	colAndRes["colNames"] = execResAll.ColNames
	colAndRes["res"] = execResAll.Res
	rc.ResData = colAndRes
}

// 执行sql文件
func (d *Db) ExecSqlFile(rc *req.Ctx) {
	g := rc.GinCtx
	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")

	file, _ := fileheader.Open()
	filename := fileheader.Filename
	dbId, db := GetIdAndDb(g)

	dbInstance := d.DbApp.GetDbInstance(dbId, db)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbInstance.Info.TagPath), "%s")
	rc.ReqParam = fmt.Sprintf("%s -> filename: %s", dbInstance.Info.GetLogDesc(), filename)

	logExecRecord := true
	// 如果执行sql文件大于该值则不记录sql执行记录
	if fileheader.Size > 50*1024 {
		logExecRecord = false
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				switch t := err.(type) {
				case *biz.BizError:
					d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrMsg("sql脚本执行失败", fmt.Sprintf("[%s]%s执行失败: [%s]", filename, dbInstance.Info.GetLogDesc(), t.Error())))
				}
			}
		}()

		execReq := &application.DbSqlExecReq{
			DbId:         dbId,
			Db:           db,
			Remark:       fileheader.Filename,
			DbInstance:   dbInstance,
			LoginAccount: rc.LoginAccount,
		}

		tokens := sqlparser.NewTokenizer(file)
		for {
			stmt, err := sqlparser.ParseNext(tokens)
			if err == io.EOF {
				break
			}
			sql := sqlparser.String(stmt)
			execReq.Sql = sql
			// 需要记录执行记录
			if logExecRecord {
				_, err = d.DbSqlExecApp.Exec(execReq)
			} else {
				_, err = dbInstance.Exec(sql)
			}

			if err != nil {
				d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrMsg("sql脚本执行失败", fmt.Sprintf("[%s][%s] -> sql=[%s] 执行失败: [%s]", filename, dbInstance.Info.GetLogDesc(), sql, err.Error())))
				return
			}
		}
		d.MsgApp.CreateAndSend(rc.LoginAccount, ws.SuccessMsg("sql脚本执行成功", fmt.Sprintf("[%s]执行完成 -> %s", filename, dbInstance.Info.GetLogDesc())))
	}()
}

// 数据库dump
func (d *Db) DumpSql(rc *req.Ctx) {
	g := rc.GinCtx
	dbId, db := GetIdAndDb(g)
	dumpType := g.Query("type")
	tablesStr := g.Query("tables")
	biz.NotEmpty(tablesStr, "请选择要导出的表")
	tables := strings.Split(tablesStr, ",")

	// 是否需要导出表结构
	needStruct := dumpType == "1" || dumpType == "3"
	// 是否需要导出数据
	needData := dumpType == "2" || dumpType == "3"

	dbInstance := d.DbApp.GetDbInstance(dbId, db)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbInstance.Info.TagPath), "%s")

	now := time.Now()
	filename := fmt.Sprintf("%s.%s.sql", db, now.Format("200601021504"))
	g.Header("Content-Type", "application/octet-stream")
	g.Header("Content-Disposition", "attachment; filename="+filename)

	writer := g.Writer
	writer.WriteString("-- ----------------------------")
	writer.WriteString("\n-- 导出平台: mayfly-go")
	writer.WriteString(fmt.Sprintf("\n-- 导出时间: %s ", now.Format("2006-01-02 15:04:05")))
	writer.WriteString(fmt.Sprintf("\n-- 导出数据库: %s ", db))
	writer.WriteString("\n-- ----------------------------\n")

	dbmeta := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta()
	for _, table := range tables {
		if needStruct {
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表结构: %s \n-- ----------------------------\n", table))
			writer.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n", table))
			writer.WriteString(dbmeta.GetCreateTableDdl(table) + ";\n")
		}

		if !needData {
			continue
		}

		writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表记录: %s \n-- ----------------------------\n", table))
		writer.WriteString("BEGIN;\n")

		pageNum := 1
		for {
			columns, result, _ := dbmeta.GetTableRecord(table, pageNum, DEFAULT_ROW_SIZE)
			resultLen := len(result)
			if resultLen == 0 {
				break
			}
			insertSql := "INSERT INTO `%s` VALUES (%s);\n"
			for _, res := range result {
				var values []string
				for _, column := range columns {
					value := res[column]
					if value == nil {
						values = append(values, "NULL")
						continue
					}
					strValue, ok := value.(string)
					if ok {
						values = append(values, fmt.Sprintf("%#v", strValue))
					} else {
						values = append(values, utils.ToString(value))
					}
				}
				writer.WriteString(fmt.Sprintf(insertSql, table, strings.Join(values, ", ")))
			}
			if resultLen < DEFAULT_ROW_SIZE {
				break
			}
			pageNum++
		}

		writer.WriteString("COMMIT;\n")
	}
	rc.NoRes = true

	rc.ReqParam = fmt.Sprintf("%s, tables: %s, dumpType: %s", dbInstance.Info.GetLogDesc(), tablesStr, dumpType)
}

// @router /api/db/:dbId/t-metadata [get]
func (d *Db) TableMA(rc *req.Ctx) {
	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	rc.ResData = dbi.GetMeta().GetTables()
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *req.Ctx) {
	g := rc.GinCtx
	tn := g.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")

	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	rc.ResData = dbi.GetMeta().GetColumns(tn)
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *req.Ctx) {
	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))

	dm := dbi.GetMeta()
	// 获取所有表
	tables := dm.GetTables()
	tableNames := make([]string, 0)
	for _, v := range tables {
		tableNames = append(tableNames, v.TableName)
	}
	// key = 表名，value = 列名数组
	res := make(map[string][]string)

	// 表为空，则直接返回
	if len(tableNames) == 0 {
		rc.ResData = res
		return
	}

	// 获取所有表下的所有列信息
	columnMds := dm.GetColumns(tableNames...)
	for _, v := range columnMds {
		tName := v.TableName
		if res[tName] == nil {
			res[tName] = make([]string, 0)
		}

		columnName := fmt.Sprintf("%s  [%s]", v.ColumnName, v.ColumnType)
		comment := v.ColumnComment
		// 如果字段备注不为空，则加上备注信息
		if comment != "" {
			columnName = fmt.Sprintf("%s[%s]", columnName, comment)
		}

		res[tName] = append(res[tName], columnName)
	}
	rc.ResData = res
}

// @router /api/db/:dbId/sql [post]
func (d *Db) SaveSql(rc *req.Ctx) {
	g := rc.GinCtx
	account := rc.LoginAccount
	dbSqlForm := &form.DbSqlSaveForm{}
	ginx.BindJsonAndValid(g, dbSqlForm)
	rc.ReqParam = dbSqlForm

	dbId := GetDbId(g)
	// 判断dbId是否存在
	err := gormx.GetById(new(entity.Db), dbId)
	biz.ErrIsNil(err, "该数据库信息不存在")

	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: dbSqlForm.Type, DbId: dbId, Name: dbSqlForm.Name, Db: dbSqlForm.Db}
	dbSql.CreatorId = account.Id
	e := gormx.GetBy(dbSql)

	dbSql.SetBaseInfo(account)
	// 更新sql信息
	dbSql.Sql = dbSqlForm.Sql
	if e == nil {
		gormx.UpdateById(dbSql)
	} else {
		gormx.Insert(dbSql)
	}
}

// 获取所有保存的sql names
func (d *Db) GetSqlNames(rc *req.Ctx) {
	id, db := GetIdAndDb(rc.GinCtx)
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: 1, DbId: id, Db: db}
	dbSql.CreatorId = rc.LoginAccount.Id
	var sqls []entity.DbSql
	gormx.ListBy(dbSql, &sqls, "id", "name")

	rc.ResData = sqls
}

// 删除保存的sql
func (d *Db) DeleteSql(rc *req.Ctx) {
	dbSql := &entity.DbSql{Type: 1, DbId: GetDbId(rc.GinCtx)}
	dbSql.CreatorId = rc.LoginAccount.Id
	dbSql.Name = rc.GinCtx.Query("name")
	dbSql.Db = rc.GinCtx.Query("db")

	gormx.DeleteByCondition(dbSql)

}

// @router /api/db/:dbId/sql [get]
func (d *Db) GetSql(rc *req.Ctx) {
	id, db := GetIdAndDb(rc.GinCtx)
	// 根据创建者id， 数据库id，以及sql模板名称查询保存的sql信息
	dbSql := &entity.DbSql{Type: 1, DbId: id, Db: db}
	dbSql.CreatorId = rc.LoginAccount.Id
	dbSql.Name = rc.GinCtx.Query("name")

	e := gormx.GetBy(dbSql)
	if e != nil {
		return
	}
	rc.ResData = dbSql
}

func GetDbId(g *gin.Context) uint64 {
	dbId, _ := strconv.Atoi(g.Param("dbId"))
	biz.IsTrue(dbId > 0, "dbId错误")
	return uint64(dbId)
}

func GetIdAndDb(g *gin.Context) (uint64, string) {
	db := g.Query("db")
	biz.NotEmpty(db, "db不能为空")
	return GetDbId(g), db
}
