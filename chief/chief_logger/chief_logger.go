package chief_logger

import (
	"fmt"

	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/fmtp_logger"

	"fdps/fmtp/chief/chief_settings"

	"fdps/go_utils/logger"
)

// ChiefLogger логгер, записывающий сообщения в БД
type ChiefLogger struct {
	SettsChan chan chief_settings.LoggerSettings
	//oracleLogCntrl *OracleLoggerController // контроллер записи в БД oracle
	redisLogCntrl *RedisLogController
	minSeverity   logger.Severity
}

func NewChiefLogger() *ChiefLogger {
	return &ChiefLogger{
		SettsChan: make(chan chief_settings.LoggerSettings, 1),
		//oracleLogCntrl: NewOracleController(),
		redisLogCntrl: NewRedisController(),
		minSeverity:   logger.SevInfo,
	}
}

var ChiefLog = NewChiefLogger()

func (cl *ChiefLogger) Work() {
	// свой формат вывода сообщений fmtp в файловый логгер
	fmtp_logger.SetUserLogFormatForLjack()

	// свой формат вывода логов на web страницу
	fmtp_logger.SetUserLogFormatForWeb()

	//go cl.oracleLogCntrl.Run()
	go cl.redisLogCntrl.Run()

	for {
		select {

		case <-cl.SettsChan:
			// cl.oracleLogCntrl.SettingsChan <- OracleLoggerSettings{
			// 	Hostname:         newSetts.DbHostname,
			// 	Port:             newSetts.DbPort,
			// 	ServiceName:      newSetts.DbServiceName,
			// 	UserName:         newSetts.DbUser,
			// 	Password:         newSetts.DbPassword,
			// 	LogStoreMaxCount: newSetts.DbMaxLogStoreCount,
			// 	LogStoreDays:     newSetts.DbStoreDays,
			// }

			cl.redisLogCntrl.SettsChan <- RedisLoggerSettings{
				Hostname: "192.168.1.24",
				Port:     6380,
				DbId:     0,
				UserName: "",
				Password: "",

				StreamMaxCount:   100000000,
				SendIntervalMSec: 50,
				MaxSendCount:     50,
			}

			// case oraLoggerState := <-cl.oracleLogCntrl.StateChan:
			// 	chief_state.SetLoggerState(oraLoggerState)
		}
	}
}

func (cl *ChiefLogger) processNewLogMsg(severity string, format string, a ...interface{}) {
	var fmtpLogMsg fmtp_logger.LogMessage
	var ok bool
	if len(a) > 0 {
		fmtpLogMsg, ok = a[0].(fmtp_logger.LogMessage)
		if !ok {
			fmtpLogMsg = fmtp_logger.LogCntrlSDT(severity, fmtp_logger.DirectionUnknown, fmt.Sprintf(format, a...))
		}
	} else {
		fmtpLogMsg = fmtp_logger.LogCntrlSDT(severity, fmtp_logger.DirectionUnknown, fmt.Sprintf(format, a...))
	}
	fmtpLogMsg.ControllerIP = chief_configurator.ChiefCfg.IPAddr

	//cl.oracleLogCntrl.MessChan <- fmtpLogMsg
	cl.redisLogCntrl.LogMsgChan <- fmtpLogMsg
}

// Printf реализация интерфейса logger
func (cl *ChiefLogger) Printf(format string, a ...interface{}) {
	if cl.minSeverity <= logger.SevInfo {
		cl.processNewLogMsg(fmtp_logger.SeverityInfo, format, a...)
	}
}

// PrintfDebug реализация интерфейса logger
func (cl *ChiefLogger) PrintfDebug(format string, a ...interface{}) {
	if cl.minSeverity <= logger.SevDebug {
		cl.processNewLogMsg(fmtp_logger.SeverityDebug, format, a...)
	}
}

// PrintfInfo реализация интерфейса logger
func (cl *ChiefLogger) PrintfInfo(format string, a ...interface{}) {
	if cl.minSeverity <= logger.SevInfo {
		cl.processNewLogMsg(fmtp_logger.SeverityInfo, format, a...)
	}
}

// PrintfWarn реализация интерфейса logger
func (cl *ChiefLogger) PrintfWarn(format string, a ...interface{}) {
	if cl.minSeverity <= logger.SevWarning {
		cl.processNewLogMsg(fmtp_logger.SeverityWarning, format, a...)
	}
}

// PrintfErr реализация интерфейса logger
func (cl *ChiefLogger) PrintfErr(format string, a ...interface{}) {
	if cl.minSeverity <= logger.SevError {
		cl.processNewLogMsg(fmtp_logger.SeverityError, format, a...)
	}
}

// SetDebugParam задать параметр и его значение для отображение в таблице
func (cl *ChiefLogger) SetDebugParam(paramName string, paramVal string, paramColor string) {
}

// SetMinSeverity задать серъезность, начиная с которой будут вестись логи
func (cl *ChiefLogger) SetMinSeverity(sev logger.Severity) {
	cl.minSeverity = sev
}

func (cl *ChiefLogger) SetWriteStatesToDb(writeState bool) {
	//cl.oracleLogCntrl.SetWriteStatesToDb(writeState)
}
