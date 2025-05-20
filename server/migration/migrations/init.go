package migrations

import (
	authentity "mayfly-go/internal/auth/domain/entity"
	dbentity "mayfly-go/internal/db/domain/entity"
	fileentity "mayfly-go/internal/file/domain/entity"
	flowentity "mayfly-go/internal/flow/domain/entity"
	machineentity "mayfly-go/internal/machine/domain/entity"
	mongoentity "mayfly-go/internal/mongo/domain/entity"
	msgentity "mayfly-go/internal/msg/domain/entity"
	redisentity "mayfly-go/internal/redis/domain/entity"
	sysentity "mayfly-go/internal/sys/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func Init() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250212-v1.9.2-init",
			Migrate: func(tx *gorm.DB) error {
				entities := [...]any{
					new(sysentity.Account),
					new(sysentity.Config),
					new(sysentity.SysLog),
					new(sysentity.Role),
					new(sysentity.AccountRole),
					new(sysentity.RoleResource),
					new(sysentity.Resource),

					new(authentity.Oauth2Account),

					new(fileentity.File),

					new(msgentity.Msg),

					new(tagentity.TagTree),          // 标签树
					new(tagentity.Team),             // 团队信息
					new(tagentity.TeamMember),       // 团队成员
					new(tagentity.ResourceAuthCert), // 资源授权凭证
					new(tagentity.TagTreeRelate),    // 与标签树有关联关系的表
					new(tagentity.ResourceOpLog),    // 资源操作记录

					new(flowentity.Procdef),
					new(flowentity.Procinst),
					new(flowentity.ProcinstTask),

					new(machineentity.Machine),
					new(machineentity.MachineFile),
					new(machineentity.MachineTermOp),
					new(machineentity.MachineScript),
					new(machineentity.MachineCronJob),
					new(machineentity.MachineCronJobExec),
					new(machineentity.MachineCmdConf),

					new(dbentity.DbInstance),
					new(dbentity.Db),
					new(dbentity.DbSql),
					new(dbentity.DbSqlExec),
					new(dbentity.DataSyncTask),
					new(dbentity.DataSyncLog),
					new(dbentity.DbTransferTask),
					new(dbentity.DbTransferFile),

					new(mongoentity.Mongo),

					new(redisentity.Redis),
				}

				for _, e := range entities {
					if err := tx.AutoMigrate(e); err != nil {
						return err
					}
				}

				// 如果存在账号数据，则不进行初始化系统数据
				var count int64
				tx.Model(&sysentity.Account{}).Count(&count)
				if count > 0 {
					return nil
				}

				// 初始化管理员账号
				if err := initAccount(tx); err != nil {
					return err
				}

				if err := initRole(tx); err != nil {
					return err
				}

				if err := initSysConfig(tx); err != nil {
					return err
				}

				// 初始化菜单权限资源
				if err := initResource(tx); err != nil {
					return err
				}

				if err := initTag(tx); err != nil {
					return err
				}

				if err := initMachine(tx); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			}},
	}
}

func initAccount(tx *gorm.DB) error {
	account := &sysentity.Account{
		Username: "admin",
		Name:     "管理员",
		Password: "$2a$10$w3Wky2U.tinvR7c/s0aKPuwZsIu6pM1/DMJalwBDMbE6niHIxVrrm", // admin123.
		Status:   1,
	}

	account.Id = 1

	now := time.Now()
	account.CreateTime = &now
	account.UpdateTime = &now
	account.CreatorId = 1
	account.ModifierId = 1
	account.Creator = "admin"
	account.Modifier = "admin"

	return tx.Create(account).Error
}

func initRole(tx *gorm.DB) error {
	role := &sysentity.Role{
		Name:   "公共角色",
		Code:   "COMMON",
		Status: 1,
		Remark: "所有账号基础角色",
	}

	role.Id = 1

	now := time.Now()
	role.CreateTime = &now
	role.UpdateTime = &now
	role.CreatorId = 1
	role.ModifierId = 1
	role.Creator = "admin"
	role.Modifier = "admin"

	roleResource := &sysentity.RoleResource{
		RoleId:     role.Id,
		ResourceId: 1,
		CreateTime: &now,
		CreatorId:  1,
		Creator:    "admin",
	}

	tx.Create(roleResource)
	return tx.Create(role).Error
}

func initSysConfig(tx *gorm.DB) error {
	configs := []*sysentity.Config{
		{
			Name:       "system.sysconf.accountLoginConf",
			Key:        "AccountLoginSecurity",
			Params:     `[{"name":"system.sysconf.useCaptcha","model":"useCaptcha","placeholder":"system.sysconf.useCaptchaPlaceholder","options":"true,false"},{"name":"system.sysconf.useOtp","model":"useOtp","placeholder":"system.sysconf.useOtpPlaceholder","options":"true,false"},{"name":"system.sysconf.otpIssuer","model":"otpIssuer","placeholder":""},{"name":"system.sysconf.loginFailCount","model":"loginFailCount","placeholder":"system.sysconf.loginFailCountPlaceholder"},{"name":"system.sysconf.loginFainMin","model":"loginFailMin","placeholder":"system.sysconf.loginFailMinPlaceholder"}]`,
			Value:      `{"useCaptcha":"true","useOtp":"false","loginFailCount":"5","loginFailMin":"10","otpIssuer":"mayfly-go"}`,
			Remark:     "system.sysconf.accountLoginConfRemark",
			Permission: "all",
		},
		{
			Name:       "system.sysconf.oauth2LoginConf",
			Key:        "Oauth2Login",
			Params:     `[{"name":"system.sysconf.oauth2Enable","model":"enable","placeholder":"system.sysconf.oauth2EnablePlaceholder","options":"true,false"},{"name":"system.sysconf.name","model":"name","placeholder":"system.sysconf.namePlaceholder"},{"name":"system.sysconf.clientId","model":"clientId","placeholder":"system.sysconf.clientIdPlaceholder"},{"name":"system.sysconf.clientSecret","model":"clientSecret","placeholder":"system.sysconf.clientSecretPlaceholder"},{"name":"system.sysconf.authorizationUrl","model":"authorizationURL","placeholder":"system.sysconf.authorizationUrlPlaceholder"},{"name":"system.sysconf.accessTokenUrl","model":"accessTokenURL","placeholder":"system.sysconf.accessTokenUrlPlaceholder"},{"name":"system.sysconf.redirectUrl","model":"redirectURL","placeholder":"system.sysconf.redirectUrlPlaceholder"},{"name":"system.sysconf.scope","model":"scopes","placeholder":"system.sysconf.scopePlaceholder"},{"name":"system.sysconf.resourceUrl","model":"resourceURL","placeholder":"system.sysconf.resourceUrlPlaceholder"},{"name":"system.sysconf.userId","model":"userIdentifier","placeholder":"system.sysconf.userIdPlaceholder"},{"name":"system.sysconf.autoRegister","model":"autoRegister","placeholder":"","options":"true,false"}]`,
			Value:      ``,
			Remark:     "system.sysconf.oauth2LoginConfRemark",
			Permission: "admin,",
		},
		{
			Name:       "system.sysconf.ldapLoginConf",
			Key:        "LdapLogin",
			Params:     `[{"name":"system.sysconf.ldapEnable","model":"enable","placeholder":"system.sysconf.dapEnablePlaceholder","options":"true,false"},{"name":"system.sysconf.host","model":"host","placeholder":"system.sysconf.host"},{"name":"system.sysconf.port","model":"port","placeholder":"system.sysconf.port"},{"name":"system.sysconf.bindDN","model":"bindDN","placeholder":"system.sysconf.bindDnPlaceholder"},{"name":"system.sysconf.bindPwd","model":"bindPwd","placeholder":"system.sysconf.bindPwdPlaceholder"},{"name":"system.sysconf.baseDN","model":"baseDN","placeholder":"system.sysconf.baseDnPlaceholder"},{"name":"system.sysconf.userFilter","model":"userFilter","placeholder":"system.sysconf.userFilerPlaceholder"},{"name":"system.sysconf.uidMap","model":"uidMap","placeholder":"system.sysconf.uidMapPlaceholder"},{"name":"system.sysconf.udnMap","model":"udnMap","placeholder":"system.sysconf.udnMapPlaceholder"},{"name":"system.sysconf.emailMap","model":"emailMap","placeholder":"system.sysconf.emailMapPlaceholder"},{"name":"system.sysconf.skipTlsVerfify","model":"skipTLSVerify","placeholder":"system.sysconf.skipTlsVerfifyPlaceholder","options":"true,false"},{"name":"system.sysconf.securityProtocol","model":"securityProtocol","placeholder":"system.sysconf.securityProtocolPlaceholder","options":"Null,StartTLS,LDAPS"}]`,
			Value:      ``,
			Remark:     "system.sysconf.ldapLoginConfRemark",
			Permission: "admin,",
		},
		{
			Name:       "system.sysconf.systemConf",
			Key:        "SysStyleConfig",
			Params:     `[{"model":"logoIcon","name":"system.sysconf.logoIcon","placeholder":"system.sysconf.logoIconPlaceholder","required":false},{"model":"title","name":"system.sysconf.title","placeholder":"system.sysconf.titlePlaceholder","required":false},{"model":"viceTitle","name":"system.sysconf.viceTitle","placeholder":"system.sysconf.viceTitlePlaceholder","required":false},{"model":"useWatermark","name":"system.sysconf.useWatermark","placeholder":"system.sysconf.useWatermarkPlaceholder","options":"true,false","required":false},{"model":"watermarkContent","name":"system.sysconf.watermarkContent","placeholder":"system.sysconf.watermarkContentPlaceholder","required":false}]`,
			Value:      `{"title":"mayfly-go","viceTitle":"mayfly-go","logoIcon":"","useWatermark":"true","watermarkContent":""}`,
			Remark:     "system.sysconf.systemConfRemark",
			Permission: "all",
		},
		{
			Name:       "system.sysconf.machineConf",
			Key:        "MachineConfig",
			Params:     `[{"name":"system.sysconf.uploadMaxFileSize","model":"uploadMaxFileSize","placeholder":"system.sysconf.uploadMaxFileSizePlaceholder"},{"model":"termOpSaveDays","name":"system.sysconf.termOpSaveDays","placeholder":"system.sysconf.termOpSaveDaysPlaceholder"},{"model":"guacdHost","name":"system.sysconf.guacdHost","placeholder":"system.sysconf.guacdHostPlaceholder","required":false},{"name":"system.sysconf.guacdPort","model":"guacdPort","placeholder":"system.sysconf.guacdPortPlaceholder","required":false},{"model":"guacdFilePath","name":"system.sysconf.guacdFilePath","placeholder":"system.sysconf.guacdFilePathPlaceholder"}]`,
			Value:      `{"uploadMaxFileSize":"1000MB","termOpSaveDays":"30","guacdHost":"","guacdPort":"","guacdFilePath":"./guacd/rdp-file"}`,
			Remark:     "system.sysconf.machineConfRemark",
			Permission: "all",
		},
		{
			Name:       "system.sysconf.dbmsConf",
			Key:        "DbmsConfig",
			Params:     `[{"model":"querySqlSave","name":"system.sysconf.recordQuerySql","placeholder":"system.sysconf.recordQuerySqlPlaceholder","options":"true,false"},{"model":"maxResultSet","name":"system.sysconf.maxResultSet","placeholder":"system.sysconf.maxResultSetPlaceholder","options":""},{"model":"sqlExecTl","name":"system.sysconf.sqlExecLimt","placeholder":"system.sysconf.sqlExecLimtPlaceholder"}]`,
			Value:      `{"querySqlSave":"false","maxResultSet":"0","sqlExecTl":"60"}`,
			Remark:     "system.sysconf.dbmsConfRemark",
			Permission: "admin,",
		},
		{
			Name:       "system.sysconf.fileConf",
			Key:        "FileConfig",
			Params:     `[{"model":"basePath","name":"system.sysconf.basePath","placeholder":"system.sysconf.baesPathPlaceholder"}]`,
			Value:      `{"basePath":"./file"}`,
			Remark:     "system.sysconf.fileConfRemark",
			Permission: "admin,",
		},
	}

	now := time.Now()
	for _, res := range configs {
		res.CreateTime = &now
		res.CreatorId = 1
		res.Creator = "admin"
		res.UpdateTime = &now
		res.ModifierId = 1
		res.Modifier = "admin"
		if err := tx.Create(res).Error; err != nil {
			return err
		}
	}
	return nil
}

func initTag(tx *gorm.DB) error {
	now := time.Now()

	tag := &tagentity.TagTree{
		Name:     "默认",
		Code:     "default",
		CodePath: "default/",
		Type:     -1,
		Remark:   "默认标签",
	}

	tag.Id = 1
	tag.CreateTime = &now
	tag.UpdateTime = &now
	tag.CreatorId = 1
	tag.ModifierId = 1
	tag.Creator = "admin"
	tag.Modifier = "admin"

	end := now.AddDate(20, 0, 0)
	team := &tagentity.Team{
		Name:              "default_team",
		ValidityStartDate: &now,
		ValidityEndDate:   &end,
		Remark:            "默认团队",
	}
	team.Id = 1
	team.CreateTime = &now
	team.UpdateTime = &now
	team.CreatorId = 1
	team.ModifierId = 1
	team.Creator = "admin"
	team.Modifier = "admin"

	teamMember := &tagentity.TeamMember{
		TeamId:    1,
		AccountId: 1,
		Username:  "admin",
	}
	teamMember.CreateTime = &now
	teamMember.UpdateTime = &now
	teamMember.CreatorId = 1
	teamMember.ModifierId = 1
	teamMember.Creator = "admin"
	teamMember.Modifier = "admin"

	tagRelate := &tagentity.TagTreeRelate{
		TagId:      1,
		RelateId:   1,
		RelateType: 1,
	}
	tagRelate.CreateTime = &now
	tagRelate.UpdateTime = &now
	tagRelate.CreatorId = 1
	tagRelate.ModifierId = 1
	tagRelate.Creator = "admin"
	tagRelate.Modifier = "admin"

	tx.Create(team)
	tx.Create(teamMember)
	tx.Create(tagRelate)
	return tx.Create(tag).Error
}

func initMachine(tx *gorm.DB) error {
	machineScripts := []*machineentity.MachineScript{
		{
			Name:      "disk-mem",
			Script:    `df -h`,
			Type:      1,
			MachineId: 9999999,
		},
		{
			Name:      "test_params",
			Script:    `echo {{.processName}}`,
			Type:      1,
			Params:    `[{\"name\": \"pname\",\"model\": \"processName\", \"placeholder\": \"enter processName\"}]`,
			MachineId: 9999999,
		},
		{
			Name:      "top",
			Script:    `top`,
			Type:      3,
			MachineId: 9999999,
		},
	}

	now := time.Now()
	for _, mc := range machineScripts {
		mc.CreateTime = &now
		mc.CreatorId = 1
		mc.Creator = "admin"
		mc.UpdateTime = &now
		mc.ModifierId = 1
		mc.Modifier = "admin"
		if err := tx.Create(mc).Error; err != nil {
			return err
		}
	}

	return nil
}

func initResource(tx *gorm.DB) error {
	resources := []*sysentity.Resource{
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1}}}},
			Pid:    0,
			UiPath: "Aexqq77l/",
			Name:   "menu.index",
			Code:   "/home",
			Type:   1,
			Meta:   `{"component":"home/Home","icon":"HomeFilled","isAffix":true,"routeName":"Home"}`,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 2}}}},
			Pid:    0,
			UiPath: "12sSjal1/",
			Name:   "menu.machine",
			Code:   "/machine",
			Type:   1,
			Meta:   `{"icon":"Monitor","isKeepAlive":true,"redirect":"machine/list","routeName":"Machine"}`,
			Weight: 49999998,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 3}}}},
			Pid:    2,
			UiPath: "12sSjal1/lskeiql1/",
			Name:   "menu.machineList",
			Code:   "machines",
			Type:   1,
			Meta:   `{"component":"ops/machine/MachineList","icon":"Monitor","isKeepAlive":true,"routeName":"MachineList"}`,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 4}}}},
			Pid:    0,
			UiPath: "Xlqig32x/",
			Name:   "menu.system",
			Code:   "/sys",
			Type:   1,
			Meta:   `{"icon":"Setting","isKeepAlive":true,"redirect":"/sys/resources","routeName":"sys"}`,
			Weight: 60000001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 5}}}},
			Pid:    4,
			UiPath: "Xlqig32x/UGxla231/",
			Name:   "menu.menuPermission",
			Code:   "resources",
			Type:   1,
			Meta:   `{"component":"system/resource/ResourceList","icon":"Menu","isKeepAlive":true,"routeName":"ResourceList"}`,
			Weight: 9999998,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 11}}}},
			Pid:    4,
			UiPath: "Xlqig32x/lxqSiae1/",
			Name:   "menu.role",
			Code:   "roles",
			Type:   1,
			Meta:   `{"component":"system/role/RoleList","icon":"icon menu/role","isKeepAlive":true,"routeName":"RoleList"}`,
			Weight: 10000001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 12}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Alw1Xkq3/",
			Name:   "menu.machineTerminal",
			Code:   "machine:terminal",
			Type:   2,
			Weight: 40000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 14}}}},
			Pid:    4,
			UiPath: "Xlqig32x/sfslfel/",
			Name:   "menu.account",
			Code:   "accounts",
			Type:   1,
			Meta:   `{"component":"system/account/AccountList","icon":"User","isKeepAlive":true,"routeName":"AccountList"}`,
			Weight: 9999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 15}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Lsew24Kx/",
			Name:   "menu.machineFileConf",
			Code:   "machine:file",
			Type:   2,
			Weight: 50000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 16}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/exIsqL31/",
			Name:   "menu.machineCreate",
			Code:   "machine:add",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 17}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Liwakg2x/",
			Name:   "menu.machineEdit",
			Code:   "machine:update",
			Type:   2,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 18}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Lieakenx/",
			Name:   "menu.machineDelete",
			Code:   "machine:del",
			Type:   2,
			Weight: 30000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 19}}}},
			Pid:    14,
			UiPath: "Xlqig32x/sfslfel/UUiex2xA/",
			Name:   "menu.accountRoleAllocation",
			Code:   "account:saveRoles",
			Type:   2,
			Weight: 50000001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 20}}}},
			Pid:    11,
			UiPath: "Xlqig32x/lxqSiae1/EMq2Kxq3/",
			Name:   "menu.roleMenuPermissionAllocation",
			Code:   "role:saveResources",
			Type:   2,
			Weight: 40000002,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 21}}}},
			Pid:    14,
			UiPath: "Xlqig32x/sfslfel/Uexax2xA/",
			Name:   "menu.accountDelete",
			Code:   "account:del",
			Type:   2,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 22}}}},
			Pid:    11,
			UiPath: "Xlqig32x/lxqSiae1/Elxq2Kxq3/",
			Name:   "menu.roleDelete",
			Code:   "role:del",
			Type:   2,
			Weight: 40000001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 23}}}},
			Pid:    11,
			UiPath: "Xlqig32x/lxqSiae1/342xKxq3/",
			Name:   "menu.roleAdd",
			Code:   "role:add",
			Type:   2,
			Weight: 19999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 24}}}},
			Pid:    11,
			UiPath: "Xlqig32x/lxqSiae1/LexqKxq3/",
			Name:   "menu.roleEdit",
			Code:   "role:update",
			Type:   2,
			Weight: 40000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 25}}}},
			Pid:    5,
			UiPath: "Xlqig32x/UGxla231/Elxq23XK/",
			Name:   "menu.menuPermissionAdd",
			Code:   "resource:add",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 26}}}},
			Pid:    5,
			UiPath: "Xlqig32x/UGxla231/eloq23XK/",
			Name:   "menu.menuPermissionDelete",
			Code:   "resource:delete",
			Type:   2,
			Weight: 30000001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 27}}}},
			Pid:    5,
			UiPath: "Xlqig32x/UGxla231/JExq23XK/",
			Name:   "menu.menuPermissionEdit",
			Code:   "resource:update",
			Type:   2,
			Weight: 30000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 28}}}},
			Pid:    5,
			UiPath: "Xlqig32x/UGxla231/Elex13XK/",
			Name:   "menu.menuPermissionEnableDisable",
			Code:   "resource:changeStatus",
			Type:   2,
			Weight: 40000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 29}}}},
			Pid:    14,
			UiPath: "Xlqig32x/sfslfel/xlawx2xA/",
			Name:   "menu.accountAdd",
			Code:   "account:add",
			Type:   2,
			Weight: 19999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 31}}}},
			Pid:    14,
			UiPath: "Xlqig32x/sfslfel/eubale13/",
			Name:   "menu.accountBase",
			Code:   "account",
			Type:   2,
			Weight: 9999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 32}}}},
			Pid:    5,
			UiPath: "Xlqig32x/UGxla231/321q23XK/",
			Name:   "menu.menuPermissionBase",
			Code:   "resource",
			Type:   2,
			Weight: 9999999,
		},

		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 33}}}},
			Pid:    11,
			UiPath: "Xlqig32x/lxqSiae1/908xKxq3/",
			Name:   "menu.roleBase",
			Code:   "role",
			Type:   2,
			Weight: 9999999,
		},

		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 34}}}},
			Pid:    14,
			UiPath: "Xlqig32x/sfslfel/32alx2xA/",
			Name:   "menu.accountEnableDisable",
			Code:   "account:changeStatus",
			Type:   2,
			Weight: 50000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 36}}}},
			Pid:    0,
			UiPath: "dbms23ax/",
			Name:   "menu.dbms",
			Code:   "/dbms",
			Type:   1,
			Meta:   `{"icon":"Coin","isKeepAlive":true,"routeName":"DBMS"}`,
			Weight: 49999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 37}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Keiqkx4L/",
			Name:   "menu.machineFileConfCreate",
			Code:   "machine:addFile",
			Type:   2,
			Weight: 60000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 38}}}},
			Pid:    36,
			UiPath: "dbms23ax/exaeca2x/",
			Name:   "menu.dbDataOp",
			Code:   "sql-exec",
			Type:   1,
			Meta:   `{"component":"ops/db/SqlExec","icon":"Coin","isKeepAlive":true,"routeName":"SqlExec"}`,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 39}}}},
			Pid:    0,
			UiPath: "sl3as23x/",
			Name:   "menu.personalCenter",
			Code:   "/personal",
			Type:   1,
			Meta:   `{"component":"personal/index","icon":"UserFilled","isHide":true,"isKeepAlive":true,"routeName":"Personal"}`,
			Weight: 19999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 40}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Keal2Xke/",
			Name:   "menu.machineFileCreate",
			Code:   "machine:file:add",
			Type:   2,
			Weight: 70000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 41}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Ihfs2xaw/",
			Name:   "menu.machineFileDelete",
			Code:   "machine:file:del",
			Type:   2,
			Weight: 80000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 42}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/3ldkxJDx/",
			Name:   "menu.machineFileWrite",
			Code:   "machine:file:write",
			Type:   2,
			Weight: 90000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 43}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Ljewix43/",
			Name:   "menu.machineFileUpload",
			Code:   "machine:file:upload",
			Type:   2,
			Weight: 100000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 44}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/L12wix43/",
			Name:   "menu.machineFileConfDelete",
			Code:   "machine:file:rm",
			Type:   2,
			Weight: 69999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 45}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Ljewisd3/",
			Name:   "menu.machineScriptSave",
			Code:   "machine:script:save",
			Type:   2,
			Weight: 120000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 46}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/Ljeew43/",
			Name:   "menu.machineScriptDelete",
			Code:   "machine:script:del",
			Type:   2,
			Weight: 130000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 47}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/ODewix43/",
			Name:   "menu.machineScriptRun",
			Code:   "machine:script:run",
			Type:   2,
			Weight: 140000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 54}}}},
			Pid:    135,
			UiPath: "dbms23ax/X0f4BxT0/leix3Axl/",
			Name:   "menu.dbSave",
			Code:   "db:save",
			Type:   2,
			Weight: 1693041086,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 55}}}},
			Pid:    135,
			UiPath: "dbms23ax/X0f4BxT0/ygjL3sxA/",
			Name:   "menu.dbDelete",
			Code:   "db:del",
			Type:   2,
			Weight: 1693041086,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 57}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/OJewex43/",
			Name:   "menu.machineBase",
			Code:   "machine",
			Type:   2,
			Weight: 9999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 58}}}},
			Pid:    135,
			UiPath: "dbms23ax/X0f4BxT0/AceXe321/",
			Name:   "menu.dbBase",
			Code:   "db",
			Type:   2,
			Weight: 1693041085,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 59}}}},
			Pid:    38,
			UiPath: "dbms23ax/exaeca2x/ealcia23/",
			Name:   "menu.dbDataOpBase",
			Code:   "db:exec",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 60}}}},
			Pid:    0,
			UiPath: "RedisXq4/",
			Name:   "menu.redis",
			Code:   "/redis",
			Type:   1,
			Meta:   `{"icon":"icon redis/redis","isKeepAlive":true,"routeName":"RDS"}`,
			Weight: 50000001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 61}}}},
			Pid:    60,
			UiPath: "RedisXq4/Exitx4al/",
			Name:   "menu.redisDataOp",
			Code:   "data-operation",
			Type:   1,
			Meta:   `{"component":"ops/redis/DataOperation","icon":"icon redis/redis","isKeepAlive":true,"routeName":"DataOperation"}`,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 62}}}},
			Pid:    61,
			UiPath: "RedisXq4/Exitx4al/LSjie321/",
			Name:   "menu.redisDataOpBase",
			Code:   "redis:data",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 63}}}},
			Pid:    60,
			UiPath: "RedisXq4/Eoaljc12/",
			Name:   "menu.redisManage",
			Code:   "manage",
			Type:   1,
			Meta:   `{"component":"ops/redis/RedisList","icon":"icon redis/redis","isKeepAlive":true,"routeName":"RedisList"}`,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 64}}}},
			Pid:    63,
			UiPath: "RedisXq4/Eoaljc12/IoxqAd31/",
			Name:   "menu.redisManageBase",
			Code:   "redis:manage",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 71}}}},
			Pid:    61,
			UiPath: "RedisXq4/Exitx4al/IUlxia23/",
			Name:   "menu.redisDataOpSave",
			Code:   "redis:data:save",
			Type:   2,
			Weight: 29999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 72}}}},
			Pid:    3,
			UiPath: "12sSjal1/lskeiql1/LIEwix43/",
			Name:   "menu.machineKillprocess",
			Code:   "machine:killprocess",
			Type:   2,
			Weight: 49999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 79}}}},
			Pid:    0,
			UiPath: "Mongo452/",
			Name:   "menu.mongo",
			Code:   "/mongo",
			Type:   1,
			Meta:   `{"icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"Mongo"}`,
			Weight: 50000002,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 80}}}},
			Pid:    79,
			UiPath: "Mongo452/eggago31/",
			Name:   "menu.mongoDataOp",
			Code:   "mongo-data-operation",
			Type:   1,
			Meta:   `{"component":"ops/mongo/MongoDataOp","icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"MongoDataOp"}`,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 81}}}},
			Pid:    80,
			UiPath: "Mongo452/eggago31/egjglal3/",
			Name:   "menu.mongoDataOpBase",
			Code:   "mongo:base",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 82}}}},
			Pid:    79,
			UiPath: "Mongo452/ghxagl43/",
			Name:   "menu.mongoManage",
			Code:   "mongo-manage",
			Type:   1,
			Meta:   `{"component":"ops/mongo/MongoList","icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"MongoList"}`,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 83}}}},
			Pid:    82,
			UiPath: "Mongo452/ghxagl43/egljbla3/",
			Name:   "menu.mongoManageBase",
			Code:   "mongo:manage:base",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 84}}}},
			Pid:    4,
			UiPath: "Xlqig32x/exlaeAlx/",
			Name:   "menu.opLog",
			Code:   "syslogs",
			Type:   1,
			Meta:   `{"component":"system/syslog/SyslogList","icon":"Tickets","routeName":"SyslogList"}`,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 85}}}},
			Pid:    84,
			UiPath: "Xlqig32x/exlaeAlx/3xlqeXql/",
			Name:   "menu.opLogBase",
			Code:   "syslog",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 87}}}},
			Pid:    4,
			UiPath: "Xlqig32x/Ulxaee23/",
			Name:   "menu.sysConf",
			Code:   "configs",
			Type:   1,
			Meta:   `{"component":"system/config/ConfigList","icon":"Setting","isKeepAlive":true,"routeName":"ConfigList"}`,
			Weight: 10000002,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 88}}}},
			Pid:    87,
			UiPath: "Xlqig32x/Ulxaee23/exlqguA3/",
			Name:   "menu.sysConfBase",
			Code:   "config:base",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 93}}}},
			Pid:    0,
			UiPath: "Tag3fhad/",
			Name:   "menu.tag",
			Code:   "/tag",
			Type:   1,
			Meta:   `{"icon":"CollectionTag","isKeepAlive":true,"routeName":"Tag"}`,
			Weight: 20000001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 94}}}},
			Pid:    93,
			UiPath: "Tag3fhad/glxajg23/",
			Name:   "menu.tagTree",
			Code:   "tag-trees",
			Type:   1,
			Meta:   `{"component":"ops/tag/TagTreeList","icon":"CollectionTag","isKeepAlive":true,"routeName":"TagTreeList"}`,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 95}}}},
			Pid:    93,
			UiPath: "Tag3fhad/Bjlag32x/",
			Name:   "menu.team",
			Code:   "teams",
			Type:   1,
			Meta:   `{"component":"ops/tag/TeamList","icon":"UserFilled","isKeepAlive":true,"routeName":"TeamList"}`,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 96}}}},
			Pid:    94,
			UiPath: "Tag3fhad/glxajg23/gkxagt23/",
			Name:   "menu.tagSave",
			Code:   "tag:save",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 97}}}},
			Pid:    95,
			UiPath: "Tag3fhad/Bjlag32x/GJslag32/",
			Name:   "menu.teamSave",
			Code:   "team:save",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 98}}}},
			Pid:    94,
			UiPath: "Tag3fhad/glxajg23/xjgalte2/",
			Name:   "menu.tagDelete",
			Code:   "tag:del",
			Type:   2,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 99}}}},
			Pid:    95,
			UiPath: "Tag3fhad/Bjlag32x/Gguca23x/",
			Name:   "menu.teamDelete",
			Code:   "team:del",
			Type:   2,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 100}}}},
			Pid:    95,
			UiPath: "Tag3fhad/Bjlag32x/Lgidsq32/",
			Name:   "menu.teamMemberAdd",
			Code:   "team:member:save",
			Type:   2,
			Weight: 30000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 101}}}},
			Pid:    95,
			UiPath: "Tag3fhad/Bjlag32x/Lixaue3G/",
			Name:   "menu.teamMemberDelete",
			Code:   "team:member:del",
			Type:   2,
			Weight: 40000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 102}}}},
			Pid:    95,
			UiPath: "Tag3fhad/Bjlag32x/Oygsq3xg/",
			Name:   "menu.teamTagSave",
			Code:   "team:tag:save",
			Type:   2,
			Weight: 50000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 103}}}},
			Pid:    93,
			UiPath: "Tag3fhad/exahgl32/",
			Name:   "menu.authorization",
			Code:   "authcerts",
			Type:   1,
			Meta:   `{"component":"ops/tag/AuthCertList","icon":"Ticket","isKeepAlive":true,"routeName":"AuthCertList"}`,
			Weight: 19999999,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 104}}}},
			Pid:    103,
			UiPath: "Tag3fhad/exahgl32/egxahg24/",
			Name:   "menu.authorizationBase",
			Code:   "authcert",
			Type:   2,
			Weight: 10000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 105}}}},
			Pid:    103,
			UiPath: "Tag3fhad/exahgl32/yglxahg2/",
			Name:   "menu.authorizationSave",
			Code:   "authcert:save",
			Type:   2,
			Weight: 20000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 106}}}},
			Pid:    103,
			UiPath: "Tag3fhad/exahgl32/Glxag234/",
			Name:   "menu.authorizationDelete",
			Code:   "authcert:del",
			Type:   2,
			Weight: 30000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 108}}}},
			Pid:    61,
			UiPath: "RedisXq4/Exitx4al/Gxlagheg/",
			Name:   "menu.redisDataOpDelete",
			Code:   "redis:data:del",
			Type:   2,
			Weight: 30000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 128}}}},
			Pid:    87,
			UiPath: "Xlqig32x/Ulxaee23/MoOWr2N0/",
			Name:   "menu.sysConfSave",
			Code:   "config:save",
			Type:   2,
			Weight: 1687315135,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 130}}}},
			Pid:    2,
			UiPath: "12sSjal1/W9XKiabq/",
			Name:   "menu.machineCronJob",
			Code:   "/machine/cron-job",
			Type:   1,
			Meta:   `{"component":"ops/machine/cronjob/CronJobList","icon":"AlarmClock","isKeepAlive":true,"routeName":"CronJobList"}`,
			Weight: 1689646396,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 131}}}},
			Pid:    130,
			UiPath: "12sSjal1/W9XKiabq/gEOqr2pD/",
			Name:   "menu.machineCronJobSvae",
			Code:   "machine:cronjob:save",
			Type:   2,
			Weight: 1689860087,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 132}}}},
			Pid:    130,
			UiPath: "12sSjal1/W9XKiabq/zxXM23i0/",
			Name:   "menu.machineCronJobDelete",
			Code:   "machine:cronjob:del",
			Type:   2,
			Weight: 1689860102,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 133}}}},
			Pid:    80,
			UiPath: "Mongo452/eggago31/xvpKk36u/",
			Name:   "menu.mongoDataOpSave",
			Code:   "mongo:data:save",
			Type:   2,
			Weight: 1692674943,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 134}}}},
			Pid:    80,
			UiPath: "Mongo452/eggago31/3sblw1Wb/",
			Name:   "menu.mongoDataOpDelete",
			Code:   "mongo:data:del",
			Type:   2,
			Weight: 1692674964,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 135}}}},
			Pid:    36,
			UiPath: "dbms23ax/X0f4BxT0/",
			Name:   "menu.dbInstance",
			Code:   "instances",
			Type:   1,
			Meta:   `{"component":"ops/db/InstanceList","icon":"Coin","isKeepAlive":true,"routeName":"InstanceList"}`,
			Weight: 1693040706,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 136}}}},
			Pid:    135,
			UiPath: "dbms23ax/X0f4BxT0/D23fUiBr/",
			Name:   "menu.dbInstanceSave",
			Code:   "db:instance:save",
			Type:   2,
			Weight: 1693041001,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 137}}}},
			Pid:    135,
			UiPath: "dbms23ax/X0f4BxT0/mJlBeTCs/",
			Name:   "menu.dbInstanceBase",
			Code:   "db:instance",
			Type:   2,
			Weight: 1693041000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 138}}}},
			Pid:    135,
			UiPath: "dbms23ax/X0f4BxT0/Sgg8uPwz/",
			Name:   "menu.dbInstanceDelete",
			Code:   "db:instance:del",
			Type:   2,
			Weight: 1693041084,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 150}}}},
			Pid:    36,
			UiPath: "Jra0n7De/",
			Name:   "menu.dbDataSync",
			Code:   "sync",
			Type:   1,
			Meta:   `{"component":"ops/db/SyncTaskList","icon":"Refresh","isKeepAlive":true,"routeName":"SyncTaskList"}`,
			Weight: 1693040707,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 151}}}},
			Pid:    150,
			UiPath: "Jra0n7De/uAnHZxEV/",
			Name:   "menu.dbDataSync",
			Code:   "db:sync",
			Type:   2,
			Weight: 1703641202,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 152}}}},
			Pid:    150,
			UiPath: "Jra0n7De/zvAMo2vk/",
			Name:   "menu.dbDataSyncSave",
			Code:   "db:sync:save",
			Type:   2,
			Weight: 1703641320,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 153}}}},
			Pid:    150,
			UiPath: "Jra0n7De/pLOA2UYz/",
			Name:   "menu.dbDataSyncDelete",
			Code:   "db:sync:del",
			Type:   2,
			Weight: 1703641342,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 154}}}},
			Pid:    150,
			UiPath: "Jra0n7De/VBt68CDx/",
			Name:   "menu.dbDataSyncChangeStatus",
			Code:   "db:sync:status",
			Type:   2,
			Weight: 1703641364,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 155}}}},
			Pid:    150,
			UiPath: "Jra0n7De/PigmSGVg/",
			Name:   "menu.dbDataSyncLog",
			Code:   "db:sync:log",
			Type:   2,
			Weight: 1704266866,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1707206386}}}},
			Pid:    2,
			UiPath: "PDPt6217/",
			Name:   "menu.machineOp",
			Code:   "machines-op",
			Type:   1,
			Meta:   `{"component":"ops/machine/MachineOp","icon":"Monitor","isKeepAlive":true,"routeName":"MachineOp"}`,
			Weight: 1,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1707206421}}}},
			Pid:    1707206386,
			UiPath: "PDPt6217/kQXTYvuM/",
			Name:   "menu.machineOpBase",
			Code:   "machine-op",
			Type:   2,
			Weight: 1707206421,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1708910975}}}},
			Pid:    0,
			UiPath: "6egfEVYr/",
			Name:   "menu.flow",
			Code:   "/flow",
			Type:   1,
			Meta:   `{"icon":"List","isKeepAlive":true,"routeName":"flow"}`,
			Weight: 60000000,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1708911264}}}},
			Pid:    1708910975,
			UiPath: "6egfEVYr/fw0Hhvye/",
			Name:   "menu.flowProcDef",
			Code:   "procdefs",
			Type:   1,
			Meta:   `{"component":"flow/ProcdefList","icon":"List","isKeepAlive":true,"routeName":"ProcdefList"}`,
			Weight: 1708911264,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709045735}}}},
			Pid:    1708910975,
			UiPath: "6egfEVYr/3r3hHEub/",
			Name:   "menu.myTask",
			Code:   "procinst-tasks",
			Type:   1,
			Meta:   `{"component":"flow/ProcinstTaskList","icon":"Tickets","isKeepAlive":true,"routeName":"ProcinstTaskList"}`,
			Weight: 1708911263,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709103180}}}},
			Pid:    1708910975,
			UiPath: "6egfEVYr/oNCIbynR/",
			Name:   "menu.myFlow",
			Code:   "procinsts",
			Type:   1,
			Meta:   `{"component":"flow/ProcinstList","icon":"Tickets","isKeepAlive":true,"routeName":"ProcinstList"}`,
			Weight: 1708911263,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709194669}}}},
			Pid:    36,
			UiPath: "SmLcpu6c/",
			Name:   "menu.dbTransfer",
			Code:   "transfer",
			Type:   1,
			Meta:   `{"component":"ops/db/DbTransferList","icon":"Switch","isKeepAlive":true,"routeName":"DbTransferList"}`,
			Weight: 1709194669,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709194694}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/A9vAm4J8/",
			Name:   "menu.dbTransferBase",
			Code:   "db:transfer",
			Type:   2,
			Weight: 1709194694,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709196697}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/5oJwPzNb/",
			Name:   "menu.dbTransferSave",
			Code:   "db:transfer:save",
			Type:   2,
			Weight: 1709196697,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709196707}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/L3ybnAEW/",
			Name:   "menu.dbTransferDelete",
			Code:   "db:transfer:del",
			Type:   2,
			Weight: 1709196707,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709196723}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/hGiLN1VT/",
			Name:   "menu.dbTransferChangeStatus",
			Code:   "db:transfer:status",
			Type:   2,
			Weight: 1709196723,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709196737}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/CZhNIbWg/",
			Name:   "menu.dbTransferRunLog",
			Code:   "db:transfer:log",
			Type:   2,
			Weight: 1709196737,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709196755}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/b6yHt6V2/",
			Name:   "menu.dbTransferRun",
			Code:   "db:transfer:run",
			Type:   2,
			Weight: 1709196736,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709208339}}}},
			Pid:    1708911264,
			UiPath: "6egfEVYr/fw0Hhvye/r9ZMTHqC/",
			Name:   "menu.flowProcDefSave",
			Code:   "flow:procdef:save",
			Type:   2,
			Weight: 1709208339,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1709208354}}}},
			Pid:    1708911264,
			UiPath: "6egfEVYr/fw0Hhvye/b4cNf3iq/",
			Name:   "menu.flowProcDefDelete",
			Code:   "flow:procdef:del",
			Type:   2,
			Weight: 1709208354,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1712717290}}}},
			Pid:    0,
			UiPath: "tLb8TKLB/",
			Name:   "menu.noPagePermission",
			Code:   "empty",
			Type:   1,
			Meta:   `{"component":"empty","icon":"Menu","isHide":true,"isKeepAlive":true,"routeName":"empty"}`,
			Weight: 60000002,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1712717337}}}},
			Pid:    1712717290,
			UiPath: "tLb8TKLB/m2abQkA8/",
			Name:   "menu.authcertShowciphertext",
			Code:   "authcert:showciphertext",
			Type:   2,
			Weight: 1712717337,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1713875842}}}},
			Pid:    2,
			UiPath: "12sSjal1/UnWIUhW0/",
			Name:   "menu.machineSecurityConfig",
			Code:   "security",
			Type:   1,
			Meta:   `{"component":"ops/machine/security/SecurityConfList","icon":"Setting","isKeepAlive":true,"routeName":"SecurityConfList"}`,
			Weight: 1713875842,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1714031981}}}},
			Pid:    1713875842,
			UiPath: "12sSjal1/UnWIUhW0/tEzIKecl/",
			Name:   "menu.machineSecurityCmdSvae",
			Code:   "cmdconf:save",
			Type:   2,
			Weight: 1714031981,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1714032002}}}},
			Pid:    1713875842,
			UiPath: "12sSjal1/UnWIUhW0/0tJwC3Gf/",
			Name:   "menu.machineSecurityCmdDelete",
			Code:   "cmdconf:del",
			Type:   2,
			Weight: 1714032002,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1724376022}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/HIURtJJA/",
			Name:   "menu.dbTransferFileDelete",
			Code:   "db:transfer:files:del",
			Type:   2,
			Weight: 1724376022,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1724395850}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/FmqK4azt/",
			Name:   "menu.dbTransferFileDownload",
			Code:   "db:transfer:files:down",
			Type:   2,
			Weight: 1724395850,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1724398262}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/btVtrbhk/",
			Name:   "menu.dbTransferFileShow",
			Code:   "db:transfer:files",
			Type:   2,
			Weight: 1724376021,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1724998419}}}},
			Pid:    1709194669,
			UiPath: "SmLcpu6c/qINungml/",
			Name:   "menu.dbTransferFileRun",
			Code:   "db:transfer:files:run",
			Type:   2,
			Weight: 1724998419,
		},
		{
			Model:  model.Model{CreateModel: model.CreateModel{DeletedModel: model.DeletedModel{IdModel: model.IdModel{Id: 1729668131}}}},
			Pid:    38,
			UiPath: "dbms23ax/exaeca2x/TGFPA3Ez/",
			Name:   "menu.dbDataOpSqlScriptRun",
			Code:   "db:sqlscript:run",
			Type:   2,
			Weight: 1729668131,
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
		if err := tx.Create(res).Error; err != nil {
			return err
		}
	}

	return nil
}
