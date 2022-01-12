package chief_state

import (
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_settings"
)

const (
	StateOk    = "ok"
	StateError = "error"
)

var CommonChiefState ChiefState

type LoggerState struct {
	LoggerConnected   string `json:"LoggerConnected"`   // признак наличия подключения контроллера к логгеру
	LoggerDbConnected string `json:"LoggerDbConnected"` // признак подключения логгера к БД
	LoggerDbError     string `json:"LoggerDbError"`     // текст ошибки при работе с БД
	LoggerVersion     string `json:"LoggerVersion"`     // версия логгера
}

type ProviderState struct {
	ProviderID           int      `json:"ProviderID"`           // идентификатор провайдера
	ProviderType         string   `json:"ProviderType"`         // тип провайдера (OLID | AODB)
	ProviderIPs          []string `json:"ProviderIPs"`          // список сетевых адресов провайдеров
	ProviderState        string   `json:"ProviderState"`        // состояние провайдера
	ProviderErrorMessage string   `json:"ProviderErrorMessage"` // текст ошибки
	ClientAddresses      string   `json:"-"`                    // адреса подключенных клиентов
	ProviderURL          string   `json:"-"`                    // URL web странички провайдера
	StateColor           string   `json:"-"`
}

type ChiefState struct {
	CntrlID            int                          `json:"ControllerID"` // идентификатор контроллера
	IPAddr             string                       `json:"ControllerIP"` // IP адрес контроллера
	CommonState        string                       `json:"CommonState"`
	CommonErrorMessage string                       `json:"CommonErrorMessage"`
	ControllerVersion  string                       `json:"ControllerVersion"`
	DockerVersion      string                       `json:"DockerVersion"`  // версия docker-engine
	LoggerState        LoggerState                  `json:"LoggerState"`    // состояние логгера
	ChannelStates      []channel_state.ChannelState `json:"DaemonStates"`   // состояние FMTP каналов
	ProviderStates     []ProviderState              `json:"ProviderStates"` // состояние провайдеров
}

func SetDockerVersion(dockerVers string) {
	CommonChiefState.DockerVersion = dockerVers
}

func SetLoggerState(loggerState LoggerState) {
	CommonChiefState.LoggerState = loggerState
	checkCommonState()
}

func SetAodbProviderState(aodbState []ProviderState) {
	var resStates []ProviderState
	for _, val := range CommonChiefState.ProviderStates {
		if val.ProviderType == chief_settings.OLDIProvider {
			resStates = append(resStates, val)
		}
	}
	resStates = append(resStates, aodbState...)
	CommonChiefState.ProviderStates = resStates
	checkCommonState()
}

func SetOldiProviderState(oldiState []ProviderState) {
	var resStates []ProviderState
	for _, val := range CommonChiefState.ProviderStates {
		if val.ProviderType == chief_settings.AODBProvider {
			resStates = append(resStates, val)
		}
	}
	resStates = append(resStates, oldiState...)
	CommonChiefState.ProviderStates = resStates
	checkCommonState()
}

func SetChannelsState(channelsState []channel_state.ChannelState) {
	CommonChiefState.ChannelStates = channelsState
	checkCommonState()
}

func checkCommonState() {
	CommonChiefState.CommonState = StateOk

	if CommonChiefState.LoggerState.LoggerConnected != StateOk ||
		CommonChiefState.LoggerState.LoggerDbConnected != StateOk {
		CommonChiefState.CommonState = StateError
		return
	}

	for _, it := range CommonChiefState.ChannelStates {
		if it.DaemonState != channel_state.ChannelStateOk {
			CommonChiefState.CommonState = StateError
			return
		}
	}

	for _, it := range CommonChiefState.ProviderStates {
		if it.ProviderState != StateOk {
			CommonChiefState.CommonState = StateError
			return
		}
	}
}
