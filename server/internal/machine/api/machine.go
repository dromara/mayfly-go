package api

import (
	"encoding/base64"
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/event"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/guac"
	"mayfly-go/internal/machine/mcm"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/ws"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/may-fly/cast"
)

type Machine struct {
	MachineApp          application.Machine       `inject:""`
	MachineTermOpApp    application.MachineTermOp `inject:""`
	TagApp              tagapp.TagTree            `inject:"TagTreeApp"`
	ResourceAuthCertApp tagapp.ResourceAuthCert   `inject:""`
}

func (m *Machine) Machines(rc *req.Ctx) {
	condition, pageParam := req.BindQueryAndPage(rc, new(entity.MachineQuery))

	tagCodePaths := m.TagApp.GetAccountTagCodePaths(rc.GetLoginAccount().Id, tagentity.TagTypeMachineAuthCert, condition.TagPath)
	// 不存在可操作的机器-授权凭证标签，即没有可操作数据
	if len(tagCodePaths) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}

	machineCodes := tagentity.GetCodeByPath(tagentity.TagTypeMachine, tagCodePaths...)
	condition.Codes = collx.ArrayDeduplicate(machineCodes)

	var machinevos []*vo.MachineVO
	res, err := m.MachineApp.GetMachineList(condition, pageParam, &machinevos)
	biz.ErrIsNil(err)
	if res.Total == 0 {
		rc.ResData = res
		return
	}

	// 填充标签信息
	m.TagApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeMachine), collx.ArrayMap(machinevos, func(mvo *vo.MachineVO) tagentity.ITagResource {
		return mvo
	})...)

	// 填充授权凭证信息
	m.ResourceAuthCertApp.FillAuthCertByAcNames(tagentity.GetCodeByPath(tagentity.TagTypeMachineAuthCert, tagCodePaths...), collx.ArrayMap(machinevos, func(mvo *vo.MachineVO) tagentity.IAuthCert {
		return mvo
	})...)

	for _, mv := range machinevos {
		if machineStats, err := m.MachineApp.GetMachineStats(mv.Id); err == nil {
			mv.Stat = collx.M{
				"cpuIdle":      machineStats.CPU.Idle,
				"memAvailable": machineStats.MemInfo.Available,
				"memTotal":     machineStats.MemInfo.Total,
				"fsInfos":      machineStats.FSInfos,
			}
		}
	}
	rc.ResData = res
}

func (m *Machine) MachineStats(rc *req.Ctx) {
	cli, err := m.MachineApp.GetCli(GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "获取客户端连接失败: %s")
	rc.ResData = cli.GetAllStats()
}

// 保存机器信息
func (m *Machine) SaveMachine(rc *req.Ctx) {
	machineForm := new(form.MachineForm)
	me := req.BindJsonAndCopyTo(rc, machineForm, new(entity.Machine))

	rc.ReqParam = machineForm

	biz.ErrIsNil(m.MachineApp.SaveMachine(rc.MetaCtx, &dto.SaveMachine{
		Machine:      me,
		TagCodePaths: machineForm.TagCodePaths,
		AuthCerts:    machineForm.AuthCerts,
	}))
}

func (m *Machine) TestConn(rc *req.Ctx) {
	machineForm := new(form.MachineForm)
	me := req.BindJsonAndCopyTo(rc, machineForm, new(entity.Machine))
	// 测试连接
	biz.ErrIsNilAppendErr(m.MachineApp.TestConn(me, machineForm.AuthCerts[0]), "该机器无法连接: %s")
}

func (m *Machine) ChangeStatus(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("machineId"))
	status := int8(rc.PathParamInt("status"))
	rc.ReqParam = collx.Kvs("id", id, "status", status)
	biz.ErrIsNil(m.MachineApp.ChangeStatus(rc.MetaCtx, id, status))
}

func (m *Machine) DeleteMachine(rc *req.Ctx) {
	idsStr := rc.PathParam("machineId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MachineApp.Delete(rc.MetaCtx, uint64(value))
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

	cli, err := m.MachineApp.GetCli(GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "获取客户端连接失败: %s")
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), "%s")

	res, err := cli.Run(cmd)
	biz.ErrIsNilAppendErr(err, "获取进程信息失败: %s")
	rc.ResData = res
}

// 终止进程
func (m *Machine) KillProcess(rc *req.Ctx) {
	pid := rc.Query("pid")
	biz.NotEmpty(pid, "进程id不能为空")

	cli, err := m.MachineApp.GetCli(GetMachineId(rc))
	biz.ErrIsNilAppendErr(err, "获取客户端连接失败: %s")
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), "%s")

	res, err := cli.Run("sudo kill -9 " + pid)
	biz.ErrIsNil(err, "终止进程失败: %s", res)
}

func (m *Machine) WsSSH(g *gin.Context) {
	wsConn, err := ws.Upgrader.Upgrade(g.Writer, g.Request, nil)
	defer func() {
		if wsConn != nil {
			if err := recover(); err != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(anyx.ToString(err)))
			}
			wsConn.Close()
		}
	}()
	biz.ErrIsNilAppendErr(err, "升级websocket失败: %s")
	wsConn.WriteMessage(websocket.TextMessage, []byte("Connecting to host..."))

	// 权限校验
	rc := req.NewCtxWithGin(g).WithRequiredPermission(req.NewPermission("machine:terminal"))
	if err = req.PermissionHandler(rc); err != nil {
		panic(errorx.NewBiz(mcm.GetErrorContentRn("您没有权限操作该机器终端,请重新登录后再试~")))
	}

	cli, err := m.MachineApp.NewCli(GetMachineAc(rc))
	biz.ErrIsNilAppendErr(err, mcm.GetErrorContentRn("获取客户端连接失败: %s"))
	defer cli.Close()
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), mcm.GetErrorContentRn("%s"))

	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, cli.Info.CodePath[0])

	cols := rc.QueryIntDefault("cols", 80)
	rows := rc.QueryIntDefault("rows", 32)

	// 记录系统操作日志
	rc.WithLog(req.NewLogSave("机器-终端操作"))
	rc.ReqParam = cli.Info
	req.LogHandler(rc)

	err = m.MachineTermOpApp.TermConn(rc.MetaCtx, cli, wsConn, rows, cols)
	biz.ErrIsNilAppendErr(err, mcm.GetErrorContentRn("连接失败: %s"))
}

func (m *Machine) MachineTermOpRecords(rc *req.Ctx) {
	mid := GetMachineId(rc)
	res, err := m.MachineTermOpApp.GetPageList(&entity.MachineTermOp{MachineId: mid}, rc.GetPageParam(), new([]entity.MachineTermOp))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *Machine) MachineTermOpRecord(rc *req.Ctx) {
	termOp, err := m.MachineTermOpApp.GetById(uint64(rc.PathParamInt("recId")))
	biz.ErrIsNil(err)

	bytes, err := os.ReadFile(path.Join(config.GetMachine().TerminalRecPath, termOp.RecordFilePath))
	biz.ErrIsNilAppendErr(err, "读取终端操作记录失败: %s")
	rc.ResData = base64.StdEncoding.EncodeToString(bytes)
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

func (m *Machine) WsGuacamole(g *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  websocketReadBufferSize,
		WriteBufferSize: websocketWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	wsConn, err := upgrader.Upgrade(g.Writer, g.Request, nil)
	biz.ErrIsNil(err)

	rc := req.NewCtxWithGin(g).WithRequiredPermission(req.NewPermission("machine:terminal"))
	if err = req.PermissionHandler(rc); err != nil {
		panic(errorx.NewBiz(mcm.GetErrorContentRn("您没有权限操作该机器终端,请重新登录后再试~")))
	}

	ac := GetMachineAc(rc)

	mi, err := m.MachineApp.ToMachineInfoByAc(ac)
	if err != nil {
		return
	}

	err = mi.IfUseSshTunnelChangeIpPort()
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

	query := g.Request.URL.Query()
	if query.Get("force") != "" {
		// 判断是否强制连接，是的话，查询是否有正在连接的会话，有的话强制关闭
		if cast.ToBool(query.Get("force")) {
			tn := sessions.Get(ac)
			if tn != nil {
				_ = tn.Close()
			}
		}
	}

	tunnel, err := guac.DoConnect(query, params, rc.GetLoginAccount().Username)
	if err != nil {
		return
	}
	defer func() {
		if err = tunnel.Close(); err != nil {
			logx.Warnf("Error closing tunnel: %v", err)
		}
	}()

	sessions.Add(ac, wsConn, g.Request, tunnel)

	defer sessions.Delete(ac, wsConn, g.Request, tunnel)

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
	biz.IsTrue(machineId != 0, "machineId错误")
	return uint64(machineId)
}

func GetMachineAc(rc *req.Ctx) string {
	ac := rc.PathParam("ac")
	biz.IsTrue(ac != "", "authCertName错误")
	return ac
}
