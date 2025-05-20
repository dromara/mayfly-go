package application

import (
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/cache"
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
	GetPageList(condition *entity.SysLogQuery, orderBy ...string) (*model.PageResult[*entity.SysLog], error)

	// 从请求上下文的参数保存系统日志
	SaveFromReq(req *req.Ctx)

	GetLogDetail(logId uint64) *entity.SysLog

	// CreateLog 创建日志信息
	CreateLog(ctx context.Context, log *CreateLogReq) (uint64, error)

	// AppendLog 追加日志信息
	AppendLog(logId uint64, appendLog *AppendLogReq)

	// SetExtra 设置指定日志的extra信息, val为空则移除该key
	SetExtra(logId uint64, key string, val any)

	// Flush 实时追加的日志到数据库里
	Flush(logId uint64, clearExtra bool)
}

var _ (Syslog) = (*syslogAppImpl)(nil)

type syslogAppImpl struct {
	syslogRepo repository.Syslog `inject:"T"`
}

func (m *syslogAppImpl) GetPageList(condition *entity.SysLogQuery, orderBy ...string) (*model.PageResult[*entity.SysLog], error) {
	return m.syslogRepo.GetPageList(condition, orderBy...)
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
		case *errorx.BizError:
			errMsg = fmt.Sprintf("errCode: %d, errMsg: %s", t.Code(), t.Error())
		case error:
			errMsg = t.Error()
		}
		syslog.Resp = errMsg
	} else {
		syslog.Type = entity.SyslogTypeSuccess
	}

	m.syslogRepo.Insert(req.MetaCtx, syslog)
}

func (m *syslogAppImpl) GetLogDetail(logId uint64) *entity.SysLog {
	syslog := m.GetCacheLog(logId)
	if syslog != nil {
		return syslog
	}
	syslog, err := m.syslogRepo.GetById(logId)
	if err != nil {
		return nil
	}
	return syslog
}

func (m *syslogAppImpl) CreateLog(ctx context.Context, log *CreateLogReq) (uint64, error) {
	syslog := new(entity.SysLog)
	structx.Copy(syslog, log)
	syslog.ReqParam = anyx.ToString(log.ReqParam)
	if len(log.Extra) > 0 {
		syslog.Extra = jsonx.ToStr(log.Extra)
	}
	if err := m.syslogRepo.Insert(ctx, syslog); err != nil {
		return 0, err
	}
	return syslog.Id, nil
}

func (m *syslogAppImpl) AppendLog(logId uint64, appendLog *AppendLogReq) {
	syslog := m.GetCacheLog(logId)
	if syslog == nil {
		sl, err := m.syslogRepo.GetById(logId)
		if err != nil {
			logx.Warnf("追加日志不存在: %d", logId)
			return
		}
		syslog = sl
	}

	appendLogMsg := fmt.Sprintf("%s %s", timex.DefaultFormat(time.Now()), appendLog.AppendResp)
	syslog.Resp = fmt.Sprintf("%s\n%s", syslog.Resp, appendLogMsg)
	syslog.Type = appendLog.Type
	if len(appendLog.Extra) > 0 {
		existExtra, _ := jsonx.ToMap(syslog.Extra)
		syslog.Extra = jsonx.ToStr(collx.MapMerge(existExtra, appendLog.Extra))
	}

	m.SetCacheLog(logId, syslog)
}

func (m *syslogAppImpl) SetExtra(logId uint64, key string, val any) {
	syslog := m.GetCacheLog(logId)
	if syslog == nil {
		sl, err := m.syslogRepo.GetById(logId)
		if err != nil {
			logx.Warnf("追加日志不存在: %d", logId)
			return
		}
		syslog = sl
	}

	extraMap, _ := jsonx.ToMap(syslog.Extra)
	if extraMap == nil {
		extraMap = make(map[string]any)
	}
	if anyx.IsBlank(val) {
		delete(extraMap, key)
	} else {
		extraMap[key] = val
	}
	syslog.Extra = jsonx.ToStr(extraMap)

	m.SetCacheLog(logId, syslog)
}

func (m *syslogAppImpl) Flush(logId uint64, clearExtra bool) {
	syslog := m.GetCacheLog(logId)
	if syslog == nil {
		return
	}

	// 如果刷入库的的时候还是执行中状态，则默认改为成功状态
	if syslog.Type == entity.SyslogTypeRunning {
		syslog.Type = entity.SyslogTypeSuccess
	}

	if clearExtra {
		syslog.Extra = ""
	}
	m.syslogRepo.UpdateById(context.Background(), syslog)
	m.DelCacheLog(logId)
}

func (m *syslogAppImpl) GetCacheLog(logId uint64) *entity.SysLog {
	log := new(entity.SysLog)
	if !cache.Get(getLogKey(logId), log) {
		return nil
	}
	return log
}

func (m *syslogAppImpl) SetCacheLog(logId uint64, log *entity.SysLog) {
	cache.Set(getLogKey(logId), log, time.Hour*1)
}

func (m *syslogAppImpl) DelCacheLog(logId uint64) {
	cache.Del(getLogKey(logId))
}

func getLogKey(logId uint64) string {
	return fmt.Sprintf("mayfly:syslog:%d", logId)
}
