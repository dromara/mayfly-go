package mlog

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
}

type LogFormatter struct{}

func (l *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("2006-01-02 15:04:05.000")

	level := entry.Level
	logMsg := fmt.Sprintf("%s [%s]", timestamp, strings.ToUpper(level.String()))
	// 如果存在调用信息，且为error级别以上记录文件及行号
	if caller := entry.Caller; caller != nil && level <= logrus.ErrorLevel {
		logMsg = logMsg + fmt.Sprintf(" [%s:%d]", caller.File, caller.Line)
	}
	for k, v := range entry.Data {
		logMsg = logMsg + fmt.Sprintf(" [%s=%v]", k, v)
	}
	logMsg = logMsg + fmt.Sprintf(" : %s\n", entry.Message)
	return []byte(logMsg), nil
}
