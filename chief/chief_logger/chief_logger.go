package chief_logger

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_logger/common"

	"fdps/fmtp/chief/chief_logger/oracle"
	"fdps/fmtp/chief/heartbeat"
	"fdps/fmtp/chief_configurator"

	"fdps/go_utils/logger"
)

// ChiefLogger логгер, записывающий сообщения в БД
type ChiefLogger struct {
	SettsChan         chan common.LoggerSettings
	ChannelStatesChan chan channel_state.ChannelState
	oracleLogCntrl    *oracle.OracleLoggerController // контроллер записи в БД oracle
	minSeverity       logger.Severity
}

func NewChiefLogger() *ChiefLogger {
	return &ChiefLogger{
		SettsChan:         make(chan common.LoggerSettings, 1),
		ChannelStatesChan: make(chan channel_state.ChannelState, 100),
		oracleLogCntrl:    oracle.NewOracleController(),
		minSeverity:       logger.SevInfo,
	}
}

var ChiefLog = NewChiefLogger()

func (l *ChiefLogger) Work() {
	// свой формат вывода сообщений fmtp в файловый логгер
	ljackFmtpFormatFunc := func(msgIface interface{}) ([]byte, bool) {
		var msgText string
		fmtpMsg, ok := msgIface.(common.LogMessage)
		if ok {
			msgText = time.Now().UTC().Format(common.LogTimeFormat)
			msgText += "\t " + fmtpMsg.ControllerIP + "\n"
			msgText += "\t Severity: " + fmtpMsg.Severity + "\n"
			msgText += "\t Source: " + fmtpMsg.Source + "\n"
			msgText += "\t ChannelId: " + strconv.Itoa(fmtpMsg.ChannelId) + "\n"
			msgText += "\t ChannelLocName: " + fmtpMsg.ChannelLocName + "\n"
			msgText += "\t ChannelRemName: " + fmtpMsg.ChannelRemName + "\n"
			msgText += "\t DataType: " + fmtpMsg.DataType + "\n"
			msgText += "\t FmtpType: " + fmtpMsg.FmtpType + "\n"
			msgText += "\t Direction: " + fmtpMsg.Direction + "\n"
			msgText += "\t Text: " + fmtpMsg.Text + "\n"
			msgText += "\n\n"
		}
		return []byte(msgText), ok
	}
	logger.SetLjackUserFormatFunc(ljackFmtpFormatFunc)

	go l.oracleLogCntrl.Run()

	for {
		select {

		case newSetts := <-l.SettsChan:
			l.oracleLogCntrl.SettingsChan <- oracle.OracleLoggerSettings{
				Hostname:         newSetts.DbHostname,
				Port:             newSetts.DbPort,
				ServiceName:      newSetts.DbServiceName,
				UserName:         newSetts.DbUser,
				Password:         newSetts.DbPassword,
				LogStoreMaxCount: newSetts.DbMaxLogStoreCount,
				LogStoreDays:     newSetts.DbStoreDays,
			}

		case oraLoggerState := <-l.oracleLogCntrl.StateChan:
			heartbeat.SetLoggerState(oraLoggerState)

		case channelState := <-l.ChannelStatesChan:
			l.oracleLogCntrl.ChannelStatesChan <- channelState
		}
	}
}

func (l *ChiefLogger) processNewLogMsg(severity string, format string, a ...interface{}) {
	var fmtpLogMsg common.LogMessage
	var ok bool
	if len(a) > 0 {
		fmtpLogMsg, ok = a[0].(common.LogMessage)
		if !ok {
			fmtpLogMsg = common.LogCntrlSDT(severity, common.DirectionUnknown, fmt.Sprintf(format, a...))
		}
	} else {
		fmtpLogMsg = common.LogCntrlSDT(severity, common.DirectionUnknown, fmt.Sprintf(format, a...))
	}
	fmtpLogMsg.ControllerIP = chief_configurator.ChiefCfg.IPAddr

	l.oracleLogCntrl.MessChan <- fmtpLogMsg
}

// Printf реализация интерфейса logger
func (l *ChiefLogger) Printf(format string, a ...interface{}) {
	if l.minSeverity <= logger.SevInfo {
		l.processNewLogMsg(common.SeverityInfo, format, a...)
	}
}

// PrintfDebug реализация интерфейса logger
func (l *ChiefLogger) PrintfDebug(format string, a ...interface{}) {
	if l.minSeverity <= logger.SevDebug {
		l.processNewLogMsg(common.SeverityDebug, format, a...)
	}
}

// PrintfInfo реализация интерфейса logger
func (l *ChiefLogger) PrintfInfo(format string, a ...interface{}) {
	if l.minSeverity <= logger.SevInfo {
		l.processNewLogMsg(common.SeverityInfo, format, a...)
	}
}

// PrintfWarn реализация интерфейса logger
func (l *ChiefLogger) PrintfWarn(format string, a ...interface{}) {
	if l.minSeverity <= logger.SevWarning {
		l.processNewLogMsg(common.SeverityWarning, format, a...)
	}
}

// PrintfErr реализация интерфейса logger
func (l *ChiefLogger) PrintfErr(format string, a ...interface{}) {
	if l.minSeverity <= logger.SevError {
		l.processNewLogMsg(common.SeverityError, format, a...)
	}
}

// SetDebugParam задать параметр и его значение для отображение в таблице
func (l *ChiefLogger) SetDebugParam(paramName string, paramVal string, paramColor string) {
}

// SetDbStats задать параметры состояния БД
func (l *ChiefLogger) SetDbStats(dbStat sql.DBStats) {
}

// SetMinSeverity задать серъезность, начиная с которой будут вестись логи
func (l *ChiefLogger) SetMinSeverity(sev logger.Severity) {
	l.minSeverity = sev
}
