package api

import (
	"fmt"
	"mayfly-go/internal/event"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/guac"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/ws"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/may-fly/cast"
)

type Machine struct {
	machineApp          application.Machine       `inject:"T"`
	machineTermOpApp    application.MachineTermOp `inject:"T"`
	tagTreeApp          tagapp.TagTree            `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert   `inject:"T"`
}

func (m *Machine) ReqConfs() *req.Confs {
	saveMachineP := req.NewPermission("machine:update")

	reqs := [...]*req.Conf{
		req.NewGet("", m.Machines),

		req.NewGet("/simple", m.SimpleMachieInfo),

		req.NewGet(":machineId/stats", m.MachineStats),

		req.NewGet(":machineId/process", m.GetProcess),

		req.NewGet(":machineId/users", m.GetUsers),

		req.NewGet(":machineId/groups", m.GetGroups),

		req.NewPost("", m.SaveMachine).Log(req.NewLogSaveI(imsg.LogMachineSave)).RequiredPermission(saveMachineP),

		req.NewDelete(":machineId", m.DeleteMachine).Log(req.NewLogSaveI(imsg.LogMachineSave)),

		req.NewPost("test-conn", m.TestConn),

		req.NewPut(":machineId/:status", m.ChangeStatus).Log(req.NewLogSaveI(imsg.LogMachineChangeStatus)).RequiredPermission(saveMachineP),

		req.NewDelete(":machineId/process", m.KillProcess).Log(req.NewLogSaveI(imsg.LogMachineKillProcess)).RequiredPermissionCode("machine:killprocess"),

		// 获取机器终端回放记录列表,目前具有保存机器信息的权限标识才有权限查看终端回放
		req.NewGet(":machineId/term-recs", m.MachineTermOpRecords).RequiredPermission(saveMachineP),

		// 终端操作
		req.NewGet("terminal/:ac", m.WsSSH).NoRes(),
		req.NewGet("rdp/:ac", m.WsGuacamole).NoRes(),
	}

	return req.NewConfs("machines", reqs[:]...)
}

func (m *Machine) Machines(rc *req.Ctx) {
	condition := req.BindQuery[*entity.MachineQuery](rc)

	tags := m.tagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeMachine, tagentity.TagTypeAuthCert)),
		CodePathLikes: collx.AsArray(condition.TagPath),
	})
	// 不存在可操作的机器-授权凭证标签，即没有可操作数据
	if len(tags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}

	tagCodePaths := tags.GetCodePaths()
	machineCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeMachine, tagCodePaths...)
	condition.Codes = collx.ArrayDeduplicate(machineCodes)

	res, err := m.machineApp.GetMachineList(condition)
	biz.ErrIsNil(err)
	if res.Total == 0 {
		rc.ResData = res
		return
	}

	resVo := model.PageResultConv[*entity.Machine, *vo.MachineVO](res)
	machinevos := resVo.List

	// 填充标签信息
	m.tagTreeApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeMachine), collx.ArrayMap(machinevos, func(mvo *vo.MachineVO) tagentity.ITagResource {
		return mvo
	})...)

	// 填充授权凭证信息
	m.resourceAuthCertApp.FillAuthCertByAcNames(tagentity.GetCodesByCodePaths(tagentity.TagTypeAuthCert, tagCodePaths...), collx.ArrayMap(machinevos, func(mvo *vo.MachineVO) tagentity.IAuthCert {
		return mvo
	})...)

	for _, mv := range machinevos {
		if machineStats, err := m.machineApp.GetMachineStats(mv.Id); err == nil {
			mv.Stat = collx.M{
				"cpuIdle":      machineStats.CPU.Idle,
				"memAvailable": machineStats.MemInfo.Available,
				"memTotal":     machineStats.MemInfo.Total,
				"fsInfos":      machineStats.FSInfos,
			}
		}
	}
	rc.ResData = resVo
}

func (m *Machine) SimpleMachieInfo(rc *req.Ctx) {
	machineCodesStr := rc.Query("codes")
	biz.NotEmpty(machineCodesStr, "codes cannot be empty")

	var vos []vo.SimpleMachineVO
	m.machineApp.ListByCondToAny(model.NewCond().In("code", strings.Split(machineCodesStr, ",")), &vos)
	rc.ResData = vos
}

func (m *Machine) MachineStats(rc *req.Ctx) {
	cli, err := m.machineApp.GetCli(rc.MetaCtx, GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "connection error: %s")

	rc.ResData = cli.GetAllStats()
}

// 保存机器信息
func (m *Machine) SaveMachine(rc *req.Ctx) {
	machineForm, me := req.BindJsonAndCopyTo[*form.MachineForm, *entity.Machine](rc)

	rc.ReqParam = machineForm

	biz.ErrIsNil(m.machineApp.SaveMachine(rc.MetaCtx, &dto.SaveMachine{
		Machine:      me,
		TagCodePaths: machineForm.TagCodePaths,
		AuthCerts:    machineForm.AuthCerts,
	}))
}

func (m *Machine) TestConn(rc *req.Ctx) {
	machineForm, me := req.BindJsonAndCopyTo[*form.MachineForm, *entity.Machine](rc)
	// 测试连接
	biz.ErrIsNilAppendErr(m.machineApp.TestConn(rc.MetaCtx, me, machineForm.AuthCerts[0]), "connection error: %s")
}

func (m *Machine) ChangeStatus(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("machineId"))
	status := int8(rc.PathParamInt("status"))
	rc.ReqParam = collx.Kvs("id", id, "status", status)
	biz.ErrIsNil(m.machineApp.ChangeStatus(rc.MetaCtx, id, status))
}

func (m *Machine) DeleteMachine(rc *req.Ctx) {
	idsStr := rc.PathParam("machineId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		m.machineApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

// 获取进程列表信息
func (m *Machine) GetProcess(rc *req.Ctx) {
	cmd := "ps -aux "
	sortType := rc.Query("sortType")
	if sortType == "2" {
		cmd += "--sort -pmem "
	} else {
		cmd += "--sort -pcpu "
	}

	pname := rc.Query("name")
	if pname != "" {
		cmd += fmt.Sprintf("| grep %s ", pname)
	}

	count := rc.QueryIntDefault("count", 10)
	cmd += "| head -n " + fmt.Sprintf("%d", count)

	cli, err := m.machineApp.GetCli(rc.MetaCtx, GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "connection error: %s")

	biz.ErrIsNilAppendErr(m.tagTreeApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), "%s")

	res, err := cli.Run(cmd)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// 终止进程
func (m *Machine) KillProcess(rc *req.Ctx) {
	pid := rc.Query("pid")
	biz.NotEmpty(pid, "pid cannot be empty")

	cli, err := m.machineApp.GetCli(rc.MetaCtx, GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "connection error: %s")

	biz.ErrIsNilAppendErr(m.tagTreeApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), "%s")

	res, err := cli.Run("sudo kill -9 " + pid)
	biz.ErrIsNil(err, "kill fail: %s", res)
}

func (m *Machine) GetUsers(rc *req.Ctx) {
	cli, err := m.machineApp.GetCli(rc.MetaCtx, GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "connection error: %s")

	res, err := cli.GetUsers()
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *Machine) GetGroups(rc *req.Ctx) {
	cli, err := m.machineApp.GetCli(rc.MetaCtx, GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "connection error: %s")

	res, err := cli.GetGroups()
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *Machine) WsSSH(rc *req.Ctx) {
	wsConn, err := ws.Upgrader.Upgrade(rc.GetWriter(), rc.GetRequest(), nil)
	defer func() {
		if wsConn != nil {
			if err := recover(); err != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(anyx.ToString(err)))
			}
			wsConn.Close()
		}
	}()
	biz.ErrIsNilAppendErr(err, "Upgrade websocket fail: %s")
	wsConn.WriteMessage(websocket.TextMessage, []byte("Connecting to host..."))

	// 权限校验
	rc = rc.WithRequiredPermission(req.NewPermission("machine:terminal"))
	err = req.PermissionHandler(rc)
	biz.ErrIsNil(err, mcm.GetErrorContentRn("You do not have permission to operate the machine terminal, please log in again and try again ~"))

	cli, err := m.machineApp.NewCli(rc.MetaCtx, GetMachineAc(rc))
	biz.ErrIsNilAppendErr(err, mcm.GetErrorContentRn("connection error: %s"))
	defer cli.Close()
	biz.ErrIsNilAppendErr(m.tagTreeApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), mcm.GetErrorContentRn("%s"))

	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, cli.Info.CodePath[0])

	cols := rc.QueryIntDefault("cols", 80)
	rows := rc.QueryIntDefault("rows", 32)

	// 记录系统操作日志
	rc.WithLog(req.NewLogSaveI(imsg.LogMachineTerminalOp))
	rc.ReqParam = cli.Info

	err = m.machineTermOpApp.TermConn(rc.MetaCtx, cli, wsConn, rows, cols)
	biz.ErrIsNilAppendErr(err, mcm.GetErrorContentRn("connect fail: %s"))
}

func (m *Machine) MachineTermOpRecords(rc *req.Ctx) {
	mid := GetMachineId(rc)
	res, err := m.machineTermOpApp.GetPageList(&entity.MachineTermOp{MachineId: mid}, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

const (
	SocketTimeout            = 15 * time.Second
	MaxGuacMessage           = 8192
	websocketReadBufferSize  = MaxGuacMessage
	websocketWriteBufferSize = MaxGuacMessage * 2
)

var (
	sessions = guac.NewMemorySessionStore()
)

func (m *Machine) WsGuacamole(rc *req.Ctx) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  websocketReadBufferSize,
		WriteBufferSize: websocketWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	reqWriter := rc.GetWriter()
	request := rc.GetRequest()

	wsConn, err := upgrader.Upgrade(reqWriter, request, nil)
	biz.ErrIsNil(err)

	rc = rc.WithRequiredPermission(req.NewPermission("machine:terminal"))
	err = req.PermissionHandler(rc)
	biz.ErrIsNil(err, mcm.GetErrorContentRn("You do not have permission to operate the machine terminal, please log in again and try again ~"))

	ac := GetMachineAc(rc)

	mi, err := m.machineApp.ToMachineInfoByAc(ac)
	if err != nil {
		return
	}

	err = mi.IfUseSshTunnelChangeIpPort(rc.MetaCtx, true)
	if err != nil {
		return
	}

	params := make(map[string]string)
	params["hostname"] = mi.Ip
	params["port"] = strconv.Itoa(mi.Port)
	params["username"] = mi.Username
	params["password"] = mi.Password
	params["ignore-cert"] = "true"

	if mi.Protocol == 2 {
		params["scheme"] = "rdp"
	} else if mi.Protocol == 3 {
		params["scheme"] = "vnc"
	}

	if mi.EnableRecorder == 1 {
		// 操作记录 查看文档：https://guacamole.apache.org/doc/gug/configuring-guacamole.html#graphical-recording
		params["recording-path"] = fmt.Sprintf("/rdp-rec/%s", ac)
		params["create-recording-path"] = "true"
		params["recording-include-keys"] = "true"
	}

	defer func() {
		if err = wsConn.Close(); err != nil {
			logx.Warnf("Error closing websocket: %v", err)
		}
	}()

	if rc.Query("force") != "" {
		// 判断是否强制连接，是的话，查询是否有正在连接的会话，有的话强制关闭
		if cast.ToBool(rc.Query("force")) {
			tn := sessions.Get(ac)
			if tn != nil {
				_ = tn.Close()
			}
		}
	}

	tunnel, err := guac.DoConnect(request.URL.Query(), params, rc.GetLoginAccount().Username)
	if err != nil {
		return
	}
	defer func() {
		if err = tunnel.Close(); err != nil {
			logx.Warnf("Error closing tunnel: %v", err)
		}
	}()

	sessions.Add(ac, wsConn, request, tunnel)

	defer sessions.Delete(ac, wsConn, request, tunnel)

	writer := tunnel.AcquireWriter()
	reader := tunnel.AcquireReader()

	defer tunnel.ReleaseWriter()
	defer tunnel.ReleaseReader()

	go guac.WsToGuacd(wsConn, tunnel, writer)
	guac.GuacdToWs(wsConn, tunnel, reader)

	//OnConnect
	//OnDisconnect
}

func GetMachineId(rc *req.Ctx) uint64 {
	machineId, _ := strconv.Atoi(rc.PathParam("machineId"))
	biz.IsTrue(machineId != 0, "machineId error")
	return uint64(machineId)
}

func GetMachineAc(rc *req.Ctx) string {
	ac := rc.PathParam("ac")
	biz.IsTrue(ac != "", "authCertName error")
	return ac
}
