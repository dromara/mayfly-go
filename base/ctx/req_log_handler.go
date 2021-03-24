package ctx

import (
	"encoding/json"
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/mlog"
	"mayfly-go/base/utils"
	"reflect"

	log "github.com/sirupsen/logrus"
)

func init() {
	// customFormatter := new(log.TextFormatter)
	// customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	// customFormatter.FullTimestamp = true
	log.SetFormatter(new(mlog.LogFormatter))
	log.SetReportCaller(true)

	AfterHandlers = append(AfterHandlers, new(LogHandler))
}

type LogHandler struct{}

func (l *LogHandler) Handler(rc *ReqCtx, err error) {
	if !rc.NeedLog {
		return
	}

	lfs := log.Fields{}
	if la := rc.LoginAccount; la != nil {
		lfs["uid"] = la.Id
		lfs["uname"] = la.Username
	}

	if err != nil {
		// lfs["errMsg"] = err.Error()

		// switch t := err.(type) {
		// case *biz.BizError:
		// 	lfs["errCode"] = t.Code()
		// 	break
		// default:
		// }
		log.WithFields(lfs).Error(getErrMsg(rc, err))
		return
	}

	// rb, _ := json.Marshal(rc.ReqParam)
	// lfs["req"] = string(rb)
	// // 返回结果不为空，则记录返回结果
	// if rc.LogResp && !utils.IsBlank(reflect.ValueOf(rc.RespObj)) {
	// 	respB, _ := json.Marshal(rc.RespObj)
	// 	lfs["resp"] = string(respB)
	// }
	log.WithFields(lfs).Info(getLogMsg(rc))
}

func getLogMsg(rc *ReqCtx) string {
	msg := rc.Description
	rb, _ := json.Marshal(rc.ReqParam)
	msg = msg + fmt.Sprintf("\n--> %s", string(rb))
	// 返回结果不为空，则记录返回结果
	if rc.LogResp && !utils.IsBlank(reflect.ValueOf(rc.RespObj)) {
		respB, _ := json.Marshal(rc.RespObj)
		msg = msg + fmt.Sprintf("\n<-- %s", string(respB))
	}
	return msg
}

func getErrMsg(rc *ReqCtx, err error) string {
	msg := rc.Description
	rb, _ := json.Marshal(rc.ReqParam)
	msg = msg + fmt.Sprintf("\n--> %s", string(rb))

	var errMsg string
	switch t := err.(type) {
	case *biz.BizError:
		errMsg = fmt.Sprintf("\n<-e errCode: %d, errMsg: %s", t.Code(), t.Error())
		break
	default:
		errMsg = fmt.Sprintf("\n<-e errMsg: %s", t.Error())
	}
	return (msg + errMsg)
}
