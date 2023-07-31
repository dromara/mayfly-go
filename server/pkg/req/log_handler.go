package req

import (
	"fmt"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/stringx"

	"github.com/sirupsen/logrus"
)

type SaveLogFunc func(*Ctx)

var saveLog SaveLogFunc

// 设置保存日志处理函数
func SetSaveLogFunc(sl SaveLogFunc) {
	saveLog = sl
}

type LogInfo struct {
	Description string // 请求描述

	LogResp bool // 是否记录返回结果
	save    bool // 是否保存日志
}

// 新建日志信息，默认不保存该日志
func NewLog(description string) *LogInfo {
	return &LogInfo{Description: description, LogResp: false, save: false}
}

// 新建日志信息,并且需要保存该日志信息
func NewLogSave(description string) *LogInfo {
	return &LogInfo{Description: description, LogResp: false, save: true}
}

// 记录返回结果
func (i *LogInfo) WithLogResp() *LogInfo {
	i.LogResp = true
	return i
}

func LogHandler(rc *Ctx) error {
	if rc.Conf == nil || rc.Conf.logInfo == nil {
		return nil
	}

	li := rc.Conf.logInfo

	lfs := logrus.Fields{}
	if la := rc.LoginAccount; la != nil {
		lfs["uid"] = la.Id
		lfs["uname"] = la.Username
	}

	req := rc.GinCtx.Request
	lfs[req.Method] = req.URL.Path

	// 如果需要保存日志，并且保存日志处理函数存在则执行保存日志函数
	if li.save && saveLog != nil {
		go saveLog(rc)
	}
	if err := rc.Err; err != nil {
		global.Log.WithFields(lfs).Error(getErrMsg(rc, err))
		return nil
	}
	global.Log.WithFields(lfs).Info(getLogMsg(rc))
	return nil
}

func getLogMsg(rc *Ctx) string {
	logInfo := rc.Conf.logInfo
	msg := logInfo.Description + fmt.Sprintf(" ->%dms", rc.timed)
	if !anyx.IsBlank(rc.ReqParam) {
		msg = msg + fmt.Sprintf("\n--> %s", stringx.AnyToStr(rc.ReqParam))
	}

	// 返回结果不为空，则记录返回结果
	if logInfo.LogResp && !anyx.IsBlank(rc.ResData) {
		msg = msg + fmt.Sprintf("\n<-- %s", stringx.AnyToStr(rc.ResData))
	}
	return msg
}

func getErrMsg(rc *Ctx, err any) string {
	msg := rc.Conf.logInfo.Description
	if !anyx.IsBlank(rc.ReqParam) {
		msg = msg + fmt.Sprintf("\n--> %s", stringx.AnyToStr(rc.ReqParam))
	}

	var errMsg string
	switch t := err.(type) {
	case biz.BizError:
		errMsg = fmt.Sprintf("\n<-e errCode: %d, errMsg: %s", t.Code(), t.Error())
	case error:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s", t.Error())
	case string:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s", t)
	}
	return (msg + errMsg)
}
