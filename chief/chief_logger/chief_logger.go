package chief_logger

import (
	"database/sql"
	"fmt"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/chief/chief_logger/file"
	"fdps/fmtp/chief/chief_logger/oracle"
	"fdps/fmtp/chief/heartbeat"
	"fdps/fmtp/chief_configurator"
	"fdps/utils/logger"
)

// ChiefLogger логгер, отправляющего сообщения по таймеру сервису fmtp_logger
type ChiefLogger struct {
	SettsChan         chan common.LoggerSettings
	FmtpLogChan       chan common.LogMessage
	ChannelStatesChan chan channel_state.ChannelState
	fileLogCntrl      *file.FileLoggerController     // контроллер записи в файлы
	oracleLogCntrl    *oracle.OracleLoggerController // контроллер записи в БД oracle
}

// NewChiefLogger - конструктор
func NewChiefLogger() *ChiefLogger {
	return &ChiefLogger{
		SettsChan:         make(chan common.LoggerSettings, 1),
		FmtpLogChan:       make(chan common.LogMessage, 10),
		ChannelStatesChan: make(chan channel_state.ChannelState, 100),
		fileLogCntrl:      file.CreateFileLoggerController(),
		oracleLogCntrl:    oracle.NewOracleController(),
	}
}

var ChiefLog = NewChiefLogger()

// Work - основной цикл работы
func (l *ChiefLogger) Work() {

	go l.fileLogCntrl.Run()
	go l.oracleLogCntrl.Run()

	for {
		select {

		case newSetts := <-l.SettsChan:
			l.fileLogCntrl.SettingsChan <- file.FileLoggerSettings{
				LogFileSizeKb:   newSetts.FileSizeKB,
				LogFolderSizeGb: newSetts.FolderSizeGB,
				LogStoreDays:    newSetts.DbStoreDays,
			}

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

		case curMsg := <-l.FmtpLogChan:
			l.processNewLogMsg(curMsg)

		case channelState := <-l.ChannelStatesChan:
			l.oracleLogCntrl.ChannelStatesChan <- channelState
		}
	}
}

// постановка сообщения в очередь.
func (l ChiefLogger) processNewLogMsg(logMsg common.LogMessage) {
	logMsg.ControllerIP = chief_configurator.ChiefCfg.IPAddr
	l.fileLogCntrl.MessChan <- logMsg

	// по требованию Калининграда, чтоб отображались только вх, исх сообщения и ошибки
	if logMsg.Severity == common.SeverityInfo || logMsg.Severity == common.SeverityWarning || logMsg.Severity == common.SeverityError {
		l.oracleLogCntrl.MessChan <- logMsg
	}
}

// Printf реализация интерфейса logger
func (l ChiefLogger) Printf(format string, a ...interface{}) {
	l.processNewLogMsg(common.LogCntrlST(common.SeverityDebug, fmt.Sprintf(format, a...)))
}

// PrintfDebug реализация интерфейса logger
func (l ChiefLogger) PrintfDebug(format string, a ...interface{}) {
	l.processNewLogMsg(common.LogCntrlST(common.SeverityDebug, fmt.Sprintf(format, a...)))
}

// PrintfInfo реализация интерфейса logger
func (l ChiefLogger) PrintfInfo(format string, a ...interface{}) {
	l.processNewLogMsg(common.LogCntrlST(common.SeverityInfo, fmt.Sprintf(format, a...)))
}

// PrintfWarn реализация интерфейса logger
func (l ChiefLogger) PrintfWarn(format string, a ...interface{}) {
	l.processNewLogMsg(common.LogCntrlST(common.SeverityWarning, fmt.Sprintf(format, a...)))
}

// PrintfErr реализация интерфейса logger
func (l ChiefLogger) PrintfErr(format string, a ...interface{}) {
	l.processNewLogMsg(common.LogCntrlST(common.SeverityError, fmt.Sprintf(format, a...)))
}

// SetDebugParam задать параметр и его значение для отображение в таблице
func (l ChiefLogger) SetDebugParam(paramName string, paramVal string, paramColor string) {
}

// SetDbStats задать параметры состояния БД
func (l ChiefLogger) SetDbStats(dbStat sql.DBStats) {
}

// SetMinSeverity задать серъезность, начиная с которой будут вестись логи
func (l ChiefLogger) SetMinSeverity(sev logger.Severity) {
}
