package migrations

import (
	esentity "mayfly-go/internal/es/domain/entity"
	flowentity "mayfly-go/internal/flow/domain/entity"
	machineentity "mayfly-go/internal/machine/domain/entity"
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
