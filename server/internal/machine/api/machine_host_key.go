package api

import (
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"strings"

	"github.com/spf13/cast"
)

// MachineHostKey 主机公钥信任库管理接口。
// 信任记录由连接时自动采集（TOFU），本接口只提供查看与撤销信任：
// 撤销后下次连接重新走首次信任流程，用于主机重装/换钥后的指纹更新
type MachineHostKey struct {
	hostKeyApp application.MachineHostKey `inject:"T"`
}

func (mk *MachineHostKey) ReqConfs() *req.Confs {
	// 信任库属于安全管理面，查看与撤销均要求机器管理权限（同终端回放记录的门槛）
	saveMachineP := req.NewPermission("machine:update")

	reqs := [...]*req.Conf{
		req.NewGet("host-keys", mk.HostKeys).RequiredPermission(saveMachineP),

		req.NewDelete("host-keys/:ids", mk.DeleteHostKey).Log(req.NewLogSaveI(imsg.LogMachineHostKeyDelete)).RequiredPermission(saveMachineP),
	}

	return req.NewConfs("machines", reqs[:]...)
}

func (mk *MachineHostKey) HostKeys(rc *req.Ctx) {
	condition := rc.BindQuery[entity.MachineHostKey]()

	// 排序须走 QueryCond.OrderBy：PageByCond 的可变参数是 SELECT 列而非 order by
	res, err := mk.hostKeyApp.PageByCond(model.NewModelCond(condition).OrderBy("create_time DESC"), rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (mk *MachineHostKey) DeleteHostKey(rc *req.Ctx) {
	idsStr := rc.PathParam("ids")
	rc.ReqParam = idsStr

	for _, v := range strings.Split(idsStr, ",") {
		biz.ErrIsNil(mk.hostKeyApp.DeleteById(rc.MetaCtx, cast.ToUint64(v)))
	}
}
