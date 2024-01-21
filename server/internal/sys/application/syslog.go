package application

import (
	"encoding/json"
	"fmt"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"time"
)

type Syslog interface {
	GetPageList(condition *entity.SysLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 从请求上下文的参数保存系统日志
	SaveFromReq(req *req.Ctx)
}

type syslogAppImpl struct {
	SyslogRepo repository.Syslog `inject:""`
}

func (m *syslogAppImpl) GetPageList(condition *entity.SysLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.SyslogRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (m *syslogAppImpl) SaveFromReq(req *req.Ctx) {
	lg := contextx.GetLoginAccount(req.MetaCtx)
	if lg == nil {
		lg = &model.LoginAccount{Id: 0, Username: "-"}
	}
	syslog := new(entity.SysLog)
	syslog.CreateTime = time.Now()
	syslog.Creator = lg.Username
	syslog.CreatorId = lg.Id

	logInfo := req.GetLogInfo()
	syslog.Description = logInfo.Description
	if logInfo.LogResp {
		respB, _ := json.Marshal(req.ResData)
		syslog.Resp = string(respB)
	}

	reqParam := req.ReqParam
	if !anyx.IsBlank(reqParam) {
		// 如果是字符串类型，则不使用json序列化
		if reqStr, ok := reqParam.(string); ok {
			syslog.ReqParam = reqStr
		} else {
			reqB, _ := json.Marshal(reqParam)
			syslog.ReqParam = string(reqB)
		}
	}

	if err := req.Err; err != nil {
		syslog.Type = entity.SyslogTypeError
		var errMsg string
		switch t := err.(type) {
		case errorx.BizError:
			errMsg = fmt.Sprintf("errCode: %d, errMsg: %s", t.Code(), t.Error())
		case error:
			errMsg = t.Error()
		}
		syslog.Resp = errMsg
	} else {
		syslog.Type = entity.SyslogTypeNorman
	}

	m.SyslogRepo.Insert(req.MetaCtx, syslog)
}
