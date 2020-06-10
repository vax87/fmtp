package chief_logger

import (
	"encoding/json"
	"errors"
	"fdps/fmtp/chief/heartbeat"
	"fdps/utils/logger"
	"fdps/utils/web_sock"
)

const (
	clntStateKey = "LoggerClient. Состояние:"

	clntStateOkValue    = "Подключен."
	clntStateErrorValue = "Не подключен."
)

// LoggerClient клиент взаимодействия контроллера и канала
type LoggerClient struct {
	SendDataChan chan []byte
	SettChan     chan web_sock.WebSockClientSettings
	setts        web_sock.WebSockClientSettings
	ws           *web_sock.WebSockClient
	lastWsErr    error
}

// NewLoggerClient конструктор
func NewLoggerClient(done chan struct{}) *LoggerClient {
	return &LoggerClient{
		SendDataChan: make(chan []byte, 1024),
		SettChan:     make(chan web_sock.WebSockClientSettings, 1),
		ws:           web_sock.NewWebSockClient(done),
		lastWsErr:    errors.New(""),
	}
}

func (l *LoggerClient) IsConnected() bool {
	return l.ws.IsConnected()
}

// Work реализация работы клиента
func (c *LoggerClient) Work() {
	logger.SetDebugParam(clntStateKey, "-", logger.StateDefaultColor)

	for {
		select {
		// новые настройки
		case newSetts := <-c.SettChan:
			if c.setts != newSetts {
				c.setts = newSetts
				go c.ws.Work(newSetts)
			}

		case wsErr := <-c.ws.ErrorChan:
			if c.lastWsErr.Error() != wsErr.Error() {
				c.lastWsErr = wsErr
				logger.PrintfErr("Ошибка при работе WS клиента сервиса fmtp_logger. Ошибка: %v", wsErr)
			}

		case wsData := <-c.ws.ReceiveDataChan:
			var logStateMsg LoggerStateMessage
			if unmErr := json.Unmarshal(wsData, &logStateMsg); unmErr == nil {
				heartbeat.SetLoggerState(logStateMsg.LoggerState)
			}

		case connState := <-c.ws.StateChan:
			switch connState {
			case web_sock.ClientConnected:
				logger.SetDebugParam(clntStateKey, clntStateOkValue, logger.StateOkColor)

			case web_sock.ClientDisconnected:
				logger.SetDebugParam(clntStateKey, clntStateErrorValue, logger.StateErrorColor)
			}
		}
	}
}
