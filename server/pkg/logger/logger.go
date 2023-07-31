package logger

import (
	"fmt"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func Init() {
	logger := logrus.New()
	logger.SetFormatter(new(LogFormatter))
	logger.SetReportCaller(true)

	logConf := config.Conf.Log
	// 如果不存在日志配置信息，则默认debug级别
	if logConf == nil {
		logger.SetLevel(logrus.DebugLevel)
		return
	}

	// 根据配置文件设置日志级别
	if level := logConf.Level; level != "" {
		l, err := logrus.ParseLevel(level)
		if err != nil {
			panic(fmt.Sprintf("日志级别不存在: %s", level))
		}
		logger.SetLevel(l)
	} else {
		logger.SetLevel(logrus.DebugLevel)
	}

	if logFile := logConf.File; logFile != nil {
		//写入文件
		file, err := os.OpenFile(logFile.GetFilename(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeAppend|0666)
		if err != nil {
			panic(fmt.Sprintf("创建日志文件失败: %s", err.Error()))
		}

		logger.Out = file
	}

	global.Log = logger
}

type LogFormatter struct{}

func (l *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("2006-01-02 15:04:05.000")
	level := entry.Level
	logMsg := fmt.Sprintf("%s [%s]", timestamp, strings.ToUpper(level.String()))
	// 如果存在调用信息，记录方法信息及行号
	if caller := entry.Caller; caller != nil {
		logMsg = logMsg + fmt.Sprintf(" [%s:%d]", caller.Function, caller.Line)
	}
	for k, v := range entry.Data {
		logMsg = logMsg + fmt.Sprintf(" [%s=%v]", k, v)
	}
	logMsg = logMsg + fmt.Sprintf(" : %s\n", entry.Message)
	return []byte(logMsg), nil
}
