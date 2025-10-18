package application

import (
	"cmp"
	"context"
	"fmt"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/writerx"
	"sort"
	"strings"
	"time"
)

type Db interface {
	base.App[*entity.Db]

	// 分页获取
	GetPageList(condition *entity.DbQuery, orderBy ...string) (*model.PageResult[*entity.DbListPO], error)

	SaveDb(ctx context.Context, entity *entity.Db) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取数据库连接实例
	// @param id 数据库id
	//
	// @param dbName 数据库名
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)

	// 根据数据库实例id获取连接，随机返回该instanceId下已连接的conn，若不存在则是使用该instanceId关联的db进行连接并返回。
	GetDbConnByInstanceId(ctx context.Context, instanceId uint64) (*dbi.DbConn, error)

	// DumpDb dumpDb
	DumpDb(ctx context.Context, reqParam *dto.DumpDb) error
}

type dbAppImpl struct {
	base.AppImpl[*entity.Db, repository.Db]

	dbSqlRepo           repository.DbSql        `inject:"T"`
	dbInstanceApp       Instance                `inject:"T"`
	dbSqlExecApp        DbSqlExec               `inject:"T"`
	tagApp              tagapp.TagTree          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

var _ (Db) = (*dbAppImpl)(nil)

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.DbQuery, orderBy ...string) (*model.PageResult[*entity.DbListPO], error) {
	return d.GetRepo().GetDbList(condition, orderBy...)
}

func (d *dbAppImpl) SaveDb(ctx context.Context, dbEntity *entity.Db) error {
	// 查找是否存在
	oldDb := &entity.Db{Name: dbEntity.Name, InstanceId: dbEntity.InstanceId, AuthCertName: dbEntity.AuthCertName}

	authCert, err := d.resourceAuthCertApp.GetAuthCert(dbEntity.AuthCertName)
	if err != nil {
		return errorx.NewBiz("ac not found")
	}

	err = d.GetByCond(oldDb)
	if dbEntity.Id == 0 {
		if err == nil {
			return errorx.NewBizI(ctx, imsg.ErrDbNameExist)
		}
		dbEntity.Code = stringx.Rand(10)

		return d.Tx(ctx, func(ctx context.Context) error {
			return d.Insert(ctx, dbEntity)
		}, func(ctx context.Context) error {
			// 将库关联至指定数据库授权凭证下
			return d.tagApp.RelateTagsByCodeAndType(ctx, &tagdto.RelateTagsByCodeAndType{
				Tags: []*tagdto.ResourceTag{{
					Code: dbEntity.Code,
					Type: tagentity.TagTypeDb,
					Name: dbEntity.Name,
				}},
				ParentTagCode: authCert.Name,
				ParentTagType: tagentity.TagTypeAuthCert,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldDb.Id != dbEntity.Id {
		return errorx.NewBizI(ctx, imsg.ErrDbNameExist)
	}

	dbId := dbEntity.Id
	old, err := d.GetById(dbId)
	if err != nil {
		return errorx.NewBiz("db not found")
	}

	oldDbs := strings.Split(old.Database, " ")
	newDbs := strings.Split(dbEntity.Database, " ")
	// 比较新旧数据库列表，需要将移除的数据库相关联的信息删除
	_, delDb, _ := collx.ArrayCompare(newDbs, oldDbs)

	// 先简单关闭可能存在的旧库连接（可能改了关联标签导致DbConn.Info.TagPath与修改后的标签不一致、导致操作权限校验出错）
	for _, v := range oldDbs {
		// 关闭数据库连接
		dbm.CloseDb(dbEntity.Id, v)
	}

	for _, v := range delDb {
		// 删除该库关联的所有sql记录
		d.dbSqlRepo.DeleteByCond(ctx, &entity.DbSql{DbId: dbId, Db: v})
	}

	// 防止误传修改
	dbEntity.Code = ""
	return d.Tx(ctx, func(ctx context.Context) error {
		return d.UpdateById(ctx, dbEntity)
	}, func(ctx context.Context) error {
		if old.Name != dbEntity.Name {
			if err := d.tagApp.UpdateTagName(ctx, tagentity.TagTypeDb, old.Code, dbEntity.Name); err != nil {
				return err
			}
		}
		if authCert.Name != old.AuthCertName {
			return d.tagApp.ChangeParentTag(ctx, tagentity.TagTypeDb, old.Code, tagentity.TagTypeAuthCert, authCert.Name)
		}
		return nil
	})
}

func (d *dbAppImpl) Delete(ctx context.Context, id uint64) error {
	db, err := d.GetById(id)
	if err != nil {
		return errorx.NewBiz("db not found")
	}
	dbs := strings.Split(db.Database, " ")
	for _, v := range dbs {
		// 关闭连接
		dbm.CloseDb(id, v)
	}

	return d.Tx(ctx,
		func(ctx context.Context) error {
			return d.DeleteById(ctx, id)
		},
		func(ctx context.Context) error {
			// 删除该库下用户保存的所有sql信息
			return d.dbSqlRepo.DeleteByCond(ctx, &entity.DbSql{DbId: id})
		}, func(ctx context.Context) error {
			return d.dbSqlExecApp.DeleteBy(ctx, &entity.DbSqlExec{DbId: id})
		}, func(ctx context.Context) error {
			return d.tagApp.DeleteTagByParam(ctx, &tagdto.DelResourceTag{
				ResourceCode: db.Code,
				ResourceType: tagentity.TagTypeDb,
			})
		})
}

func (d *dbAppImpl) GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error) {
	return dbm.GetDbConn(ctx, dbId, dbName, func() (*dbi.DbInfo, error) {
		db, err := d.GetById(dbId)
		if err != nil {
			return nil, errorx.NewBiz("db not found")
		}

		instance, err := d.dbInstanceApp.GetById(db.InstanceId)
		if err != nil {
			return nil, errorx.NewBiz("db instance not found")
		}

		di, err := d.dbInstanceApp.ToDbInfo(instance, db.AuthCertName, dbName)
		if err != nil {
			return nil, err
		}
		di.CodePath = d.tagApp.ListTagPathByTypeAndCode(int8(tagentity.TagTypeDb), db.Code)
		di.Id = db.Id

		checkDb := di.GetDatabase()
		if db.GetDatabaseMode == entity.DbGetDatabaseModeAssign && !strings.Contains(" "+db.Database+" ", " "+checkDb+" ") {
			return nil, errorx.NewBizI(context.Background(), imsg.ErrDbNotAccess, "dbName", dbName)
		}

		return di, nil
	})
}

func (d *dbAppImpl) GetDbConnByInstanceId(ctx context.Context, instanceId uint64) (*dbi.DbConn, error) {
	conn := dbm.GetDbConnByInstanceId(ctx, instanceId)
	if conn != nil {
		return conn, nil
	}

	dbs, err := d.ListByCond(&entity.Db{InstanceId: instanceId}, "id", "database")
	if err != nil {
		return nil, errorx.NewBiz("failed to get database list")
	}
	if len(dbs) == 0 {
		return nil, errorx.NewBizf("DB instance [%d] database is not configured, please configure it first", instanceId)
	}

	// 使用该实例关联的已配置数据库中的第一个库进行连接并返回
	firstDb := dbs[0]
	return d.GetDbConn(ctx, firstDb.Id, strings.Split(firstDb.Database, " ")[0])
}

func (d *dbAppImpl) DumpDb(ctx context.Context, reqParam *dto.DumpDb) error {
	log := dto.DefaultDumpLog
	if reqParam.Log != nil {
		log = reqParam.Log
	}
	progress := dto.DefaultDumpProgress
	if reqParam.Progress != nil {
		progress = reqParam.Progress
	}

	writer := writerx.NewStringWriter(reqParam.Writer)
	defer writer.Close()
	dbId := reqParam.DbId
	dbName := reqParam.DbName
	tables := reqParam.Tables

	dbConn, err := d.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return err
	}

	writer.WriteString("\n-- ----------------------------")
	writer.WriteString("\n-- Dump Platform: mayfly-go")
	writer.WriteString(fmt.Sprintf("\n-- Dump Time: %s ", time.Now().Format("2006-01-02 15:04:05")))
	writer.WriteString(fmt.Sprintf("\n-- Dump DB: %s ", dbName))
	writer.WriteString(fmt.Sprintf("\n-- DB Dialect: %s ", cmp.Or(reqParam.TargetDbType, dbConn.Info.Type)))
	writer.WriteString("\n-- ----------------------------\n\n")

	// 获取目标元数据，仅生成sql，用于生成建表语句和插入数据，不能用于查询
	targetDialect := dbConn.GetDialect()
	if reqParam.TargetDbType != "" && dbConn.Info.Type != reqParam.TargetDbType {
		targetDialect = dbi.GetDialect(reqParam.TargetDbType)
	}

	srcMeta := dbConn.GetMetadata()
	srcDialect := dbConn.GetDialect()
	if len(tables) == 0 {
		log("gets the table information that can be export...")
		ti, err := srcMeta.GetTables()
		if err != nil {
			log(fmt.Sprintf("failed to get table info %s", err.Error()))
		}
		biz.ErrIsNil(err)
		tables = make([]string, len(ti))
		for i, table := range ti {
			tables[i] = table.TableName
		}
		log(fmt.Sprintf("Get %d tables", len(tables)))
	}
	if len(tables) == 0 {
		log("no table to export. end export")
		return errorx.NewBiz("there is no table to export")
	}

	log("querying column information...")
	// 查询列信息，后面生成建表ddl和insert都需要列信息
	columns, err := srcMeta.GetColumns(tables...)
	if err != nil {
		log(fmt.Sprintf("failed to query column information: %s", err.Error()))
	}
	biz.ErrIsNil(err)

	// 以表名分组，存放每个表的列信息
	columnMap := make(map[string][]dbi.Column)
	for _, column := range columns {
		if err := dbi.ConvToTargetDbColumn(dbConn.Info.Type, cmp.Or(reqParam.TargetDbType, dbConn.Info.Type), targetDialect, &column); err != nil {
			return err
		}
		columnMap[column.TableName] = append(columnMap[column.TableName], column)
	}

	// 按表名排序
	sort.Strings(tables)
	quoteSchema := srcDialect.Quoter().Quote(dbConn.Info.CurrentSchema())
	dumpHelper := targetDialect.GetDumpHelper()

	targetSqlGenerator := targetDialect.GetSQLGenerator()
	targetDialectQuote := targetDialect.Quoter().Quote
	// 遍历获取每个表的信息
	for _, tableName := range tables {
		log(fmt.Sprintf("get table [%s] information...", tableName))
		quoteTableName := targetDialectQuote(tableName)

		// 查询表信息，主要是为了查询表注释
		tbs, err := srcMeta.GetTables(tableName)
		if err != nil {
			log(fmt.Sprintf("failed to get table [%s] information: %s", tableName, err.Error()))
			return err
		}
		if len(tbs) <= 0 {
			log(fmt.Sprintf("failed to get table [%s] information: No table information was retrieved", tableName))
			return errorx.NewBizf("Failed to get table information: %s", tableName)
		}

		tableInfo := tbs[0]
		columns := columnMap[tableName]

		// 生成表结构信息
		if reqParam.DumpDDL {
			log(fmt.Sprintf("generate table [%s] DDL...", tableName))
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- Table structure: %s \n-- ----------------------------\n", tableName))
			tbDdlArr := targetSqlGenerator.GenTableDDL(tableInfo, columns, true)
			for _, ddl := range tbDdlArr {
				if _, err := writer.WriteString(ddl + ";\n"); err != nil {
					return err
				}
			}
			progress(tableName, dbi.StmtTypeDDL, len(tbDdlArr), true)
		}

		// 生成insert sql，数据在索引前，加速insert
		if reqParam.DumpData {
			log(fmt.Sprintf("generate table [%s] DML...", tableName))
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- Data: %s \n-- ----------------------------\n", tableName))

			dumpHelper.BeforeInsert(writer, quoteTableName)

			dataCount := 0
			rows := make([][]any, 0)
			_, err = dbConn.WalkTableRows(ctx, tableName, func(row map[string]any, _ []*dbi.QueryColumn) error {
				rowValues := make([]any, len(columns))
				for i, col := range columns {
					rowValues[i] = row[col.ColumnName]
				}
				rows = append(rows, rowValues)
				dataCount++
				if dataCount%500 != 0 {
					return nil
				}

				beforeInsert := dumpHelper.BeforeInsertSql(quoteSchema, quoteTableName)
				if beforeInsert != "" {
					writer.WriteString(beforeInsert)
				}
				insertSql := targetSqlGenerator.GenInsert(tableName, columns, rows, dbi.DuplicateStrategyNone)
				if _, err := writer.WriteString(strings.Join(insertSql, ";\n") + ";\n"); err != nil {
					return err
				}
				progress(tableName, dbi.StmtTypeInsert, dataCount, false)
				rows = make([][]any, 0)
				return nil
			})

			if err != nil {
				return err
			}

			if len(rows) > 0 {
				beforeInsert := dumpHelper.BeforeInsertSql(quoteSchema, quoteTableName)
				if beforeInsert != "" {
					writer.WriteString(beforeInsert)
				}
				insertSql := targetSqlGenerator.GenInsert(tableName, columns, rows, dbi.DuplicateStrategyNone)
				if _, err := writer.WriteString(strings.Join(insertSql, ";\n") + ";\n"); err != nil {
					return err
				}
			}

			dumpHelper.AfterInsert(writer, tableName, columns)
			progress(tableName, dbi.StmtTypeInsert, dataCount, true)
		}

		log(fmt.Sprintf("get table [%s] index information...", tableName))
		indexs, err := srcMeta.GetTableIndex(tableName)
		if err != nil {
			log(fmt.Sprintf("failed to get table [%s] index information: %s", tableName, err.Error()))
			return err
		}

		if len(indexs) > 0 {
			// 最后添加索引
			log(fmt.Sprintf("generate table [%s] index...", tableName))
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- Table Index: %s \n-- ----------------------------\n", tableName))
			sqlArr := targetSqlGenerator.GenIndexDDL(tableInfo, indexs)
			for _, sqlStr := range sqlArr {
				if _, err := writer.WriteString(sqlStr + ";\n"); err != nil {
					return err
				}
			}
			progress(tableName, dbi.StmtTypeDDL, len(sqlArr), true)
		}
	}

	return nil
}
