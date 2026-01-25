package migrations

import (
	machineentity "mayfly-go/internal/machine/domain/entity"
	msgentity "mayfly-go/internal/msg/domain/entity"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func V1_9() []*gormigrate.Migration {
	var migrations []*gormigrate.Migration
	migrations = append(migrations, V1_9_3()...)
	migrations = append(migrations, V1_9_4()...)
	return migrations
}

func V1_9_3() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250213-v1.9.3-addMachineExtra-updateMenuIcon",
			Migrate: func(tx *gorm.DB) error {
				tx.AutoMigrate(&machineentity.Machine{})

				// 更新菜单图标
				resourceModel := &sysentity.Resource{}
				tx.Model(resourceModel).Where("id = ?", 11).Update("meta", `{"component":"system/role/RoleList","icon":"icon menu/role","isKeepAlive":true,"routeName":"RoleList"}`)
				tx.Model(resourceModel).Where("id = ?", 14).Update("meta", `{"component":"system/account/AccountList","icon":"User","isKeepAlive":true,"routeName":"AccountList"}`)
				tx.Model(resourceModel).Where("id = ?", 150).Update("meta", `{"component":"ops/db/SyncTaskList","icon":"Refresh","isKeepAlive":true,"routeName":"SyncTaskList"}`)
				tx.Model(resourceModel).Where("id = ?", 60).Update("meta", `{"icon":"icon redis/redis","isKeepAlive":true,"routeName":"RDS"}`)
				tx.Model(resourceModel).Where("id = ?", 61).Update("meta", `{"component":"ops/redis/DataOperation","icon":"icon redis/redis","isKeepAlive":true,"routeName":"DataOperation"}`)
				tx.Model(resourceModel).Where("id = ?", 63).Update("meta", `{"component":"ops/redis/RedisList","icon":"icon redis/redis","isKeepAlive":true,"routeName":"RedisList"}`)
				tx.Model(resourceModel).Where("id = ?", 79).Update("meta", `{"icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"Mongo"}`)
				tx.Model(resourceModel).Where("id = ?", 80).Update("meta", `{"component":"ops/mongo/MongoDataOp","icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"MongoDataOp"}`)
				tx.Model(resourceModel).Where("id = ?", 82).Update("meta", `{"component":"ops/mongo/MongoList","icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"MongoList"}`)
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}

func V1_9_4() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250213-v1.9.4-addMsg",
			Migrate: func(tx *gorm.DB) error {
				tx.AutoMigrate(&sysentity.Account{})
				tx.AutoMigrate(&msgentity.MsgTmpl{}, &msgentity.MsgTmplChannel{}, &msgentity.MsgChannel{}, &msgentity.MsgTmplBiz{})

				la := &model.LoginAccount{Id: 1, Username: "admin"}
				// 创建审批默认消息模板
				processMsgTmplCode := "7u2MRCaB"
				if err := tx.Where("code = ?", processMsgTmplCode).First(&msgentity.MsgTmpl{}).Error; err != nil {
					tmplRemark := "工单审批通知模板"
					msgTmpl := &msgentity.MsgTmpl{
						Code: processMsgTmplCode,
						Name: "工单审批通知",
						Tmpl: `{{.receiver}}
您有新的工单需要审批
发起人：{{.creator}}
工单标题：{{.procdefName}}
备注：{{.procinstRemark}}
业务编号：{{.bizKey}}`,
						Title:   "工单审批",
						MsgType: 1,
						Status:  1,
						Remark:  &tmplRemark,
					}
					msgTmpl.FillBaseInfo(model.IdGenTypeNone, la)
					if err := tx.Create(msgTmpl).Error; err != nil {
						logx.ErrorTrace("create msg tmpl error", err)
						return err
					}
				}

				resources := []*sysentity.Resource{
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742816076}}}},
						Pid:    0,
						UiPath: "ckg5ICnd/",
						Name:   "menu.msgManage",
						Code:   "/msg",
						Type:   1,
						Meta:   `{"icon":"Message","isKeepAlive":true,"routeName":"msg"}`,
						Weight: 60000000,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742816279}}}},
						Pid:    1742816076,
						UiPath: "ckg5ICnd/eKQ8qAlH/",
						Name:   "menu.channel",
						Code:   "channels",
						Type:   1,
						Meta:   `{"component":"msg/channel/ChannelList","icon":"Message","isKeepAlive":true,"routeName":"ChannelList"}`,
						Weight: 1742816279,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742876893}}}},
						Pid:    1742816279,
						UiPath: "ckg5ICnd/eKQ8qAlH/p2Xi8asv/",
						Name:   "menu.msgChannelBase",
						Code:   "msg:channel:base",
						Type:   2,
						Meta:   ``,
						Weight: 1742823660,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742823661}}}},
						Pid:    1742816279,
						UiPath: "ckg5ICnd/eKQ8qAlH/Iu82rFKW/",
						Name:   "menu.saveMsgChannel",
						Code:   "msg:channel:save",
						Type:   2,
						Meta:   ``,
						Weight: 1742823661,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742826138}}}},
						Pid:    1742816279,
						UiPath: "ckg5ICnd/eKQ8qAlH/Y4kRzNJp/",
						Name:   "menu.delMsgChannel",
						Code:   "msg:channel:del",
						Type:   2,
						Meta:   ``,
						Weight: 1742826138,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742876469}}}},
						Pid:    1742816076,
						UiPath: "ckg5ICnd/XiJf38uW/",
						Name:   "menu.msgTmpl",
						Code:   "tmpls",
						Type:   1,
						Meta:   `{"component":"msg/tmpl/TmplList","icon":"Message","isKeepAlive":true,"routeName":"TmplList"}`,
						Weight: 1742876469,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742876795}}}},
						Pid:    1742876469,
						UiPath: "ckg5ICnd/XiJf38uW/ExV9tz2l/",
						Name:   "menu.saveMsgTmpl",
						Code:   "msg:tmpl:save",
						Type:   2,
						Meta:   ``,
						Weight: 1742876795,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742876813}}}},
						Pid:    1742876469,
						UiPath: "ckg5ICnd/XiJf38uW/2y7drhga/",
						Name:   "menu.delMsgTmpl",
						Code:   "msg:tmpl:del",
						Type:   2,
						Meta:   ``,
						Weight: 1742876813,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742876922}}}},
						Pid:    1742876469,
						UiPath: "ckg5ICnd/XiJf38uW/VRX9YtM3/",
						Name:   "menu.msgTmplBase",
						Code:   "msg:tmpl:base",
						Type:   2,
						Meta:   ``,
						Weight: 1742876794,
					},
					{
						Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1742912893}}}},
						Pid:    1742876469,
						UiPath: "ckg5ICnd/XiJf38uW/42PkAmLB/",
						Name:   "menu.sendMsg",
						Code:   "msg:tmpl:send",
						Type:   2,
						Meta:   ``,
						Weight: 1742912893,
					},
				}

				now := time.Now()
				for _, r := range resources {
					if err := tx.Where("ui_path = ?", r.UiPath).First(&sysentity.Resource{}).Error; err == nil {
						continue
					}
					r.Status = 1
					r.CreateTime = &now
					r.UpdateTime = &now
					r.Creator = la.Username
					r.Modifier = la.Username
					r.CreatorId = la.Id
					r.ModifierId = la.Id
					if err := tx.Create(r).Error; err != nil {
						logx.ErrorTrace("create msg resource menu error", err)
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
