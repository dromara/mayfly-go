package migrations

import (
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
