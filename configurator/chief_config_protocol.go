package configurator

import (
	"fmtp/chief/chief_settings"
	"fmtp/chief/chief_state"

	"lemz.com/fdps/utils"
)

// от конфигуратора могут быть получены сообщения:
//		- настройки контроллера(chief)
// 		- ответ на сообщени о состоянии(содержит временную метку актуальных настроек)
// конфигуратору отправляется соообщение:
//		- запрос настроек контроллера(chief)
//		- сообщение о состоянии контроллера(chief)

const (
	ChiefSettsRequestHdr = "SettingsRequest"
	ChiefSettsAnswerHdr  = "SettingsAnswer"
	ChiefHbtHdr          = "ControllerState"
	ChiefHbtAnswerHdr    = "WebServerState"
)

// ChiefHdr заголовок сообщений протокола
type ChiefHdr struct {
	Header string `json:"MessageHeader"`
}

// ChiefSettsRequestMsg сообщение запроса настроек контроллера
// контроллер (chief) -> конфигуратор
type ChiefSettsRequestMsg struct {
	ChiefHdr
	Versions    []string `json:"AvailableVersions"` // список доступных версий приложения канал
	IPAddresses []string `json:"ControllerIPs"`     // список сетевых адресов контроллера
}

func CreateChiefSettsRequestMsg(versions []string) ChiefSettsRequestMsg {
	curIPAddresses := utils.GetLocalIpv4List()
	return ChiefSettsRequestMsg{ChiefHdr: ChiefHdr{Header: ChiefSettsRequestHdr},
		Versions: versions, IPAddresses: curIPAddresses}
}

// ChiefSettsAnswerMsg сообщение c настройками контроллера
// конфигуратор -> контроллер (chief)
type ChiefSettsAnswerMsg struct {
	ChiefHdr
	chief_settings.ChiefSettings
}

// ChiefHbtMsg сообщение о состоянии контроллера
// контроллер (chief) -> конфигуратор
type ChiefHbtMsg struct {
	ChiefHdr
	chief_state.ChiefState
}

// ChiefHbtAnswerMsg сообщение - ответ на сообщение о состоянии контроллера
// конфигуратор -> контроллер (chief)
type ChiefHbtAnswerMsg struct {
	ChiefHdr
	ConfigTimestamp string `json:"ConfigTimestamp"` // временная метка акуальных настроек
	WebState        string `json:"WebServerState"`
}
