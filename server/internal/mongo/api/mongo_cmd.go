package api

import (
	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

// Commands 下发命令控制台目录。
//
// 级别、要求的权限码、是否需要二次确认、命令模板全部由服务端单源给出：
// 旧实现把可选命令与模板硬编码在前端，既无法与后端鉴权口径对齐，也让「找不到入口」等于「功能丢失」。
// 目录是静态知识、不涉及实例数据，因此不需要建立连接（按实例探测能力属于后续演进）。
func (m *Mongo) Commands(rc *req.Ctx) {
	rc.ResData = mongodoc.Catalog()
}

// RunCommand 执行管理命令。
//
// 与静态权限码路由不同：透传型入口能做的事取决于命令本身，所以必须按命令语义动态判定权限。
// 旧入口零鉴权，只读账号即可 dropDatabase / createUser，等于绕过平台的全部数据面约束；
// 现在未登记的命令一律按管理级处理（fail-closed）。
func (m *Mongo) RunCommand(rc *req.Ctx) {
	cmdForm := rc.BindJson[form.RunCommandForm]()
	conn := m.getConn(rc)

	// 命令以原始 JSON 文本进入：字段顺序按用户输入保留，多字段命令不再被 map 迭代序吞掉。
	// 旧契约用 []map[string]any 承载顺序，前端必须把每个顶层字段拆成单键 map 才不出错，
	// 那是没有文档与校验的隐性约定，任何直连调用方都会踩。
	command := decodeJSONField(rc, cmdForm.Command, "command")

	name, level, err := mongodoc.Classify(command)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	m.requireCmdPerm(rc, name, level)

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", cmdForm.Database,
		"command", name,
		"level", level.String(),
		"content", string(cmdForm.Command),
	)
	m.publishResourceOp(rc, conn)

	res, err := m.mongoData.RunCommand(rc.MetaCtx, conn, cmdForm.Database, command)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = res
}

// levelNames 命令级别的可读名称，用于拼进权限拒绝提示。
//
// 裸权限码对操作者没有信息量，必须说清「是这条命令太危险」而不是「你没权限」。
var levelNames = map[mongodoc.Level]i18n.MsgId{
	mongodoc.LevelRead:       imsg.LevelReadName,
	mongodoc.LevelDataSave:   imsg.LevelDataSaveName,
	mongodoc.LevelDataDel:    imsg.LevelDataDelName,
	mongodoc.LevelStructSave: imsg.LevelStructSaveName,
	mongodoc.LevelStructDel:  imsg.LevelStructDelName,
	mongodoc.LevelAdmin:      imsg.LevelAdminName,
}

// requireCmdPerm 按命令语义校验权限码。只读命令在此直接放行（资源可见性已在 getConn 校验）。
func (m *Mongo) requireCmdPerm(rc *req.Ctx, command string, level mongodoc.Level) {
	code := level.PermissionCode()
	if code == "" {
		return
	}
	if req.GetPermissionCodeRegistery().HasCode(rc.GetLoginAccount().Id, code) {
		return
	}

	levelMsgId, ok := levelNames[level]
	if !ok {
		levelMsgId = imsg.LevelAdminName
	}
	biz.ErrIsNil(errorx.NewBizI(rc.MetaCtx, imsg.ErrMongoCmdPermDenied,
		"command", command,
		"level", i18n.TC(rc.MetaCtx, levelMsgId),
		"permission", code,
	))
}
