package chief_logger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/phf/go-queue/queue"

	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/logger/common"
	"fdps/utils/logger"
	"fdps/utils/web_sock"
)

const (
	// максимальное кол-во логов в logCache
	maxLogQueueSize = 30000

	// максимальное кол-во отправляемых сообщений по сети
	maxLogSendCount = 20
)

// ChiefLogger логгер, отправляющего сообщения по таймеру сервису fmtp_logger
type ChiefLogger struct {
	sync.Mutex
	SettsChan    chan web_sock.WebSockClientSettings
	logQueue     queue.Queue  // очередь сообщений для отправки
	loggerTicker *time.Ticker // тикер отправки сообщений логгеру
	loggerClnt   *LoggerClient
	FmtpLogChan  chan common.LogMessage
}

var chiefLogDone = make(chan struct{}, 1)
var ChiefLog = NewChiefLogger()

// NewChiefLogger - конструктор
func NewChiefLogger() *ChiefLogger {
	return &ChiefLogger{
		SettsChan:    make(chan web_sock.WebSockClientSettings, 1),
		loggerTicker: time.NewTicker(time.Second),
		loggerClnt:   NewLoggerClient(chiefLogDone),
		FmtpLogChan:  make(chan common.LogMessage, 10),
	}
}

// Work - основной цикл работы
func (l *ChiefLogger) Work() {

	go l.loggerClnt.Work()

	for {
		select {

		case newSetts := <-l.SettsChan:
			l.loggerClnt.SettChan <- newSetts

		case curMsg := <-l.FmtpLogChan:
			l.processNewLogMsg(curMsg)

		// сработал тикер отправки логов
		case <-l.loggerTicker.C:
			if l.loggerClnt.IsConnected() && l.logQueue.Len() > 0 {

				logMsg := LoggerMsgSlice{MessageHeader: MessageHeader{Header: LogMessagesHeader}}

				for nn := 0; nn < maxLogSendCount; nn++ {
					if l.logQueue.Len() > 0 {
						curLogMsg := common.LogMessage(l.logQueue.PopFront().(common.LogMessage))
						curLogMsg.ControllerIP = chief_configurator.ChiefCfg.IPAddr
						logMsg.Messages = append(logMsg.Messages, curLogMsg)
					} else {
						break
					}
				}

				if len(logMsg.Messages) > 0 {
					dataToSend, err := json.Marshal(logMsg)
					if err == nil {
						l.loggerClnt.SendDataChan <- dataToSend
					} else {
						log.Println("Ошибка маршаллинга сообщения логов. Ошибка:", err.Error())
					}
				}
			}
		}
	}
}

// постановка сообщения в очередь.
func (l ChiefLogger) processNewLogMsg(logMsg common.LogMessage) {
	l.Lock()
	defer l.Unlock()

	if l.logQueue.Len() >= maxLogQueueSize {
		l.logQueue.PopFront()
	}
	l.logQueue.PushBack(logMsg)
}

// Printf реализация интерфейса logger
func (l ChiefLogger) Printf(format string, a ...interface{}) {
	l.processNewLogMsg(common.LogCntrlST(common.SeverityInfo, fmt.Sprintf(format, a...)))
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
