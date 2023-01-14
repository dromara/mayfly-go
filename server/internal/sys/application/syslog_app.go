package application

import (
	"encoding/json"
	"fmt"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
	"reflect"
	"time"
)

type Syslog interface {
	GetPageList(condition *entity.Syslog, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 从请求上下文的参数保存系统日志
	SaveFromReq(req *req.Ctx)
}

func newSyslogApp(syslogRepo repository.Syslog) Syslog {
	return &syslogAppImpl{
		syslogRepo: syslogRepo,
	}
}

type syslogAppImpl struct {
	syslogRepo repository.Syslog
}

func (m *syslogAppImpl) GetPageList(condition *entity.Syslog, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return m.syslogRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (m *syslogAppImpl) SaveFromReq(req *req.Ctx) {
	lg := req.LoginAccount
	if lg == nil {
		return
	}
	syslog := new(entity.Syslog)
	syslog.CreateTime = time.Now()
	syslog.Creator = lg.Username
	syslog.CreatorId = lg.Id
	syslog.Description = req.LogInfo.Description

	if req.LogInfo.LogResp {
		respB, _ := json.Marshal(req.ResData)
		syslog.Resp = string(respB)
	}

	reqParam := req.ReqParam
	if !utils.IsBlank(reflect.ValueOf(reqParam)) {
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
		case *biz.BizError:
			errMsg = fmt.Sprintf("errCode: %d, errMsg: %s", t.Code(), t.Error())
		case error:
			errMsg = t.Error()
		}
		syslog.Resp = errMsg
	} else {
		syslog.Type = entity.SyslogTypeNorman
	}

	m.syslogRepo.Insert(syslog)
}
