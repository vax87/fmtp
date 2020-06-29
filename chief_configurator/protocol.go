package chief_configurator

import (
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/fdps"
	"fdps/utils"
)

// от конфигуратора могут быть получены сообщения:
//		- настройки контроллера(chief)
// 		- ответ на сообщени о состоянии(содержит временную метку актуальных настроек)
// конфигуратору отправляется соообщение:
//		- запрос настроек контроллера(chief)
//		- сообщение о состоянии контроллера(chief)

const (
	// RequestSettingsHeader заголовок сообщения запроса настроек контроллера
	RequestSettingsHeader = "SettingsRequest"
	// AnswerSettingsHeader заголовок сообщения с настройками контроллера
	AnswerSettingsHeader = "SettingsAnswer"
	// HeartbeatHeader заголовок сообщения о состоянии контроллера
	HeartbeatHeader = "ControllerState"
	// HeartbeatAnswerHeader ответ на сообщение о состоянии контроллера
	HeartbeatAnswerHeader = "WebServerState"
)

// MessageHeader заголовок сообщений протокола
type MessageHeader struct {
	Header string `json:"MessageHeader"`
}

// SettingsRequestMsg сообщение запроса настроек контроллера
// контроллер (chief) -> конфигуратор
type SettingsRequestMsg struct {
	MessageHeader
	Versions    []string `json:"AvailableVersions"` // список доступных версий приложения канал
	IPAddresses []string `json:"ControllerIPs"`     // список сетевых адресов контроллера
}

// CreateSettingsRequestMsg формирование запроса настроек
func CreateSettingsRequestMsg(versions []string) SettingsRequestMsg {
	curIPAddresses := utils.GetLocalIpv4List()
	return SettingsRequestMsg{MessageHeader: MessageHeader{Header: RequestSettingsHeader},
		Versions: versions, IPAddresses: curIPAddresses}
}

// SettingsAnswerMsg сообщение c настройками контроллера
// конфигуратор -> контроллер (chief)
type SettingsAnswerMsg struct {
	MessageHeader
	chief_settings.ChiefSettings
}

// HeartbeatMsg сообщение о состоянии контроллера
// контроллер (chief) -> конфигуратор
type HeartbeatMsg struct {
	MessageHeader
	CntrlID            int                          `json:"ControllerID"` // идентификатор контроллера
	IPAddr             string                       `json:"ControllerIP"` // IP адрес контроллера
	CommonState        string                       `json:"CommonState"`
	CommonErrorMessage string                       `json:"CommonErrorMessage"`
	ControllerVersion  string                       `json:"ControllerVersion"`
	DockerVersion      string                       `json:"DockerVersion"`  // версия docker-engine
	LoggerState        common.LoggerState           `json:"LoggerState"`    // состояние логгера
	ChannelStates      []channel_state.ChannelState `json:"DaemonStates"`   // состояние FMTP каналов
	ProviderStates     []fdps.ProviderState         `json:"ProviderStates"` // состояние провайдеров
}

// HeartbeatAnswerMsg сообщение - ответ на сообщение о состоянии контроллера
// конфигуратор -> контроллер (chief)
type HeartbeatAnswerMsg struct {
	MessageHeader
	ConfigTimestamp string `json:"ConfigTimestamp"` // временная метка акуальных настроек
	WebState        string `json:"WebServerState"`
}
