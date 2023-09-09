package api

import (
	"encoding/base64"
	"fmt"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/infrastructure/machine"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/ws"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Machine struct {
	MachineApp application.Machine
	TagApp     tagapp.TagTree
}

func (m *Machine) Machines(rc *req.Ctx) {
	condition, pageParam := ginx.BindQueryAndPage(rc.GinCtx, new(entity.MachineQuery))

	// 不存在可访问标签id，即没有可操作数据
	tagIds := m.TagApp.ListTagIdByAccountId(rc.LoginAccount.Id)
	if len(tagIds) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	condition.TagIds = tagIds

	res := m.MachineApp.GetMachineList(condition, pageParam, new([]*vo.MachineVO))
	if res.Total == 0 {
		rc.ResData = res
		return
	}

	for _, mv := range *res.List {
		mv.HasCli = machine.HasCli(mv.Id)
	}
	rc.ResData = res
}

func (m *Machine) MachineTags(rc *req.Ctx) {
	rc.ResData = m.TagApp.ListTagByAccountIdAndResource(rc.LoginAccount.Id, new(entity.Machine))
}

func (m *Machine) MachineStats(rc *req.Ctx) {
	stats := m.MachineApp.GetCli(GetMachineId(rc.GinCtx)).GetAllStats()
	rc.ResData = stats
}

// 保存机器信息
func (m *Machine) SaveMachine(rc *req.Ctx) {
	machineForm := new(form.MachineForm)
	me := ginx.BindJsonAndCopyTo(rc.GinCtx, machineForm, new(entity.Machine))

	machineForm.Password = "******"
	rc.ReqParam = machineForm

	me.SetBaseInfo(rc.LoginAccount)
	m.MachineApp.Save(me)
}

func (m *Machine) TestConn(rc *req.Ctx) {
	me := ginx.BindJsonAndCopyTo(rc.GinCtx, new(form.MachineForm), new(entity.Machine))
	m.MachineApp.TestConn(me)
}

func (m *Machine) ChangeStatus(rc *req.Ctx) {
	g := rc.GinCtx
	id := uint64(ginx.PathParamInt(g, "machineId"))
	status := int8(ginx.PathParamInt(g, "status"))
	rc.ReqParam = fmt.Sprintf("id: %d -- status: %d", id, status)
	m.MachineApp.ChangeStatus(id, status)
}

func (m *Machine) DeleteMachine(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "machineId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MachineApp.Delete(uint64(value))
	}
}

// 关闭机器客户端
func (m *Machine) CloseCli(rc *req.Ctx) {
	machine.DeleteCli(GetMachineId(rc.GinCtx))
}

// 获取进程列表信息
func (m *Machine) GetProcess(rc *req.Ctx) {
	g := rc.GinCtx
	cmd := "ps -aux "
	sortType := g.Query("sortType")
	if sortType == "2" {
		cmd += "--sort -pmem "
	} else {
		cmd += "--sort -pcpu "
	}

	pname := g.Query("name")
	if pname != "" {
		cmd += fmt.Sprintf("| grep %s ", pname)
	}

	count := ginx.QueryInt(g, "count", 10)
	cmd += "| head -n " + fmt.Sprintf("%d", count)

	cli := m.MachineApp.GetCli(GetMachineId(rc.GinCtx))
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().TagPath), "%s")

	res, err := cli.Run(cmd)
	biz.ErrIsNilAppendErr(err, "获取进程信息失败: %s")
	rc.ResData = res
}

// 终止进程
func (m *Machine) KillProcess(rc *req.Ctx) {
	pid := rc.GinCtx.Query("pid")
	biz.NotEmpty(pid, "进程id不能为空")

	cli := m.MachineApp.GetCli(GetMachineId(rc.GinCtx))
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().TagPath), "%s")

	res, err := cli.Run("sudo kill -9 " + pid)
	biz.ErrIsNil(err, "终止进程失败: %s", res)
}

func (m *Machine) WsSSH(g *gin.Context) {
	wsConn, err := ws.Upgrader.Upgrade(g.Writer, g.Request, nil)
	defer func() {
		if wsConn != nil {
			if err := recover(); err != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(err.(error).Error()))
			}
			wsConn.Close()
		}
	}()

	biz.ErrIsNilAppendErr(err, "升级websocket失败: %s")
	// 权限校验
	rc := req.NewCtxWithGin(g).WithRequiredPermission(req.NewPermission("machine:terminal"))
	if err = req.PermissionHandler(rc); err != nil {
		panic(biz.NewBizErr("\033[1;31m您没有权限操作该机器终端,请重新登录后再试~\033[0m"))
	}

	cli := m.MachineApp.GetCli(GetMachineId(g))
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().TagPath), "%s")

	cols := ginx.QueryInt(g, "cols", 80)
	rows := ginx.QueryInt(g, "rows", 40)

	var recorder *machine.Recorder
	if cli.GetMachine().EnableRecorder == 1 {
		now := time.Now()
		// 回放文件路径为: 基础配置路径/机器id/操作日期/操作者账号/操作时间.cast
		recPath := fmt.Sprintf("%s/%d/%s/%s", config.Conf.Server.GetMachineRecPath(), cli.GetMachine().Id, now.Format("20060102"), rc.LoginAccount.Username)
		os.MkdirAll(recPath, 0766)
		fileName := path.Join(recPath, fmt.Sprintf("%s.cast", now.Format("20060102_150405")))
		f, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0766)
		biz.ErrIsNilAppendErr(err, "创建终端回放记录文件失败: %s")
		defer f.Close()
		recorder = machine.NewRecorder(f)
	}

	mts, err := machine.NewTerminalSession(stringx.Rand(16), wsConn, cli, rows, cols, recorder)
	biz.ErrIsNilAppendErr(err, "\033[1;31m连接失败: %s\033[0m")

	// 记录系统操作日志
	rc.WithLog(req.NewLogSave("机器-终端操作"))
	rc.ReqParam = cli.GetMachine()
	req.LogHandler(rc)

	mts.Start()
	defer mts.Stop()
}

// 获取机器终端回放记录的相应文件夹名或文件内容
func (m *Machine) MachineRecDirNames(rc *req.Ctx) {
	readPath := rc.GinCtx.Query("path")
	biz.NotEmpty(readPath, "path不能为空")
	path_ := path.Join(config.Conf.Server.GetMachineRecPath(), readPath)

	// 如果是读取文件内容，则读取对应回放记录文件内容，否则读取文件夹名列表。小小偷懒一会不想再加个接口
	isFile := rc.GinCtx.Query("isFile")
	if isFile == "1" {
		bytes, err := os.ReadFile(path_)
		biz.ErrIsNilAppendErr(err, "还未有相应终端操作记录: %s")
		rc.ResData = base64.StdEncoding.EncodeToString(bytes)
		return
	}

	files, err := os.ReadDir(path_)
	biz.ErrIsNilAppendErr(err, "还未有相应终端操作记录: %s")
	var names []string
	for _, f := range files {
		names = append(names, f.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	rc.ResData = names
}

func GetMachineId(g *gin.Context) uint64 {
	machineId, _ := strconv.Atoi(g.Param("machineId"))
	biz.IsTrue(machineId != 0, "machineId错误")
	return uint64(machineId)
}
