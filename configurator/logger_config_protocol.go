package configurator

import (
	"fmtp/ora_logger/logger_settings"
	"fmtp/ora_logger/logger_state"

	"lemz.com/fdps/utils"
)

// от конфигуратора могут быть получены сообщения:
//		- настройки логгера
// 		- ответ на сообщени о состоянии(содержит временную метку актуальных настроек)
// конфигуратору отправляется соообщение:
//		- запрос настроек логгера
//		- сообщение о состоянии логгера

const (
	LoggerSettRequestHdr  = "LoggerSettingsRequest"
	LogggerSettsAnswerHdr = "LoggerSettingsAnswer"
	LoggerHbtHdr          = "LoggerState"
	LoggerHbtAnswerHdr    = "LoggerStateAnswer"
)

// LoggerHdr заголовок сообщений протокола
type LoggerHdr struct {
	Header string `json:"MessageHeader"`
}

// LoggerSettsRequestMsg сообщение запроса настроек логгер
// логгер -> конфигуратор
type LoggerSettsRequestMsg struct {
	LoggerHdr
	IPAddresses []string `json:"ControllerIPs"` // список сетевых адресов логгера
}

// CreateLoggerSettsRequestMsg формирование запроса настроек логгера
func CreateLoggerSettsRequestMsg() LoggerSettsRequestMsg {
	curIPAddresses := utils.GetLocalIpv4List()
	return LoggerSettsRequestMsg{LoggerHdr: LoggerHdr{Header: LoggerSettRequestHdr},
		IPAddresses: curIPAddresses}
}

// LoggerSettsAnswerMsg сообщение c настройками логгер
// конфигуратор -> логгер
type LoggerSettsAnswerMsg struct {
	LoggerHdr
	logger_settings.LoggerSettings
}

// LoggerHbtMsg сообщение о состоянии логгер
// логгер -> конфигуратор
type LoggerHbtMsg struct {
	LoggerHdr
	logger_state.LoggerState
}

// LoggerHbtAnswerMsg сообщение - ответ на сообщение о состоянии логгера
// конфигуратор -> логгер
type LoggerHbtAnswerMsg struct {
	LoggerHdr
	ConfigTimestamp string `json:"ConfigTimestamp"` // временная метка акуальных настроек
	WebState        string `json:"WebServerState"`
}
