package application

import (
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/structx"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"time"
)

type CreateLogReq struct {
	Type        int8           `json:"type"`
	Description string         `json:"description"`
	ReqParam    any            `json:"reqParam" ` // 请求参数
	Resp        string         `json:"resp" `     // 响应结构
	Extra       map[string]any // 额外日志信息
}

type AppendLogReq struct {
	Type       int8           `json:"type"`
	AppendResp string         `json:"appendResp" ` // 追加日志信息
	Extra      map[string]any // 额外日志信息
}

type Syslog interface {
	GetPageList(condition *entity.SysLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 从请求上下文的参数保存系统日志
	SaveFromReq(req *req.Ctx)

	GetLogDetail(logId uint64) *entity.SysLog

	// CreateLog 创建日志信息
	CreateLog(ctx context.Context, log *CreateLogReq) (uint64, error)

	// AppendLog 追加日志信息
	AppendLog(logId uint64, appendLog *AppendLogReq)

	// Flush 实时追加的日志到库里
	Flush(logId uint64)
}

type syslogAppImpl struct {
	SyslogRepo repository.Syslog `inject:""`

	appendLogs map[uint64]*entity.SysLog
	rwLock     sync.RWMutex
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
	now := time.Now()
	syslog.CreateTime = &now
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

	if err := req.Error; err != nil {
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
		syslog.Type = entity.SyslogTypeSuccess
	}

	m.SyslogRepo.Insert(req.MetaCtx, syslog)
}

func (m *syslogAppImpl) GetLogDetail(logId uint64) *entity.SysLog {
	syslog := new(entity.SysLog)
	if err := m.SyslogRepo.GetById(syslog, logId); err != nil {
		return nil
	}

	if syslog.Type == entity.SyslogTypeRunning {
		m.rwLock.RLock()
		defer m.rwLock.RUnlock()
		return m.appendLogs[logId]
	}

	return syslog
}

func (m *syslogAppImpl) CreateLog(ctx context.Context, log *CreateLogReq) (uint64, error) {
	syslog := new(entity.SysLog)
	structx.Copy(syslog, log)
	syslog.ReqParam = anyx.ToString(log.ReqParam)
	if log.Extra != nil {
		syslog.Extra = jsonx.ToStr(log.Extra)
	}
	if err := m.SyslogRepo.Insert(ctx, syslog); err != nil {
		return 0, err
	}
	return syslog.Id, nil
}

func (m *syslogAppImpl) AppendLog(logId uint64, appendLog *AppendLogReq) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	if m.appendLogs == nil {
		m.appendLogs = make(map[uint64]*entity.SysLog)
	}

	syslog := m.appendLogs[logId]
	if syslog == nil {
		syslog = new(entity.SysLog)
		if err := m.SyslogRepo.GetById(syslog, logId); err != nil {
			logx.Warnf("追加日志不存在: %d", logId)
			return
		}
		m.appendLogs[logId] = syslog
	}

	appendLogMsg := fmt.Sprintf("%s %s", timex.DefaultFormat(time.Now()), appendLog.AppendResp)
	syslog.Resp = fmt.Sprintf("%s\n%s", syslog.Resp, appendLogMsg)
	syslog.Type = appendLog.Type
	if appendLog.Extra != nil {
		existExtra := jsonx.ToMap(syslog.Extra)
		syslog.Extra = jsonx.ToStr(collx.MapMerge(existExtra, appendLog.Extra))
	}
}

func (m *syslogAppImpl) Flush(logId uint64) {
	syslog := m.appendLogs[logId]
	if syslog == nil {
		return
	}

	// 如果刷入库的的时候还是执行中状态，则默认改为成功状态
	if syslog.Type == entity.SyslogTypeRunning {
		syslog.Type = entity.SyslogTypeSuccess
	}

	m.SyslogRepo.UpdateById(context.Background(), syslog)
	delete(m.appendLogs, logId)
}
