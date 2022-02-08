package chief_logger

import (
	"fmt"

	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/fmtp_log"

	"fdps/go_utils/logger"
)

// ChiefLogger логгер, записывающий сообщения в БД
type ChiefLogger struct {
	SettsChangedChan chan struct{}
	redisLogCntrl    *RedisLogController
	minSeverity      logger.Severity
}

func NewChiefLogger() *ChiefLogger {
	return &ChiefLogger{
		SettsChangedChan: make(chan struct{}, 1),
		redisLogCntrl:    NewRedisController(),
		minSeverity:      logger.SevInfo,
	}
}

var ChiefLog = NewChiefLogger()

func (cl *ChiefLogger) Work() {
	// свой формат вывода сообщений fmtp в файловый логгер
	fmtp_log.SetUserLogFormatForLjack()

	// свой формат вывода логов на web страницу
	fmtp_log.SetUserLogFormatForWeb()

	go cl.redisLogCntrl.Run()

	for {
		select {

		case <-cl.SettsChangedChan:
			cl.redisLogCntrl.SettsChan <- RedisLoggerSettings{
				Hostname: "192.168.1.24",
				Port:     6389,
				DbId:     0,
				UserName: "",
				Password: "",

				StreamMaxCount:   1000,
				SendIntervalMSec: 20,
				MaxSendCount:     50,
			}
		}
	}
}

func (cl *ChiefLogger) processNewLogMsg(sev logger.Severity, fmtpSev string, format string, a ...interface{}) {
	if cl.minSeverity < sev {
		var fmtpLogMsg fmtp_log.LogMessage
		var ok bool
		if len(a) > 0 {
			fmtpLogMsg, ok = a[0].(fmtp_log.LogMessage)
			if !ok {
				fmtpLogMsg = fmtp_log.LogCntrlSDT(fmtpSev, fmtp_log.DirectionUnknown, fmt.Sprintf(format, a...))
			}
		} else {
			fmtpLogMsg = fmtp_log.LogCntrlSDT(fmtpSev, fmtp_log.DirectionUnknown, fmt.Sprintf(format, a...))
		}
		fmtpLogMsg.ControllerIP = chief_configurator.ChiefCfg.IPAddr
		cl.redisLogCntrl.LogMsgChan <- fmtpLogMsg
	}
}

// Printf реализация интерфейса logger
func (cl *ChiefLogger) Printf(format string, a ...interface{}) {
	cl.processNewLogMsg(logger.SevInfo, fmtp_log.SeverityInfo, format, a...)
}

// PrintfDebug реализация интерфейса logger
func (cl *ChiefLogger) PrintfDebug(format string, a ...interface{}) {
	cl.processNewLogMsg(logger.SevDebug, fmtp_log.SeverityDebug, format, a...)
}

// PrintfInfo реализация интерфейса logger
func (cl *ChiefLogger) PrintfInfo(format string, a ...interface{}) {
	cl.processNewLogMsg(logger.SevInfo, fmtp_log.SeverityInfo, format, a...)
}

// PrintfWarn реализация интерфейса logger
func (cl *ChiefLogger) PrintfWarn(format string, a ...interface{}) {
	cl.processNewLogMsg(logger.SevWarning, fmtp_log.SeverityWarning, format, a...)
}

// PrintfErr реализация интерфейса logger
func (cl *ChiefLogger) PrintfErr(format string, a ...interface{}) {
	cl.processNewLogMsg(logger.SevError, fmtp_log.SeverityInfo, format, a...)
}

// SetDebugParam задать параметр и его значение для отображение в таблице
func (cl *ChiefLogger) SetDebugParam(paramName string, paramVal string, paramColor string) {
}

// SetMinSeverity задать серъезность, начиная с которой будут вестись логи
func (cl *ChiefLogger) SetMinSeverity(sev logger.Severity) {
	cl.minSeverity = sev
}
