package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

func init() {
	// customFormatter := new(logrus.TextFormatter)
	// customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	// customFormatter.FullTimestamp = true
	Log.SetFormatter(new(LogFormatter))
	Log.SetReportCaller(true)
	Log.SetLevel(logrus.DebugLevel)
}

type LogFormatter struct{}

func (l *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("2006-01-02 15:04:05.000")
	level := entry.Level
	logMsg := fmt.Sprintf("%s [%s]", timestamp, strings.ToUpper(level.String()))
	// 如果存在调用信息，且为error级别以上记录文件及行号
	if caller := entry.Caller; caller != nil {
		var fp string
		// 全路径切割，只获取项目相关路径，
		// 即/Users/hml/Desktop/project/go/mayfly-go/server/test.go只获取/server/test.go
		ps := strings.Split(caller.File, "mayfly-go/")
		if len(ps) >= 2 {
			fp = ps[1]
		} else {
			fp = ps[0]
		}
		logMsg = logMsg + fmt.Sprintf(" [%s:%d]", fp, caller.Line)
	}
	for k, v := range entry.Data {
		logMsg = logMsg + fmt.Sprintf(" [%s=%v]", k, v)
	}
	logMsg = logMsg + fmt.Sprintf(" : %s\n", entry.Message)
	return []byte(logMsg), nil
}
