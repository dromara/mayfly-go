package migrations

import (
	"errors"
	dbentity "mayfly-go/internal/db/domain/entity"
	dockerentity "mayfly-go/internal/docker/domain/entity"
	esentity "mayfly-go/internal/es/domain/entity"
	flowentity "mayfly-go/internal/flow/domain/entity"
	machineentity "mayfly-go/internal/machine/domain/entity"
	msgentity "mayfly-go/internal/msg/domain/entity"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func V1_10() []*gormigrate.Migration {
	var migrations []*gormigrate.Migration
	migrations = append(migrations, V1_10_0()...)
	migrations = append(migrations, V1_10_1()...)
	migrations = append(migrations, V1_10_2()...)
	migrations = append(migrations, V1_10_3()...)
	migrations = append(migrations, V1_10_4()...)
	migrations = append(migrations, V1_10_5()...)
	migrations = append(migrations, V1_10_6()...)
	return migrations
}

func V1_10_0() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250520-v1.10.0-flow-recode",
			Migrate: func(tx *gorm.DB) error {
				err := tx.AutoMigrate(&flowentity.Procdef{},
					&flowentity.Procinst{},
					&flowentity.Execution{},
					&flowentity.ProcinstTask{},
					&flowentity.ProcinstTaskCandidate{},
					&flowentity.HisProcinstOp{})
				if err != nil {
					return err
				}

				// 添加实例表
				entities := [...]any{
					new(esentity.EsInstance),
				}
				for _, e := range entities {
					if err := tx.AutoMigrate(e); err != nil {
						return err
					}
				}

				// 添加ES相关菜单资源
				resources := []*sysentity.Resource{
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1745292787}}}},
						Pid:    0,
						UiPath: "lbOU73qg/",
						Name:   "Elasticsearch",
						Code:   "/es",
						Type:   1,
						Meta:   `{"icon":"icon es/es-color","isKeepAlive":true,"routeName":"ES"}`,
						Weight: 50000001,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1745319348}}}},
						Pid:    1745292787,
						UiPath: "lbOU73qg/gZ2MHF0b/",
						Name:   "es.instance",
						Code:   "es-instance ",
						Type:   1,
						Meta:   `{"component":"ops/es/EsInstanceList","icon":"icon es/es-color","isKeepAlive":true,"routeName":"EsInstanceList"}`,
						Weight: 1745319348,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1745319410}}}},
						Pid:    1745319348,
						UiPath: "lbOU73qg/gZ2MHF0b/rcKBdxB5/",
						Name:   "es.instanceSave",
						Code:   "es:instance:save",
						Type:   2,
						Weight: 1745319410,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1745319424}}}},
						Pid:    1745319348,
						UiPath: "lbOU73qg/gZ2MHF0b/IMGhLSJK/",
						Name:   "es.instanceDel",
						Code:   "es:instance:del",
						Type:   2,
						Weight: 1745319424,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1745494931}}}},
						Pid:    1745292787,
						UiPath: "lbOU73qg/2sDi4isw/",
						Name:   "es.operation",
						Code:   "es-operation",
						Type:   1,
						Meta:   `{"component":"ops/es/EsOperation","icon":"icon es/es-color","isKeepAlive":true,"routeName":"EsOperation"}`,
						Weight: 1745319347,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1745659240}}}},
						Pid:    1745494931,
						UiPath: "lbOU73qg/2sDi4isw/SQNFhhhn/",
						Name:   "es.dataSave",
						Code:   "es:data:save",
						Type:   2,
						Weight: 1745659240,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1745659315}}}},
						Pid:    1745494931,
						UiPath: "lbOU73qg/2sDi4isw/XAgy5Uvp/",
						Name:   "es.dataDel",
						Code:   "es:data:del",
						Type:   2,
						Weight: 1745659315,
					},
				}
				now := time.Now()
				for _, res := range resources {
					res.Status = 1
					res.CreateTime = &now
					res.CreatorId = 1
					res.Creator = "admin"
					res.UpdateTime = &now
					res.ModifierId = 1
					res.Modifier = "admin"
					tx.Create(res)
				}
				// 给超管授权

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}

func V1_10_1() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250610-v1.10.1",
			Migrate: func(tx *gorm.DB) error {
				if !tx.Migrator().HasColumn(&machineentity.MachineScript{}, "category") {
					if err := tx.Migrator().AddColumn(&machineentity.MachineScript{}, "category"); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}

func V1_10_2() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250726-v1.10.2",
			Migrate: func(tx *gorm.DB) error {
				// 新增subtype,extra
				if err := tx.Migrator().AutoMigrate(&msgentity.Msg{}); err != nil {
					return err
				}

				// 机器列表相关菜单权限
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'machines'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'machines-op'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Alw1Xkq3/', pid=94 where ui_path = '12sSjal1/lskeiql1/Alw1Xkq3/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Lsew24Kx/', pid=94  where ui_path = '12sSjal1/lskeiql1/Lsew24Kx/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/exIsqL31/', pid=94  where ui_path = '12sSjal1/lskeiql1/exIsqL31/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Liwakg2x/', pid=94  where ui_path = '12sSjal1/lskeiql1/Liwakg2x/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Lieakenx/', pid=94  where ui_path = '12sSjal1/lskeiql1/Lieakenx/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Keiqkx4L/', pid=94  where ui_path = '12sSjal1/lskeiql1/Keiqkx4L/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Keal2Xke/', pid=94  where ui_path = '12sSjal1/lskeiql1/Keal2Xke/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Ihfs2xaw/', pid=94  where ui_path = '12sSjal1/lskeiql1/Ihfs2xaw/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/3ldkxJDx/', pid=94  where ui_path = '12sSjal1/lskeiql1/3ldkxJDx/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Ljewix43/', pid=94  where ui_path = '12sSjal1/lskeiql1/Ljewix43/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/L12wix43/', pid=94  where ui_path = '12sSjal1/lskeiql1/L12wix43/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Ljewisd3/', pid=94  where ui_path = '12sSjal1/lskeiql1/Ljewisd3/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Ljeew43/', pid=94  where ui_path = '12sSjal1/lskeiql1/Ljeew43/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/ODewix43/', pid=94  where ui_path = '12sSjal1/lskeiql1/ODewix43/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/OJewex43/', pid=94  where ui_path = '12sSjal1/lskeiql1/OJewex43/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/LIEwix43/', pid=94  where ui_path = '12sSjal1/lskeiql1/LIEwix43/'")

				// redis
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = '/redis'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'data-operation'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'manage'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/IoxqAd31/', pid=94  where ui_path = 'RedisXq4/Eoaljc12/IoxqAd31/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/IUlxia23/', pid=94  where ui_path = 'RedisXq4/Exitx4al/IUlxia23/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Gxlagheg/', pid=94  where ui_path = 'RedisXq4/Exitx4al/Gxlagheg/'")

				// db
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'instances'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'sql-exec'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'instances'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/mJlBeTCs/', pid=94 where ui_path = 'dbms23ax/X0f4BxT0/mJlBeTCs/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/D23fUiBr/', pid=94 where ui_path = 'dbms23ax/X0f4BxT0/D23fUiBr/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/Sgg8uPwz/', pid=94 where ui_path = 'dbms23ax/X0f4BxT0/Sgg8uPwz/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/AceXe321/', pid=94 where ui_path = 'dbms23ax/X0f4BxT0/AceXe321/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/leix3Axl/', pid=94 where ui_path = 'dbms23ax/X0f4BxT0/leix3Axl/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/ygjL3sxA/', pid=94 where ui_path = 'dbms23ax/X0f4BxT0/ygjL3sxA/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/TGFPA3Ez/', pid=94 where ui_path = 'dbms23ax/exaeca2x/TGFPA3Ez/'")

				// es
				tx.Exec("Update t_sys_resource set is_deleted = 1 where name = 'Elasticsearch'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where name = 'es.operation'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where name = 'es.instance'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/rcKBdxB5/', pid=94  where ui_path = 'lbOU73qg/gZ2MHF0b/rcKBdxB5/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/IMGhLSJK/', pid=94  where ui_path = 'lbOU73qg/gZ2MHF0b/IMGhLSJK/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/SQNFhhhn/', pid=94  where ui_path = 'lbOU73qg/2sDi4isw/SQNFhhhn/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/XAgy5Uvp/', pid=94  where ui_path = 'lbOU73qg/2sDi4isw/XAgy5Uvp/'")

				// mongo
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = '/mongo'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'mongo-data-operation'")
				tx.Exec("Update t_sys_resource set is_deleted = 1 where code = 'mongo-manage'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/egljbla3/', pid=94  where ui_path = 'Mongo452/ghxagl43/egljbla3/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/xvpKk36u/', pid=94  where ui_path = 'Mongo452/eggago31/xvpKk36u/'")
				tx.Exec("Update t_sys_resource set ui_path='Tag3fhad/glxajg23/3sblw1Wb/', pid=94  where ui_path = 'Mongo452/eggago31/3sblw1Wb/'")

				// 新增菜单
				resources := []*sysentity.Resource{{
					Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1756122788}}}},
					Pid:    0,
					UiPath: "ocdrUNaa/",
					Name:   "menu.myResource",
					Code:   "/my-resource",
					Meta:   `{"icon":"Menu","isKeepAlive":true,"routeName":"ResourceOp"}`,
					Type:   1,
					Weight: 19999998,
				}, {
					Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1756122789}}}},
					Pid:    2,
					UiPath: "12sSjal1/OgOqxbnt/",
					Name:   "docker.container",
					Code:   "/container",
					Meta:   `{"icon":"icon docker/docker","isKeepAlive":true,"routeName":"Container"}`,
					Type:   1,
					Weight: 1713875843,
				}}

				now := time.Now()
				for _, res := range resources {
					res.Status = 1
					res.CreateTime = &now
					res.CreatorId = 1
					res.Creator = "admin"
					res.UpdateTime = &now
					res.ModifierId = 1
					res.Modifier = "admin"
					tx.Create(res)
				}

				roleResource := &sysentity.RoleResource{
					RoleId:     1,
					ResourceId: 1756122788,
					CreateTime: &now,
					CreatorId:  1,
					Creator:    "admin",
				}
				tx.Create(roleResource)

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}

func V1_10_3() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250904-v1.10.3",
			Migrate: func(tx *gorm.DB) error {
				tx.AutoMigrate(&dockerentity.Container{})

				// 删除容器菜单
				tx.Exec("update t_sys_resource set is_deleted = 1 where code = '/container'")

				// 新增容器管理基本权限
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1757145306, 94, 'Tag3fhad/glxajg23/Bbrte5UH/', 2, 1, 'menu.containerManageBase', 'container', 1757145306, 'null', 1, 'admin', 1, 'admin', '2025-09-06 15:55:06', '2025-09-06 15:56:10', 0, NULL)")

				// 机器列表相关菜单权限
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Alw1Xkq3/', pid=1756122788 where ui_path = 'Tag3fhad/glxajg23/Alw1Xkq3/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Lsew24Kx/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Lsew24Kx/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Keiqkx4L/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Keiqkx4L/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Keal2Xke/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Keal2Xke/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Ihfs2xaw/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Ihfs2xaw/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/3ldkxJDx/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/3ldkxJDx/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Ljewix43/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Ljewix43/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/L12wix43/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/L12wix43/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Ljewisd3/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Ljewisd3/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Ljeew43/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Ljeew43/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/ODewix43/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/ODewix43/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/LIEwix43/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/LIEwix43/'")

				// redis
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/IUlxia23/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/IUlxia23/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/Gxlagheg/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/Gxlagheg/'")

				// db
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/TGFPA3Ez/', pid=1756122788 where ui_path = 'Tag3fhad/glxajg23/TGFPA3Ez/'")

				// es
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/SQNFhhhn/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/SQNFhhhn/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/XAgy5Uvp/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/XAgy5Uvp/'")

				// mongo
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/xvpKk36u/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/xvpKk36u/'")
				tx.Exec("Update t_sys_resource set ui_path='ocdrUNaa/3sblw1Wb/', pid=1756122788  where ui_path = 'Tag3fhad/glxajg23/3sblw1Wb/'")

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}

func V1_10_4() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20251023-v1.10.4",
			Migrate: func(tx *gorm.DB) error {
				// 给EsInstance表添加protocol列，默认值为http, 20251023,fudawei
				if !tx.Migrator().HasColumn(&esentity.EsInstance{}, "protocol") {
					// 先添加可为空的列
					if err := tx.Exec("ALTER TABLE t_es_instance ADD COLUMN protocol VARCHAR(10) DEFAULT 'http'").Error; err != nil {
						return err
					}
					// 更新所有现有记录为默认值http
					if err := tx.Exec("UPDATE t_es_instance SET protocol = 'http' WHERE protocol IS NULL OR protocol = ''").Error; err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "20251207-v1.10.4.1",
			Migrate: func(tx *gorm.DB) error {
				config := &sysentity.Config{}
				// 查询是否存在该配置
				result := tx.Model(config).Where("`key` = ?", "AiModelConfig").First(config)
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					// 如果不存在，则创建默认配置
					now := time.Now()
					aiConfig := &sysentity.Config{
						Key:        "AiModelConfig",
						Name:       "system.sysconf.aiModelConf",
						Value:      "{}", // 默认空JSON值
						Params:     `[{"model":"modelType","name":"system.sysconf.aiModelType","placeholder":"system.sysconf.aiModelTypePlaceholder","options":"openai"},{"model":"model","name":"system.sysconf.aiModel","placeholder":"system.sysconf.aiModelPlaceholder"},{"model":"baseUrl","name":"system.sysconf.aiBaseUrl","placeholder":"system.sysconf.aiBaseUrlPlaceholder"},{"model":"apiKey","name":"ApiKey","placeholder":"api key"}]`,
						Permission: "all",
					}
					aiConfig.CreateTime = &now
					aiConfig.Modifier = "admin"
					aiConfig.ModifierId = 1
					aiConfig.UpdateTime = &now
					aiConfig.Creator = "admin"
					aiConfig.CreatorId = 1
					aiConfig.IsDeleted = 0
					return tx.Create(aiConfig).Error
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}

func V1_10_5() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20260107-v1.10.5",
			Migrate: func(tx *gorm.DB) error {
				resource := &sysentity.Resource{}
				// 查询是否存在该配置
				result := tx.Model(resource).Where("code = ?", "db:data:export").First(resource)
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					// 如果不存在，则创建默认配置
					resource = &sysentity.Resource{
						Code:   "db:data:export",
						Name:   "menu.dbDataExport",
						Type:   sysentity.ResourceTypePermission,
						UiPath: "ocdrUNaa/fo59olyi/",
						Pid:    1756122788,
						Status: sysentity.ResourceStatusEnable,
					}
					resource.FillBaseInfo(model.IdGenTypeTimestamp, model.SysAccount)
					resource.Id = 1767788697
					return tx.Create(resource).Error
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}

func V1_10_6() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20260107-v1.10.6-addDbTransferTaskExtraColumn",
			Migrate: func(tx *gorm.DB) error {
				tx.AutoMigrate(&dbentity.DbTransferTask{})
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "20260107-v1.10.6-menuResourceChange",
			Migrate: func(tx *gorm.DB) error {
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768728951, 1756122788, 'ocdrUNaa/IV13ydwK/', 2, 1, 'menu.machine', 'machine:base', 29999998, 'null', 1, 'admin', 1, 'admin', '2026-01-18 17:35:52', '2026-01-18 18:02:40', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768729405, 1768728951, 'ocdrUNaa/IV13ydwK/V83yBBCM/', 2, 1, 'menu.machineScript', 'machine:script', 1768729405, 'null', 1, 'admin', 1, 'admin', '2026-01-18 17:43:25', '2026-01-18 17:49:39', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768729856, 1756122788, 'ocdrUNaa/sd9PqK7U/', 2, 1, 'Redis', 'redis:base', 1768729856, 'null', 1, 'admin', 1, 'admin', '2026-01-18 17:50:57', '2026-01-18 17:50:57', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768729911, 1756122788, 'ocdrUNaa/GCElqxQr/', 2, 1, 'Mongo', 'mongo', 1768729911, 'null', 1, 'admin', 1, 'admin', '2026-01-18 17:51:52', '2026-01-18 17:51:52', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768729997, 1756122788, 'ocdrUNaa/mcZUtOzR/', 2, 1, 'menu.db', 'db:base', 1768729855, 'null', 1, 'admin', 1, 'admin', '2026-01-18 17:53:18', '2026-01-18 18:02:53', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768730024, 1756122788, 'ocdrUNaa/BHfgTyd4/', 2, 1, 'Es', 'es:base', 1768730024, 'null', 1, 'admin', 1, 'admin', '2026-01-18 17:53:44', '2026-01-18 17:53:44', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768731024, 94, 'Tag3fhad/glxajg23/FyQOFAkk/', 2, 1, 'Es', 'res:es', 1757145305, 'null', 1, 'admin', 1, 'admin', '2026-01-18 18:10:24', '2026-01-18 18:10:48', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768731749, 64, 'Tag3fhad/glxajg23/IoxqAd31/fX8iVBwb/', 2, 1, 'menu.redisSave', 'redis:save', 1768731749, 'null', 1, 'admin', 1, 'admin', '2026-01-18 18:22:30', '2026-01-18 19:00:50', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768731764, 64, 'Tag3fhad/glxajg23/IoxqAd31/ng7B41YS/', 2, 1, 'menu.redisDel', 'redis:del', 1768731764, 'null', 1, 'admin', 1, 'admin', '2026-01-18 18:22:45', '2026-01-18 19:00:55', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768733843, 83, 'Tag3fhad/glxajg23/egljbla3/tiBNOllg/', 2, 1, 'menu.mongoSave', 'mongo:save', 1768733843, 'null', 1, 'admin', 1, 'admin', '2026-01-18 18:57:24', '2026-01-18 18:57:24', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768733856, 83, 'Tag3fhad/glxajg23/egljbla3/S4lXWpyV/', 2, 1, 'menu.mongoDel', 'mongo:del', 1768733856, 'null', 1, 'admin', 1, 'admin', '2026-01-18 18:57:36', '2026-01-18 18:57:36', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768734233, 1757145306, 'Tag3fhad/glxajg23/Bbrte5UH/emC3wrlg/', 2, 1, 'menu.containerSave', 'container:save', 1768734233, 'null', 1, 'admin', 1, 'admin', '2026-01-18 19:03:54', '2026-01-18 19:03:54', 0, NULL)")
				tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1768734253, 1757145306, 'Tag3fhad/glxajg23/Bbrte5UH/ZcXsgEYp/', 2, 1, 'menu.containerDel', 'container:del', 1768734253, 'null', 1, 'admin', 1, 'admin', '2026-01-18 19:04:14', '2026-01-18 19:04:14', 0, NULL)")

				tx.Exec("UPDATE t_sys_resource SET pid=94,ui_path='Tag3fhad/glxajg23/Bbrte5UH/',type=2,status=1,name='menu.container',code='container',weight=1757145306,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2025-09-06 15:55:06',update_time='2026-01-18 19:02:57',is_deleted=0,delete_time=NULL WHERE id=1757145306")
				tx.Exec("UPDATE t_sys_resource SET pid=94,ui_path='Tag3fhad/glxajg23/egljbla3/',type=2,status=1,name='Mongo',code='mongo:manage:base',weight=1757145304,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2022-05-16 18:13:25',update_time='2026-01-18 18:57:12',is_deleted=0,delete_time=NULL WHERE id=83")
				tx.Exec("UPDATE t_sys_resource SET pid=1768731024,ui_path='Tag3fhad/glxajg23/FyQOFAkk/rcKBdxB5/',type=2,status=1,name='es.instanceSave',code='es:instance:save',weight=-1,meta='',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2025-05-21 12:44:18',update_time='2026-01-18 18:10:43',is_deleted=0,delete_time=NULL WHERE id=1745319410")
				tx.Exec("UPDATE t_sys_resource SET pid=1768731024,ui_path='Tag3fhad/glxajg23/FyQOFAkk/IMGhLSJK/',type=2,status=1,name='es.instanceDel',code='es:instance:del',weight=0,meta='',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2025-05-21 12:44:18',update_time='2026-01-18 18:10:41',is_deleted=0,delete_time=NULL WHERE id=1745319424")
				tx.Exec("UPDATE t_sys_resource SET pid=94,ui_path='Tag3fhad/glxajg23/mJlBeTCs/',type=2,status=1,name='menu.db',code='db:instance',weight=9999999,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2023-08-26 09:10:55',update_time='2026-01-18 18:09:25',is_deleted=0,delete_time=NULL WHERE id=137")
				tx.Exec("UPDATE t_sys_resource SET pid=94,ui_path='Tag3fhad/glxajg23/IoxqAd31/',type=2,status=1,name='menu.redis',code='redis:manage',weight=10000000,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-07-20 10:48:26',update_time='2026-01-18 18:09:07',is_deleted=0,delete_time=NULL WHERE id=64")
				tx.Exec("UPDATE t_sys_resource SET pid=94,ui_path='Tag3fhad/glxajg23/OJewex43/',type=2,status=1,name='menu.machine',code='machine',weight=9999999,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-07-09 10:48:02',update_time='2026-01-18 18:08:50',is_deleted=0,delete_time=NULL WHERE id=57")
				tx.Exec("UPDATE t_sys_resource SET pid=137,ui_path='Tag3fhad/glxajg23/mJlBeTCs/AceXe321/',type=2,status=1,name='menu.dbBase',code='db',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-07-09 10:48:22',update_time='2026-01-18 18:08:37',is_deleted=0,delete_time=NULL WHERE id=58")
				tx.Exec("UPDATE t_sys_resource SET pid=137,ui_path='Tag3fhad/glxajg23/mJlBeTCs/ygjL3sxA/',type=2,status=1,name='menu.dbDelete',code='db:del',weight=2,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-07-08 17:30:48',update_time='2026-01-18 18:08:18',is_deleted=0,delete_time=NULL WHERE id=55")
				tx.Exec("UPDATE t_sys_resource SET pid=137,ui_path='Tag3fhad/glxajg23/mJlBeTCs/Sgg8uPwz/',type=2,status=1,name='menu.dbInstanceDelete',code='db:instance:del',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2023-08-26 09:11:24',update_time='2026-01-18 18:08:13',is_deleted=0,delete_time=NULL WHERE id=138")
				tx.Exec("UPDATE t_sys_resource SET pid=137,ui_path='Tag3fhad/glxajg23/mJlBeTCs/D23fUiBr/',type=2,status=1,name='menu.dbInstanceSave',code='db:instance:save',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2023-08-26 09:10:02',update_time='2026-01-18 18:08:08',is_deleted=0,delete_time=NULL WHERE id=136")
				tx.Exec("UPDATE t_sys_resource SET pid=137,ui_path='Tag3fhad/glxajg23/mJlBeTCs/leix3Axl/',type=2,status=1,name='menu.dbSave',code='db:save',weight=1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-07-08 17:30:36',update_time='2026-01-18 18:07:57',is_deleted=0,delete_time=NULL WHERE id=54")
				tx.Exec("UPDATE t_sys_resource SET pid=57,ui_path='Tag3fhad/glxajg23/OJewex43/exIsqL31/',type=2,status=1,name='menu.machineCreate',code='machine:add',weight=-2,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-05-31 17:46:11',update_time='2026-01-18 18:06:44',is_deleted=0,delete_time=NULL WHERE id=16")
				tx.Exec("UPDATE t_sys_resource SET pid=57,ui_path='Tag3fhad/glxajg23/OJewex43/Liwakg2x/',type=2,status=1,name='menu.machineEdit',code='machine:update',weight=-1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-05-31 17:46:23',update_time='2026-01-18 18:06:33',is_deleted=0,delete_time=NULL WHERE id=17")
				tx.Exec("UPDATE t_sys_resource SET pid=57,ui_path='Tag3fhad/glxajg23/OJewex43/Lieakenx/',type=2,status=1,name='menu.machineDelete',code='machine:del',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-05-31 17:46:36',update_time='2026-01-18 18:06:29',is_deleted=0,delete_time=NULL WHERE id=18")
				tx.Exec("UPDATE t_sys_resource SET pid=15,ui_path='ocdrUNaa/IV13ydwK/Lsew24Kx/Ihfs2xaw/',type=2,status=1,name='menu.machineFileConfDelete',code='machine:file:del',weight=-1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:06:49',update_time='2026-01-18 17:58:42',is_deleted=0,delete_time=NULL WHERE id=41")
				tx.Exec("UPDATE t_sys_resource SET pid=15,ui_path='ocdrUNaa/IV13ydwK/Lsew24Kx/L12wix43/',type=2,status=1,name='menu.machineFileDelete',code='machine:file:rm',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:08:12',update_time='2026-01-18 17:58:35',is_deleted=0,delete_time=NULL WHERE id=44")
				tx.Exec("UPDATE t_sys_resource SET pid=15,ui_path='ocdrUNaa/IV13ydwK/Lsew24Kx/Keiqkx4L/',type=2,status=1,name='menu.machineFileConfCreate',code='machine:addFile',weight=-1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-01 19:54:23',update_time='2026-01-18 17:56:02',is_deleted=0,delete_time=NULL WHERE id=37")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729911,ui_path='ocdrUNaa/BHfgTyd4/TGFPA3Ez/TGFPA3Ez/TGFPA3Ez/',type=2,status=1,name='menu.mongoDataOpDelete',code='mongo:data:del',weight=1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2023-08-22 11:29:24',update_time='2026-01-18 17:54:04',is_deleted=0,delete_time=NULL WHERE id=134")
				tx.Exec("UPDATE t_sys_resource SET pid=1768730024,ui_path='ocdrUNaa/BHfgTyd4/TGFPA3Ez/',type=2,status=1,name='es.dataDel',code='es:data:del',weight=2,meta='',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2025-05-21 12:44:18',update_time='2026-01-18 17:54:04',is_deleted=0,delete_time=NULL WHERE id=1745659315")
				tx.Exec("UPDATE t_sys_resource SET pid=1768730024,ui_path='ocdrUNaa/BHfgTyd4/TGFPA3Ez/',type=2,status=1,name='es.dataSave',code='es:data:save',weight=1,meta='',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2025-05-21 12:44:18',update_time='2026-01-18 17:53:49',is_deleted=0,delete_time=NULL WHERE id=1745659240")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729997,ui_path='ocdrUNaa/mcZUtOzR/fo59olyi/',type=2,status=1,name='menu.dbDataExport',code='db:data:export',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2026-01-07 20:24:57',update_time='2026-01-18 17:53:31',is_deleted=0,delete_time=NULL WHERE id=1767788697")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729997,ui_path='ocdrUNaa/mcZUtOzR/TGFPA3Ez/',type=2,status=1,name='menu.dbDataOpSqlScriptRun',code='db:sqlscript:run',weight=1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2024-10-23 15:22:12',update_time='2026-01-18 17:53:27',is_deleted=0,delete_time=NULL WHERE id=1729668131")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729911,ui_path='ocdrUNaa/GCElqxQr/TGFPA3Ez/',type=2,status=1,name='menu.mongoDataOpSave',code='mongo:data:save',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2023-08-22 11:29:04',update_time='2026-01-18 17:52:23',is_deleted=0,delete_time=NULL WHERE id=133")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729856,ui_path='ocdrUNaa/sd9PqK7U/IUlxia23/',type=2,status=1,name='menu.redisDataOpSave',code='redis:data:save',weight=-1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-08-17 11:20:37',update_time='2026-01-18 17:51:21',is_deleted=0,delete_time=NULL WHERE id=71")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729856,ui_path='ocdrUNaa/sd9PqK7U/Gxlagheg/',type=2,status=1,name='menu.redisDataOpDelete',code='redis:data:del',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2023-03-14 17:20:00',update_time='2026-01-18 17:51:18',is_deleted=0,delete_time=NULL WHERE id=108")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729405,ui_path='ocdrUNaa/IV13ydwK/V83yBBCM/Ljeew43/',type=2,status=1,name='menu.machineScriptDelete',code='machine:script:del',weight=1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:09:27',update_time='2026-01-18 17:50:13',is_deleted=0,delete_time=NULL WHERE id=46")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729405,ui_path='ocdrUNaa/IV13ydwK/V83yBBCM/ODewix43/',type=2,status=1,name='menu.machineScriptRun',code='machine:script:run',weight=2,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:09:50',update_time='2026-01-18 17:50:10',is_deleted=0,delete_time=NULL WHERE id=47")
				tx.Exec("UPDATE t_sys_resource SET pid=1768729405,ui_path='ocdrUNaa/IV13ydwK/V83yBBCM/Ljewisd3/',type=2,status=1,name='menu.machineScriptSave',code='machine:script:save',weight=1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:09:01',update_time='2026-01-18 17:49:50',is_deleted=0,delete_time=NULL WHERE id=45")
				tx.Exec("UPDATE t_sys_resource SET pid=15,ui_path='ocdrUNaa/IV13ydwK/Lsew24Kx/Ljewix43/',type=2,status=1,name='menu.machineFileUpload',code='machine:file:upload',weight=5,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:07:42',update_time='2026-01-18 17:45:03',is_deleted=0,delete_time=NULL WHERE id=43")
				tx.Exec("UPDATE t_sys_resource SET pid=15,ui_path='ocdrUNaa/IV13ydwK/Lsew24Kx/3ldkxJDx/',type=2,status=1,name='menu.machineFileWrite',code='machine:file:write',weight=2,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:07:27',update_time='2026-01-18 17:44:46',is_deleted=0,delete_time=NULL WHERE id=42")
				tx.Exec("UPDATE t_sys_resource SET pid=15,ui_path='ocdrUNaa/IV13ydwK/Lsew24Kx/Keal2Xke/',type=2,status=1,name='menu.machineFileCreate',code='machine:file:add',weight=1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-06-08 11:06:26',update_time='2026-01-18 17:44:38',is_deleted=0,delete_time=NULL WHERE id=40")
				tx.Exec("UPDATE t_sys_resource SET pid=1768728951,ui_path='ocdrUNaa/IV13ydwK/Lsew24Kx/',type=2,status=1,name='menu.machineFileConf',code='machine:file',weight=1768729331,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-05-31 17:44:37',update_time='2026-01-18 17:44:34',is_deleted=0,delete_time=NULL WHERE id=15")
				tx.Exec("UPDATE t_sys_resource SET pid=1768728951,ui_path='ocdrUNaa/IV13ydwK/LIEwix43/',type=2,status=1,name='menu.machineKillprocess',code='machine:killprocess',weight=0,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-08-17 11:20:37',update_time='2026-01-18 17:41:40',is_deleted=0,delete_time=NULL WHERE id=72")
				tx.Exec("UPDATE t_sys_resource SET pid=1768728951,ui_path='ocdrUNaa/IV13ydwK/Alw1Xkq3/',type=2,status=1,name='menu.machineTerminal',code='machine:terminal',weight=1,meta='null',creator_id=1,creator='admin',modifier_id=1,modifier='admin',create_time='2021-05-28 14:06:02',update_time='2026-01-18 17:41:26',is_deleted=0,delete_time=NULL WHERE id=12")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}
