package req

import (
	"fmt"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/runtimex"
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

const DefaultLogFrames = 10

func LogHandler(rc *Ctx) error {
	if rc.Conf == nil || rc.Conf.logInfo == nil {
		return nil
	}

	li := rc.Conf.logInfo

	attrMap := make(map[string]any, 0)

	req := rc.GinCtx.Request
	attrMap[req.Method] = req.URL.Path

	if la := contextx.GetLoginAccount(rc.MetaCtx); la != nil {
		attrMap["uid"] = la.Id
		attrMap["uname"] = la.Username
	}

	// 如果需要保存日志，并且保存日志处理函数存在则执行保存日志函数
	if li.save && saveLog != nil {
		go saveLog(rc)
	}

	logMsg := li.Description

	if logx.GetConfig().IsJsonType() {
		// json格式日志处理
		attrMap["req"] = rc.ReqParam
		if li.LogResp {
			attrMap["resp"] = rc.ResData
		}
		attrMap["exeTime"] = rc.timed

		if rc.Err != nil {
			nFrames := DefaultLogFrames
			if _, ok := rc.Err.(errorx.BizError); ok {
				nFrames = nFrames / 2
			}
			attrMap["error"] = rc.Err
			// 跳过log_handler等相关堆栈
			attrMap["stacktrace"] = runtimex.StatckStr(5, nFrames)
		}
	} else {
		// 处理文本格式日志信息
		if err := rc.Err; err != nil {
			logMsg = getErrMsg(rc, err)
		} else {
			logMsg = getLogMsg(rc)
		}
	}

	if err := rc.Err; err != nil {
		logx.ErrorWithFields(rc.MetaCtx, logMsg, attrMap)
		return nil
	}
	logx.InfoWithFields(rc.MetaCtx, logMsg, attrMap)
	return nil
}

func getLogMsg(rc *Ctx) string {
	logInfo := rc.Conf.logInfo
	msg := logInfo.Description + fmt.Sprintf(" ->%dms", rc.timed)
	if !anyx.IsBlank(rc.ReqParam) {
		msg = msg + fmt.Sprintf("\n--> %s", anyx.ToString(rc.ReqParam))
	}

	// 返回结果不为空，则记录返回结果
	if logInfo.LogResp && !anyx.IsBlank(rc.ResData) {
		msg = msg + fmt.Sprintf("\n<-- %s", anyx.ToString(rc.ResData))
	}
	return msg
}

func getErrMsg(rc *Ctx, err any) string {
	msg := rc.Conf.logInfo.Description + fmt.Sprintf(" ->%dms", rc.timed)
	if !anyx.IsBlank(rc.ReqParam) {
		msg = msg + fmt.Sprintf("\n--> %s", anyx.ToString(rc.ReqParam))
	}

	nFrames := DefaultLogFrames
	var errMsg string
	switch t := err.(type) {
	case errorx.BizError:
		errMsg = fmt.Sprintf("\n<-e %s", t.String())
		nFrames = nFrames / 2
	case error:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s", t.Error())
	case string:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s", t)
	}
	// 加上堆栈信息
	errMsg += fmt.Sprintf("\n<-stacktrace: %s", runtimex.StatckStr(5, nFrames))
	return (msg + errMsg)
}
