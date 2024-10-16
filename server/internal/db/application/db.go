package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"sort"
	"strings"
	"time"
)

type Db interface {
	base.App[*entity.Db]

	// 分页获取
	GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	SaveDb(ctx context.Context, entity *entity.Db) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取数据库连接实例
	// @param id 数据库id
	//
	// @param dbName 数据库名
	GetDbConn(dbId uint64, dbName string) (*dbi.DbConn, error)

	// 根据数据库实例id获取连接，随机返回该instanceId下已连接的conn，若不存在则是使用该instanceId关联的db进行连接并返回。
	GetDbConnByInstanceId(instanceId uint64) (*dbi.DbConn, error)

	// DumpDb dumpDb
	DumpDb(ctx context.Context, reqParam *dto.DumpDb) error
}

type dbAppImpl struct {
	base.AppImpl[*entity.Db, repository.Db]

	dbSqlRepo           repository.DbSql        `inject:"DbSqlRepo"`
	dbInstanceApp       Instance                `inject:"DbInstanceApp"`
	dbSqlExecApp        DbSqlExec               `inject:"DbSqlExecApp"`
	tagApp              tagapp.TagTree          `inject:"TagTreeApp"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"ResourceAuthCertApp"`
}

var _ (Db) = (*dbAppImpl)(nil)

// 注入DbRepo
func (d *dbAppImpl) InjectDbRepo(repo repository.Db) {
	d.Repo = repo
}

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.GetRepo().GetDbList(condition, pageParam, toEntity, orderBy...)
}

func (d *dbAppImpl) SaveDb(ctx context.Context, dbEntity *entity.Db) error {
	// 查找是否存在
	oldDb := &entity.Db{Name: dbEntity.Name, InstanceId: dbEntity.InstanceId}

	authCert, err := d.resourceAuthCertApp.GetAuthCert(dbEntity.AuthCertName)
	if err != nil {
		return errorx.NewBiz("授权凭证不存在")
	}

	err = d.GetByCond(oldDb)
	if dbEntity.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该实例下数据库名已存在")
		}
		if d.CountByCond(&entity.Db{Name: dbEntity.Name}) > 0 {
			return errorx.NewBiz("该编码已存在")
		}

		dbEntity.Code = stringx.Rand(10)

		return d.Tx(ctx, func(ctx context.Context) error {
			return d.Insert(ctx, dbEntity)
		}, func(ctx context.Context) error {
			// 将库关联至指定数据库授权凭证下
			return d.tagApp.RelateTagsByCodeAndType(ctx, &tagdto.RelateTagsByCodeAndType{
				Tags: []*tagdto.ResourceTag{{
					Code: dbEntity.Code,
					Type: tagentity.TagTypeDbName,
					Name: dbEntity.Name,
				}},
				ParentTagCode: authCert.Name,
				ParentTagType: tagentity.TagTypeDbAuthCert,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldDb.Id != dbEntity.Id {
		return errorx.NewBiz("该实例下数据库名已存在")
	}

	dbId := dbEntity.Id
	old, err := d.GetById(dbId)
	if err != nil {
		return errorx.NewBiz("该数据库不存在")
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
			if err := d.tagApp.UpdateTagName(ctx, tagentity.TagTypeDbName, old.Code, dbEntity.Name); err != nil {
				return err
			}
		}
		if authCert.Name != old.AuthCertName {
			return d.tagApp.ChangeParentTag(ctx, tagentity.TagTypeDbName, old.Code, tagentity.TagTypeDbAuthCert, authCert.Name)
		}
		return nil
	})
}

func (d *dbAppImpl) Delete(ctx context.Context, id uint64) error {
	db, err := d.GetById(id)
	if err != nil {
		return errorx.NewBiz("该数据库不存在")
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
				ResourceType: tagentity.TagTypeDbName,
			})
		})
}

func (d *dbAppImpl) GetDbConn(dbId uint64, dbName string) (*dbi.DbConn, error) {
	return dbm.GetDbConn(dbId, dbName, func() (*dbi.DbInfo, error) {
		db, err := d.GetById(dbId)
		if err != nil {
			return nil, errorx.NewBiz("数据库信息不存在")
		}

		instance, err := d.dbInstanceApp.GetById(db.InstanceId)
		if err != nil {
			return nil, errorx.NewBiz("数据库实例不存在")
		}

		di, err := d.dbInstanceApp.ToDbInfo(instance, db.AuthCertName, dbName)
		if err != nil {
			return nil, err
		}
		di.CodePath = d.tagApp.ListTagPathByTypeAndCode(int8(tagentity.TagTypeDbName), db.Code)
		di.Id = db.Id

		checkDb := di.GetDatabase()
		if db.GetDatabaseMode == entity.DbGetDatabaseModeAssign && !strings.Contains(" "+db.Database+" ", " "+checkDb+" ") {
			return nil, errorx.NewBiz("未配置数据库【%s】的操作权限", dbName)
		}

		return di, nil
	})
}

func (d *dbAppImpl) GetDbConnByInstanceId(instanceId uint64) (*dbi.DbConn, error) {
	conn := dbm.GetDbConnByInstanceId(instanceId)
	if conn != nil {
		return conn, nil
	}

	dbs, err := d.ListByCond(&entity.Db{InstanceId: instanceId}, "id", "database")
	if err != nil {
		return nil, errorx.NewBiz("获取数据库列表失败")
	}
	if len(dbs) == 0 {
		return nil, errorx.NewBiz("实例[%d]未配置数据库, 请先进行配置", instanceId)
	}

	// 使用该实例关联的已配置数据库中的第一个库进行连接并返回
	firstDb := dbs[0]
	return d.GetDbConn(firstDb.Id, strings.Split(firstDb.Database, " ")[0])
}

func (d *dbAppImpl) DumpDb(ctx context.Context, reqParam *dto.DumpDb) error {
	writer := newGzipWriter(reqParam.Writer)
	defer writer.Close()
	dbId := reqParam.DbId
	dbName := reqParam.DbName
	tables := reqParam.Tables

	dbConn, err := d.GetDbConn(dbId, dbName)
	if err != nil {
		return err
	}
	writer.WriteString("\n-- ----------------------------")
	writer.WriteString("\n-- 导出平台: mayfly-go")
	writer.WriteString(fmt.Sprintf("\n-- 导出时间: %s ", time.Now().Format("2006-01-02 15:04:05")))
	writer.WriteString(fmt.Sprintf("\n-- 导出数据库: %s ", dbName))
	writer.WriteString("\n-- ----------------------------\n\n")

	dbMeta := dbConn.GetMetaData()
	if len(tables) == 0 {
		ti, err := dbMeta.GetTables()
		biz.ErrIsNil(err)
		tables = make([]string, len(ti))
		for i, table := range ti {
			tables[i] = table.TableName
		}
	}
	if len(tables) == 0 {
		return errorx.NewBiz("不存在可导出的表")
	}

	// 查询列信息，后面生成建表ddl和insert都需要列信息
	columns, err := dbMeta.GetColumns(tables...)
	biz.ErrIsNil(err)

	// 以表名分组，存放每个表的列信息
	columnMap := make(map[string][]dbi.Column)
	for _, column := range columns {
		columnMap[column.TableName] = append(columnMap[column.TableName], column)
	}

	// 按表名排序
	sort.Strings(tables)

	quoteSchema := dbMeta.QuoteIdentifier(dbConn.Info.CurrentSchema())
	dumpHelper := dbMeta.GetDumpHelper()
	dataHelper := dbMeta.GetDataHelper()

	// 遍历获取每个表的信息
	for _, tableName := range tables {
		quoteTableName := dbMeta.QuoteIdentifier(tableName)

		writer.TryFlush()
		// 查询表信息，主要是为了查询表注释
		tbs, err := dbMeta.GetTables(tableName)
		if err != nil {
			return err
		}
		if len(tbs) <= 0 {
			return errorx.NewBiz(fmt.Sprintf("获取表信息失败：%s", tableName))
		}
		tabInfo := dbi.Table{
			TableName:    tableName,
			TableComment: tbs[0].TableComment,
		}

		// 生成表结构信息
		if reqParam.DumpDDL {
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表结构: %s \n-- ----------------------------\n", tableName))
			tbDdlArr := dbMeta.GenerateTableDDL(columnMap[tableName], tabInfo, true)
			for _, ddl := range tbDdlArr {
				writer.WriteString(ddl + ";\n")
			}
		}

		// 生成insert sql，数据在索引前，加速insert
		if reqParam.DumpData {
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表数据: %s \n-- ----------------------------\n", tableName))

			dumpHelper.BeforeInsert(writer, quoteTableName)
			// 获取列信息
			quoteColNames := make([]string, 0)
			for _, col := range columnMap[tableName] {
				quoteColNames = append(quoteColNames, dbMeta.QuoteIdentifier(col.ColumnName))
			}

			_, _ = dbConn.WalkTableRows(ctx, quoteTableName, func(row map[string]any, _ []*dbi.QueryColumn) error {
				rowValues := make([]string, len(columnMap[tableName]))
				for i, col := range columnMap[tableName] {
					rowValues[i] = dataHelper.WrapValue(row[col.ColumnName], dataHelper.GetDataType(string(col.DataType)))
				}

				beforeInsert := dumpHelper.BeforeInsertSql(quoteSchema, quoteTableName)
				insertSQL := fmt.Sprintf("%s INSERT INTO %s (%s) values(%s)", beforeInsert, quoteTableName, strings.Join(quoteColNames, ", "), strings.Join(rowValues, ", "))
				writer.WriteString(insertSQL + ";\n")
				return nil
			})

			dumpHelper.AfterInsert(writer, tableName, columnMap[tableName])
		}

		indexs, err := dbMeta.GetTableIndex(tableName)
		if err != nil {
			return err
		}

		if len(indexs) > 0 {
			// 最后添加索引
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表索引: %s \n-- ----------------------------\n", tableName))
			sqlArr := dbMeta.GenerateIndexDDL(indexs, tabInfo)
			for _, sqlStr := range sqlArr {
				writer.WriteString(sqlStr + ";\n")
			}
		}

	}

	return nil
}
