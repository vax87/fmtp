package chief_logger

import (
	"fmt"

	"fmtp/configurator"
	"fmtp/fmtp_log"

	"lemz.com/fdps/logger"
	"lemz.com/fdps/utils"
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
			////////////////////////////////////////////////////////
			tempConfFilePath := utils.AppPath() + "/config/temp_redis_settings.json"
			var tempRedisSetts RedisLoggerSettings

			if errRead := utils.ReadFromFile(tempConfFilePath, &tempRedisSetts); errRead != nil {
				logger.PrintfErr("Ошибка чтения файла конфигураций Redis. Ошибка: %v.", errRead)
				tempRedisSetts = RedisLoggerSettings{
					Hostname: "192.168.1.24",
					Port:     6389,
					DbId:     0,
					UserName: "",
					Password: "",

					StreamMaxCount: 100000,
					MaxSendCount:   1000,
				}
			}

			cl.redisLogCntrl.SettsChan <- tempRedisSetts
			/////////////////////////////////////////////////////////
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
		fmtpLogMsg.ControllerIP = configurator.ChiefCfg.IPAddr
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
