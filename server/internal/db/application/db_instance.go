package application

import (
	"context"
	"fmt"
	"math"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/internal/pkg/tlsx"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/structx"
	"strings"
	"time"
)

type Instance interface {
	base.App[*entity.DbInstance]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbInstanceQuery, orderBy ...string) (*model.PageResult[*entity.DbInstance], error)

	// TestConn 测试实例连通性并回报本连接实际协商出的加密状态。
	// material 为表单明文提交的证书材料（编辑态不重传则为 nil，由 normalizeDbTLS 按实例 id 回查旧密文），
	// 与保存共用同一条 TLS 装配路径。返回值只含观测事实：读不到加密状态时 Available=false + Note，
	// 绝不把「读不到」折叠成「未加密」或「已加密」。
	TestConn(ctx context.Context, instanceEntity *entity.DbInstance, authCert *tagentity.ResourceAuthCert, material *tlsx.Material) (*dto.ConnTestResult, error)

	SaveDbInstance(ctx context.Context, instance *dto.SaveDbInstance) (uint64, error)

	// Delete 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// GetDatabases 获取数据库实例的所有数据库列表
	GetDatabases(ctx context.Context, entity *entity.DbInstance, authCert *tagentity.ResourceAuthCert) ([]string, error)

	// GetDatabasesByAc 根据授权凭证名获取所有数据库名称列表
	GetDatabasesByAc(ctx context.Context, acName string) ([]string, error)

	// ToDbInfo 根据实例与授权凭证返回对应的DbInfo
	ToDbInfo(instance *entity.DbInstance, authCertName string, database string) (*dbi.DbInfo, error)

	// Health 采样实例的运行时健康事实（可达性 + 活动会话），供告警评估器消费。
	//
	// 返回的 error 只表示「连采样都无法开始」（实例不存在/取不到连接句柄）；
	// 库不可达、方言不支持会话内省都是被采样到的**事实**，写进 DbHealth 而非当错误返回，
	// 否则上层无法区分「采样失败」与「采到了坏消息」，会把坏消息当成没有数据而静默放过
	Health(ctx context.Context, instanceId uint64) (*dbi.DbHealth, error)
}

type instanceAppImpl struct {
	base.AppImpl[*entity.DbInstance, repository.Instance]

	tagApp              tagapp.TagTreeService     `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert   `inject:"T"`
	dbApp               Db                        `inject:"T"`
	baselineRepo        repository.SchemaBaseline `inject:"T"`
}

var _ Instance = (*instanceAppImpl)(nil)

var _ (Instance) = (*instanceAppImpl)(nil)

// GetPageList 分页获取数据库实例
func (app *instanceAppImpl) GetPageList(condition *entity.DbInstanceQuery, orderBy ...string) (*model.PageResult[*entity.DbInstance], error) {
	return app.GetRepo().GetPageList(condition, orderBy...)
}

func (app *instanceAppImpl) TestConn(ctx context.Context, instanceEntity *entity.DbInstance, authCert *tagentity.ResourceAuthCert, material *tlsx.Material) (*dto.ConnTestResult, error) {
	instanceEntity.Network = resolveInstanceNetwork(instanceEntity)

	authCert, err := app.resourceAuthCertApp.GetRealAuthCert(authCert)
	if err != nil {
		return nil, err
	}

	// 编辑态测试连接：表单不回传证书内容，按 id 回查旧密文，保证「留空=沿用旧证书」
	var old *entity.DbInstance
	if instanceEntity.Id > 0 {
		if stored, getErr := app.GetById(instanceEntity.Id); getErr == nil {
			old = stored
		}
	}
	if err := app.normalizeDbTLS(instanceEntity, old, material); err != nil {
		return nil, err
	}

	dbInfo, err := app.toDbInfoByAc(instanceEntity, authCert, "")
	if err != nil {
		return nil, err
	}
	dbConn, err := dbm.Conn(ctx, dbInfo)
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	// 连上之后立刻问服务端「这条连接到底加密了没有」：配置对不等于协商成功，
	// 而验收需要的正是后者。读不到就如实标注观测缺口。
	res := &dto.ConnTestResult{}
	status, statusErr := dbConn.Metadata().TLSStatus(ctx)
	if statusErr != nil {
		res.TLS = &dbi.TLSStatus{Available: false}
		res.TLSNote = i18n.TC(ctx, imsg.ErrTLSStatusUnavailable, "reason", statusErr.Error())
	} else {
		res.TLS = status
	}
	return res, nil
}

func (app *instanceAppImpl) SaveDbInstance(ctx context.Context, instance *dto.SaveDbInstance) (uint64, error) {
	instanceEntity := instance.DbInstance
	// 默认tcp连接
	instanceEntity.Network = resolveInstanceNetwork(instanceEntity)
	resourceType := consts.ResourceTypeDbInstance
	authCerts := instance.AuthCerts
	tagCodePaths := instance.TagCodePaths

	if len(authCerts) == 0 {
		return 0, errorx.NewBiz("ac cannot be empty")
	}

	// 查找是否存在该库
	oldInstance := &entity.DbInstance{
		Host:               instanceEntity.Host,
		Port:               instanceEntity.Port,
		SshTunnelMachineId: instanceEntity.SshTunnelMachineId,
	}

	err := app.GetByCond(oldInstance)
	if instanceEntity.Id == 0 {
		if err == nil {
			return 0, errorx.NewBizI(ctx, imsg.ErrDbInstExist)
		}
		instanceEntity.Code = stringx.RandLower(10)
		// 新增：没有任何旧密文可沿用，材料必须由本次请求完整提供
		if err := app.normalizeDbTLS(instanceEntity, nil, instance.TLSMaterial); err != nil {
			return 0, err
		}

		return instanceEntity.Id, app.Tx(ctx, func(ctx context.Context) error {
			return app.Insert(ctx, instanceEntity)
		}, func(ctx context.Context) error {
			return app.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
				ResourceCode: instanceEntity.Code,
				ResourceType: tagentity.TagType(resourceType),
				AuthCerts:    authCerts,
			})
		}, func(ctx context.Context) error {
			return app.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
				ResourceTag:        app.genDbInstanceResourceTag(instanceEntity, authCerts),
				ParentTagCodePaths: tagCodePaths,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil {
		if oldInstance.Id != instanceEntity.Id {
			return 0, errorx.NewBizI(ctx, imsg.ErrDbInstExist)
		}
	} else {
		// 根据host等未查到旧数据，则需要根据id重新获取，因为后续需要使用到code
		oldInstance, err = app.GetById(instanceEntity.Id)
		if err != nil {
			return 0, errorx.NewBiz("db instance not found")
		}
	}

	// 校验当前操作者是否有权操作该资源，防止越权修改他人资源信息
	if err := app.tagApp.CanAccessByCode(ctx, resourceType, oldInstance.Code); err != nil {
		return 0, err
	}

	if err := app.normalizeDbTLS(instanceEntity, oldInstance, instance.TLSMaterial); err != nil {
		return 0, err
	}
	tlsChanged := oldInstance.GetExtraString(entity.ExtraKeyTLSFingerprint) != instanceEntity.GetExtraString(entity.ExtraKeyTLSFingerprint) ||
		oldInstance.TLSMaterialEnc != instanceEntity.TLSMaterialEnc

	err = app.Tx(ctx, func(ctx context.Context) error {
		return app.UpdateById(ctx, instanceEntity)
	}, func(ctx context.Context) error {
		// extra/tls_material 必须显式点名：结构体更新会跳过零值，
		// 「关闭加密后 extra 少了一个键」「tls_material 清空」都写不进去，
		// 表现为保存成功但连接仍按旧配置建立（与 mask 模块 SaveAllColumns 同一取舍）
		return app.GetRepo().UpdateById(ctx, instanceEntity, "extra", "tls_material",
			"update_time", "modifier_id", "modifier")
	}, func(ctx context.Context) error {
		return app.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: oldInstance.Code,
			ResourceType: tagentity.TagType(resourceType),
			AuthCerts:    authCerts,
		})
	}, func(ctx context.Context) error {
		if instanceEntity.Name != oldInstance.Name {
			if err := app.tagApp.UpdateTagName(ctx, tagentity.TagTypeDbInstance, oldInstance.Code, instanceEntity.Name); err != nil {
				return err
			}
		}
		return app.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag:        app.genDbInstanceResourceTag(oldInstance, authCerts),
			ParentTagCodePaths: tagCodePaths,
		})
	})
	if err != nil {
		return 0, err
	}
	// 连接级配置变了必须让旧连接池失效：池按「库+locator」缓存，不关就会继续
	// 用旧证书/明文握手，用户只看到「改了没生效」。放在事务提交之后，避免回滚了却已断连
	if tlsChanged {
		dbm.CloseInstanceConns(oldInstance.Id)
	}
	return oldInstance.Id, nil
}

func (app *instanceAppImpl) Delete(ctx context.Context, instanceId uint64) error {
	instance, err := app.GetById(instanceId)
	if err != nil {
		return errorx.NewBiz("db instnace not found")
	}

	dbs, err := app.dbApp.ListByCond(&entity.Db{
		InstanceId: instanceId,
	})
	if err != nil {
		return errorx.NewBizf("failed to list dbs of instance: %s", err.Error())
	}

	return app.Tx(ctx, func(ctx context.Context) error {
		return app.DeleteById(ctx, instanceId)
	}, func(ctx context.Context) error {
		// 删除该实例关联的授权凭证信息
		return app.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: instance.Code,
			ResourceType: tagentity.TagType(consts.ResourceTypeDbInstance),
		})
	}, func(ctx context.Context) error {
		return app.tagApp.DeleteTagByParam(ctx, &tagdto.DelResourceTag{
			ResourceCode: instance.Code,
			ResourceType: tagentity.TagType(consts.ResourceTypeDbInstance),
		})
	}, func(ctx context.Context) error {
		// 删除所有库配置
		for _, db := range dbs {
			if err := app.dbApp.Delete(ctx, db.Id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (app *instanceAppImpl) GetDatabases(ctx context.Context, ed *entity.DbInstance, authCert *tagentity.ResourceAuthCert) ([]string, error) {
	if authCert == nil {
		return nil, errorx.NewBiz("auth cert cannot be empty")
	}
	if authCert.Id != 0 {
		// 密文可能被清除，故需要重新获取
		newAuthCert, err := app.resourceAuthCertApp.GetAuthCert(authCert.Name)
		if err != nil {
			return nil, err
		}
		authCert = newAuthCert
	} else {
		if authCert.CiphertextType == tagentity.AuthCertCiphertextTypePublic {
			publicAuthCert, err := app.resourceAuthCertApp.GetAuthCert(authCert.Ciphertext)
			if err != nil {
				return nil, err
			}
			authCert = publicAuthCert
		}
	}

	return app.getDatabases(ctx, ed, authCert)
}

func (app *instanceAppImpl) GetDatabasesByAc(ctx context.Context, acName string) ([]string, error) {
	ac, err := app.resourceAuthCertApp.GetAuthCert(acName)
	if err != nil {
		return nil, errorx.NewBiz("db ac not found")
	}

	instance := &entity.DbInstance{Code: ac.ResourceCode}
	err = app.GetByCond(instance)
	if err != nil {
		return nil, errorx.NewBiz("the db instance information for this ac does not exist")
	}

	return app.getDatabases(ctx, instance, ac)
}

func (app *instanceAppImpl) ToDbInfo(instance *entity.DbInstance, authCertName string, database string) (*dbi.DbInfo, error) {
	ac, err := app.resourceAuthCertApp.GetAuthCert(authCertName)
	if err != nil {
		return nil, err
	}

	return app.toDbInfoByAc(instance, ac, database)
}

func (app *instanceAppImpl) getDatabases(ctx context.Context, instance *entity.DbInstance, ac *tagentity.ResourceAuthCert) ([]string, error) {
	instance.Network = resolveInstanceNetwork(instance)
	info, err := app.toDbInfoByAc(instance, ac, "")
	if err != nil {
		return nil, err
	}

	dbConn, err := dbm.Conn(ctx, info)
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	return dbConn.Metadata().GetDbNames()
}

// toDbInfoByAc 把实例 + 授权凭证 + 库位组装成连接信息。
//
// 这是「实例配置 → 连接参数」的唯一转换点：TLS 的解密与结构化解码也收在这里，
// 因此测试连接、取库名、正式建连三条路径看到的加密配置必然一致。
func (app *instanceAppImpl) toDbInfoByAc(instance *entity.DbInstance, ac *tagentity.ResourceAuthCert, database string) (*dbi.DbInfo, error) {
	di := new(dbi.DbInfo)
	di.InstanceId = instance.Id
	structx.Copy(di, instance)
	di.Extra = instance.Extra

	// 库位 locator（`库/schema`）在此单点解析为结构化字段：di.Database 收窄为纯库名、schema 进 di.Schema。
	// 用 di.Type（structx.Copy 已从实例拷入方言类型）判定是否拆 `/`；无 schema 层的方言整串即库名，不拆。
	di.Database, di.Schema = dbi.ParseLocator(di.Type, database)

	di.Username = ac.Username
	di.Password = ac.Ciphertext

	tlsCfg, err := app.resolveTLS(instance)
	if err != nil {
		return nil, err
	}
	di.TLS = tlsCfg
	return di, nil
}

// resolveTLS 把实例上「可读参数（Extra）+ 密文材料（列）」组装成运行期 TLS 配置。
//
// 明文档不读取也不解释材料：mode 为 disable 时残留密文一律忽略，
// 避免「界面显示已关闭、连接却仍按旧证书走」的错位。
func (app *instanceAppImpl) resolveTLS(instance *entity.DbInstance) (*tlsx.Config, error) {
	mode := tlsx.Mode(strings.TrimSpace(instance.GetExtraString(entity.ExtraKeyTLSMode)))
	if !mode.Encrypted() {
		return nil, nil
	}
	material, err := tlsx.OpenMaterial(instance.TLSMaterialEnc)
	if err != nil {
		return nil, err
	}
	return &tlsx.Config{
		Mode:        mode,
		ServerName:  strings.TrimSpace(instance.GetExtraString(entity.ExtraKeyTLSServerName)),
		UseSystemCA: instance.GetExtraBool(entity.ExtraKeyTLSUseSystemCA),
		Material:    material,
	}, nil
}

// normalizeDbTLS 保存/测试连接前归一化 TLS 配置并落库密文。
//
// 语义与机器 SOCKS5 代理（normalizeMachineProxy）保持同构：
//   - 档位白名单校验（含方言能力位），不支持就不让存，而不是等着建连时报错；
//   - 材料留空 = 不修改（编辑表单不回显证书，用户不重传就沿用旧密文）；
//   - 关闭加密即清空材料，避免残留凭据随 extra/列一起被带走。
func (app *instanceAppImpl) normalizeDbTLS(me *entity.DbInstance, old *entity.DbInstance, material *tlsx.Material) error {
	mode := tlsx.Mode(strings.TrimSpace(me.GetExtraString(entity.ExtraKeyTLSMode)))
	if !mode.Encrypted() {
		me.TLSMaterialEnc = ""
		me.SetExtraValue(entity.ExtraKeyTLSFingerprint, "")
		me.SetExtraValue(entity.ExtraKeyTLSMode, "")
		return nil
	}
	if !mode.Valid() {
		return errorx.NewBizf("unsupported tls mode: %q", string(mode))
	}
	backend := dbi.GetBackend(dbi.DbType(me.Type))
	if backend == nil {
		return errorx.NewBizf("unsupported database type: %q", me.Type)
	}
	if !backend.GetCapabilities().SupportsTLSMode(mode) {
		return errorx.NewBizf("database type [%s] does not support tls mode [%s]", me.Type, string(mode))
	}

	// 留空沿用旧密文：编辑表单从不回显证书内容，用户不重传就代表「没改」。
	//
	// 必须**逐字段**合并而不是「整体为空才回退」：三个输入框是独立提交的，
	// 只轮换 CA 时 resolved 非空，整体替换会把已存的客户端证书/私钥静默清掉——
	// 保存显示成功，下次建连才失败，而旧密文已被覆盖，不可恢复。
	resolved := tlsx.Material{}
	if material != nil {
		resolved = *material
	}
	if old != nil && old.TLSMaterialEnc != "" {
		oldMaterial, err := tlsx.OpenMaterial(old.TLSMaterialEnc)
		if err != nil {
			return err
		}
		resolved = resolved.MergeMissing(oldMaterial)
	}

	cfg := &tlsx.Config{
		Mode:        mode,
		ServerName:  strings.TrimSpace(me.GetExtraString(entity.ExtraKeyTLSServerName)),
		UseSystemCA: me.GetExtraBool(entity.ExtraKeyTLSUseSystemCA),
		Material:    resolved,
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	sealed, err := resolved.Seal()
	if err != nil {
		return err
	}
	me.TLSMaterialEnc = sealed
	me.SetExtraValue(entity.ExtraKeyTLSMode, string(mode))
	me.SetExtraValue(entity.ExtraKeyTLSFingerprint, cfg.Key())
	return nil
}

func (m *instanceAppImpl) genDbInstanceResourceTag(me *entity.DbInstance, authCerts []*tagentity.ResourceAuthCert) *tagdto.ResourceTag {
	authCertTags := collx.ArrayMap[*tagentity.ResourceAuthCert, *tagdto.ResourceTag](authCerts, func(val *tagentity.ResourceAuthCert) *tagdto.ResourceTag {
		return &tagdto.ResourceTag{
			Code: val.Name,
			Name: val.Username,
			Type: tagentity.TagTypeAuthCert,
		}
	})

	dbs, err := m.dbApp.ListByCond(&entity.Db{
		InstanceId: me.Id,
	})
	if err != nil {
		logx.Errorf("failed to retrieve the database associated with the instance: %v", err)
	}

	authCertName2DbTags := make(map[string][]*tagdto.ResourceTag)
	for _, db := range dbs {
		authCertName2DbTags[db.AuthCertName] = append(authCertName2DbTags[db.AuthCertName], &tagdto.ResourceTag{
			Code: db.Code,
			Name: db.Name,
			Type: tagentity.TagTypeDb,
		})
	}

	// 将数据库挂至授权凭证下
	for _, ac := range authCertTags {
		ac.Children = authCertName2DbTags[ac.Code]
	}

	return &tagdto.ResourceTag{
		Code:     me.Code,
		Type:     tagentity.TagTypeDbInstance,
		Name:     me.Name,
		Children: authCertTags,
	}
}

// resolveInstanceNetwork 根据实例配置计算实际连接网络协议。
// 未使用 SSH 隧道时返回配置的 Network（缺省 "tcp"）；
// 使用 SSH 隧道时返回 "{type}+ssh:{machineId}" 格式的网络标识。
func resolveInstanceNetwork(d *entity.DbInstance) string {
	if d.SshTunnelMachineId <= 0 {
		if d.Network == "" {
			return "tcp"
		}
		return d.Network
	}
	return fmt.Sprintf("%s+ssh:%d", d.Type, d.SshTunnelMachineId)
}

// Health 见接口注释：事实采集，不做判定（阈值语义归告警规则）
func (i *instanceAppImpl) Health(ctx context.Context, instanceId uint64) (*dbi.DbHealth, error) {
	health := &dbi.DbHealth{InstanceId: instanceId, CheckedAt: time.Now()}

	// 漂移事实先填：它存在平台自己的基线表里，不依赖库可达——库连不上时「结构有没有走形」仍然是已知事实，
	// 排在后面就会被两条提前 return 跳过
	if rows, drifted, err := i.baselineRepo.DriftStat(instanceId); err == nil {
		// 基线行数为 0 就是没建过基线：BaselineExists 保持 false，不能拿「没漂移行」当成「有基线且稳定」
		health.BaselineExists = rows > 0
		health.DriftedTables = int(drifted)
	} else {
		logx.DebugfContext(ctx, "count schema drift for db instance [%d] failed: %s", instanceId, err.Error())
	}

	// TLS 证书剩余天数不依赖库可达：它来自平台保存的证书本身，库连不上时「证书什么时候过期」
	// 依然是已知事实，因此排在取连接之前
	if instance, instErr := i.GetById(instanceId); instErr == nil {
		if tlsCfg, tlsErr := i.resolveTLS(instance); tlsErr == nil && tlsCfg != nil {
			if expiry, ok := tlsCfg.Material.EarliestExpiry(); ok {
				days := int(math.Floor(time.Until(expiry).Hours() / 24))
				health.TLSCertDaysLeft = &days
			}
		} else if tlsErr != nil {
			// 解密失败同样是事实（如 aes.key 轮换后旧密文读不出），但证书指标只能标不可判定
			logx.DebugfContext(ctx, "resolve tls config of db instance [%d] failed: %s", instanceId, tlsErr.Error())
		}
	}

	conn, err := i.dbApp.GetDbConnByInstanceId(ctx, instanceId)
	if err != nil {
		// 取不到连接句柄本身就是「不可达」的事实（含实例不存在），而不是采样故障：
		// 若按 error 返回，规则里的 ping_fail 指标永远取不到值，实例真挂了也不告警
		health.PingErr = err
		return health, nil
	}

	if err := conn.Ping(); err != nil {
		health.PingErr = err
		return health, nil
	}

	// PITR 前提：与可达性一起采，但失败不影响其它事实（读不到就保持 nil，不谎报为「未就绪」）
	if readiness, recErr := conn.Metadata().RecoveryReadiness(ctx); recErr == nil {
		prepared := readiness.Prepared
		health.RecoveryPrepared = &prepared
		health.RecoveryReason = readiness.Reason
	} else {
		logx.DebugfContext(ctx, "inspect recovery readiness of db instance [%d] failed: %s", instanceId, recErr.Error())
	}

	sessions, err := conn.Metadata().ListSessions(ctx)
	if err != nil {
		// 能力缺失（或库侧拒绝查系统表）：标记为不可观测，让依赖会话的指标返回错误而非 0
		logx.DebugfContext(ctx, "list sessions for db instance [%d] health failed: %s", instanceId, err.Error())
		return health, nil
	}
	health.Sessions = sessions
	health.SessionsAvailable = true
	return health, nil
}
