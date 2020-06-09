package chief_channel

import (
	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/fmtp"
	"fdps/fmtp/logger/common"
)

// коды завершения приложения канала
const (
	InvalidParamCount = 1001 // приложению передано не верное кол-во аргументов
	InvalidNetPort    = 1002 // невалидное значение сетевого порта для связи с контроллером
	InvalidDaemonID   = 1003 // невалидное значение идентификатора канала
	InvalidWebPort    = 1004 // невалидное значение порта web странички
)

// от контроллера (chief) могут быть получены сообщения:
//		- настройки канала
// 		- сообщение поверх FMTP
// контроллеру(chief) отправляется соообщение:
//		- запрос настроек канала
//		- сообщение для журнала
//		- сообщение о состоянии канала
//		- сообщение поверх FMTP

const (
	// RequestSettingsHeader заголовок сообщения запроса настроек канала
	RequestSettingsHeader = "RequestSettings"

	// AnswerSettingsHeader заголовок сообщения с настройками канала
	AnswerSettingsHeader = "AnswerSettings"

	// ChannelHeartbeatHeader заголовок сообщения о состоянии канала
	ChannelHeartbeatHeader = "DaemonHeartbeat"

	// ChannelLogHeader заголовок сообщения для журнала
	ChannelLogHeader = "DaemonLog"

	// FdpsMessageHeader заголовок сообщения поверх FMTP от контроллера
	FdpsMessageHeader = "FdpsMessage"

	// ChannelMessageHeader заголовок сообщения поверх FMTP от канала
	ChannelMessageHeader = "DaemonMessage"
)

// HeaderMsg описание заголовка сообщений, получаемых от контроллера(chief)
// канал <- контроллер (chief)
type HeaderMsg struct {
	Header string `json:"MessageType"` // текст заголовка сообщения
}

// SettingsRequestMsg сообщение запроса настроек канала
// канал -> контроллер (chief)
type SettingsRequestMsg struct {
	HeaderMsg
	ChannelID int `json:"ChannelID"` // идентификатор канала
}

// CreateSettingsRequestMsg сформировать сообщение запроса настроек
func CreateSettingsRequestMsg(chID int) SettingsRequestMsg {
	return SettingsRequestMsg{HeaderMsg: HeaderMsg{Header: RequestSettingsHeader}, ChannelID: chID}
}

// SettingsAnswerMsg ответ на запроса настроек канала
// контроллер (chief) -> канал
type SettingsAnswerMsg struct {
	HeaderMsg
	channel_settings.ChannelSettings // настройки канала
}

// CreateSettingsAnswerMsg сформировать ответ на запрос настроек
func CreateSettingsAnswerMsg(chSett channel_settings.ChannelSettings) SettingsAnswerMsg {
	return SettingsAnswerMsg{HeaderMsg{Header: AnswerSettingsHeader}, chSett}
}

// ChannelHeartbeatMsg сообщение о состоянии канала
// канал -> контроллер (chief)
type ChannelHeartbeatMsg struct {
	HeaderMsg
	channel_state.ChannelState
}

// CreateChannelHeartbeatMsg сформировать сообщение о состоянии канала
func CreateChannelHeartbeatMsg(state channel_state.ChannelState) ChannelHeartbeatMsg {
	return ChannelHeartbeatMsg{HeaderMsg: HeaderMsg{Header: ChannelHeartbeatHeader}, ChannelState: state}
}

// ChannelLogMsg сообщение для журнала
// канал -> контроллер (chief)
type ChannelLogMsg struct {
	HeaderMsg
	common.LogMessage // сообщение для канала
}

// CreateChannelLogMsg сформировать сообщение для журнала
func CreateChannelLogMsg(log common.LogMessage) ChannelLogMsg {
	return ChannelLogMsg{HeaderMsg: HeaderMsg{Header: ChannelLogHeader}, LogMessage: log}
}

// DataMsg сообщение поверх FMTP
// канал <-> контроллер (chief)
type DataMsg struct {
	HeaderMsg
	ChannelID        int `json:"ChannelID"` // идентификатор канала
	fmtp.FmtpMessage     // сообщение для канала
}

// CreateChiefDataMsg сформировать сообщение поверх FMTP от контроллера
func CreateChiefDataMsg(chID int, msgText string) DataMsg {
	return DataMsg{HeaderMsg: HeaderMsg{Header: FdpsMessageHeader}, ChannelID: chID, FmtpMessage: fmtp.FmtpMessage{Type: fmtp.Operational, Text: msgText}}
}

// CreateChannelDataMsg софрмировать сообщение поверх FMTP от канала
func CreateChannelDataMsg(chID int, message fmtp.FmtpMessage) DataMsg {
	return DataMsg{HeaderMsg: HeaderMsg{Header: ChannelMessageHeader}, ChannelID: chID, FmtpMessage: message}
}
