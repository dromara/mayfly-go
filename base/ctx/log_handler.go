package ctx

import (
	"encoding/json"
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/mlog"
	"mayfly-go/base/utils"
	"reflect"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

func init() {
	// customFormatter := new(log.TextFormatter)
	// customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	// customFormatter.FullTimestamp = true
	log.SetFormatter(new(mlog.LogFormatter))
	log.SetReportCaller(true)

	AfterHandlers = append(AfterHandlers, new(LogInfo))
}

type LogInfo struct {
	LogResp     bool   // 是否记录返回结果
	Description string // 请求描述
}

func NewLogInfo(description string) *LogInfo {
	return &LogInfo{Description: description, LogResp: false}
}

func (i *LogInfo) WithLogResp(logResp bool) *LogInfo {
	i.LogResp = logResp
	return i
}

func (l *LogInfo) AfterHandle(rc *ReqCtx) {
	li := rc.LogInfo
	if li == nil {
		return
	}

	lfs := log.Fields{}
	if la := rc.LoginAccount; la != nil {
		lfs["uid"] = la.Id
		lfs["uname"] = la.Username
	}

	req := rc.Req
	lfs[req.Method] = req.URL.Path

	if err := rc.err; err != nil {
		log.WithFields(lfs).Error(getErrMsg(rc, err))
		return
	}
	log.WithFields(lfs).Info(getLogMsg(rc))
}

func getLogMsg(rc *ReqCtx) string {
	msg := rc.LogInfo.Description + fmt.Sprintf(" ->%dms", rc.timed)
	if !utils.IsBlank(reflect.ValueOf(rc.ReqParam)) {
		rb, _ := json.Marshal(rc.ReqParam)
		msg = msg + fmt.Sprintf("\n--> %s", string(rb))
	}

	// 返回结果不为空，则记录返回结果
	if rc.LogInfo.LogResp && !utils.IsBlank(reflect.ValueOf(rc.ResData)) {
		respB, _ := json.Marshal(rc.ResData)
		msg = msg + fmt.Sprintf("\n<-- %s", string(respB))
	}
	return msg
}

func getErrMsg(rc *ReqCtx, err interface{}) string {
	msg := rc.LogInfo.Description
	if !utils.IsBlank(reflect.ValueOf(rc.ReqParam)) {
		rb, _ := json.Marshal(rc.ReqParam)
		msg = msg + fmt.Sprintf("\n--> %s", string(rb))
	}

	var errMsg string
	switch t := err.(type) {
	case *biz.BizError:
		errMsg = fmt.Sprintf("\n<-e errCode: %d, errMsg: %s", t.Code(), t.Error())
		break
	case error:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s\n%s", t.Error(), string(debug.Stack()))
		break
	case string:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s\n%s", t, string(debug.Stack()))
	}
	return (msg + errMsg)
}
