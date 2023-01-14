package req

import (
	"fmt"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logger"
	"mayfly-go/pkg/utils"
	"reflect"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

type SaveLogFunc func(*Ctx)

var saveLog SaveLogFunc

// 设置保存日志处理函数
func SetSaveLogFunc(sl SaveLogFunc) {
	saveLog = sl
}

type LogInfo struct {
	LogResp     bool   // 是否记录返回结果
	Description string // 请求描述
	Save        bool   // 是否保存日志
}

// 新建日志信息
func NewLogInfo(description string) *LogInfo {
	return &LogInfo{Description: description, LogResp: false}
}

// 是否记录返回结果
func (i *LogInfo) WithLogResp(logResp bool) *LogInfo {
	i.LogResp = logResp
	return i
}

// 是否保存日志
func (i *LogInfo) WithSave(saveLog bool) *LogInfo {
	i.Save = saveLog
	return i
}

func LogHandler(rc *Ctx) error {
	li := rc.LogInfo
	if li == nil {
		return nil
	}

	lfs := logrus.Fields{}
	if la := rc.LoginAccount; la != nil {
		lfs["uid"] = la.Id
		lfs["uname"] = la.Username
	}

	req := rc.GinCtx.Request
	lfs[req.Method] = req.URL.Path

	// 如果需要保存日志，并且保存日志处理函数存在则执行保存日志函数
	if li.Save && saveLog != nil {
		go saveLog(rc)
	}
	if err := rc.Err; err != nil {
		logger.Log.WithFields(lfs).Error(getErrMsg(rc, err))
		return nil
	}
	logger.Log.WithFields(lfs).Info(getLogMsg(rc))
	return nil
}

func getLogMsg(rc *Ctx) string {
	msg := rc.LogInfo.Description + fmt.Sprintf(" ->%dms", rc.timed)
	if !utils.IsBlank(reflect.ValueOf(rc.ReqParam)) {
		msg = msg + fmt.Sprintf("\n--> %s", utils.ToString(rc.ReqParam))
	}

	// 返回结果不为空，则记录返回结果
	if rc.LogInfo.LogResp && !utils.IsBlank(reflect.ValueOf(rc.ResData)) {
		msg = msg + fmt.Sprintf("\n<-- %s", utils.ToString(rc.ResData))
	}
	return msg
}

func getErrMsg(rc *Ctx, err interface{}) string {
	msg := rc.LogInfo.Description
	if !utils.IsBlank(reflect.ValueOf(rc.ReqParam)) {
		msg = msg + fmt.Sprintf("\n--> %s", utils.ToString(rc.ReqParam))
	}

	var errMsg string
	switch t := err.(type) {
	case biz.BizError:
		errMsg = fmt.Sprintf("\n<-e errCode: %d, errMsg: %s", t.Code(), t.Error())
	case error:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s\n%s", t.Error(), string(debug.Stack()))
	case string:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s\n%s", t, string(debug.Stack()))
	}
	return (msg + errMsg)
}
