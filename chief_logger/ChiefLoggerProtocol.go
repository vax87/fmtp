package chief_logger

import (
	"fdps/fmtp/logger/common"
)

// логгеру от контроллера отправляется соообщения:
//		- настройки логгера
// 	    - массив логов для записи в БД
// контроллеру от логгера отправляется соообщения:
//		- сообщение о состоянии логгера
const (
	LoggerSettingsHeader = "NeedUpdateSettingsHeader" // заголовок сообщения c настройками логгера
	LoggerStateHeader    = "LoggerStateHeader"        // заголовок сообщения о состоянии логера
	LogMessagesHeader    = "LogMessageVectorHeader"   // заголовок сообщения массива логов
)

// заголовок сообщений протокола
type MessageHeader struct {
	Header string `json:"MessageHeader"`
}

// сообщение с настройками логгера
// контроллер (chief) -> логгер
type LoggerSettingsMsg struct {
	MessageHeader
	common.LoggerSettings
}

// сообщение с массивом логов
// контроллер (chief) -> логгер
type LoggerMsgSlice struct {
	MessageHeader
	Messages []common.LogMessage `json:"LogMessageVector"` // массив сообщений
}

// сообщение о состоянии логгера
// логгер -> контроллер (chief)
type LoggerStateMessage struct {
	MessageHeader
	common.LoggerState
}

func CreateLoggerStateMsg(err error, version string) LoggerStateMessage {
	retValue := LoggerStateMessage{MessageHeader: MessageHeader{Header: LoggerStateHeader}}
	retValue.LoggerConnected = common.LoggerStateOk
	retValue.LoggerVersion = version

	if err == nil {
		retValue.LoggerDbError = ""
		retValue.LoggerDbConnected = common.LoggerStateOk
	} else {
		retValue.LoggerDbError = err.Error()
		retValue.LoggerDbConnected = common.LoggerStateError
	}
	return retValue
}
