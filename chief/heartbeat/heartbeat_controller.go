package heartbeat

import (
	"time"

	"fdps/fmtp/channel/channel_state"
	log_state "fdps/fmtp/chief/chief_logger/state"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief/version"
	"fdps/fmtp/chief_configurator"
)

const (
	commonStateOk    = "ok"
	commonStateError = "error"
)

// HeartbeatController контроллер отвечает за прием сообщений о состоянии от
// логгера, провайдеров, каналов. Аккумулирует состояния и отправляет раз в секунду
// сообщение с общим состоянием контроллера/
type HeartbeatController struct {
	HeartbeatChannel chan chief_configurator.HeartbeatMsg
	sendTicker       *time.Ticker
	curHeartbeatMsg  chief_configurator.HeartbeatMsg
}

// HeartbeatCntrl - экземпляр контроллера
var HeartbeatCntrl = newHeartbeatController()

// SetDockerVersion - задать версию docker сервиса
func SetDockerVersion(dockerVers string) {
	HeartbeatCntrl.curHeartbeatMsg.DockerVersion = dockerVers
}

// SetLoggerState - задать состояние подключения к логгеру
func SetLoggerState(loggerState log_state.LoggerState) {
	HeartbeatCntrl.curHeartbeatMsg.LoggerState = loggerState
}

// SetAodbProviderState - задать состояния FDPS провайдеров
func SetAodbProviderState(aodbState []fdps.ProviderState) {
	var resStates []fdps.ProviderState
	for _, val := range HeartbeatCntrl.curHeartbeatMsg.ProviderStates {
		if val.ProviderType == fdps.OLDIProvider {
			resStates = append(resStates, val)
		}
	}
	resStates = append(resStates, aodbState...)
	HeartbeatCntrl.curHeartbeatMsg.ProviderStates = resStates
}

// SetOldiProviderState - задать состояния FDPS провайдеров
func SetOldiProviderState(oldiState []fdps.ProviderState) {
	var resStates []fdps.ProviderState
	for _, val := range HeartbeatCntrl.curHeartbeatMsg.ProviderStates {
		if val.ProviderType == fdps.AODBProvider {
			resStates = append(resStates, val)
		}
	}
	resStates = append(resStates, oldiState...)
	HeartbeatCntrl.curHeartbeatMsg.ProviderStates = resStates
}

// SetChannelsState - задать состояния FMTP каналов
func SetChannelsState(channelsState []channel_state.ChannelState) {
	HeartbeatCntrl.curHeartbeatMsg.ChannelStates = channelsState
}

// NewHeartbeatController конструктор
func newHeartbeatController() *HeartbeatController {
	return &HeartbeatController{
		HeartbeatChannel: make(chan chief_configurator.HeartbeatMsg, 10),
		sendTicker:       time.NewTicker(time.Second),
		curHeartbeatMsg: chief_configurator.HeartbeatMsg{MessageHeader: chief_configurator.MessageHeader{Header: chief_configurator.HeartbeatHeader},
			CommonState: commonStateError,
		},
	}
}

// Work рабочий цикл контроллера
func Work() {

	// work вызывается после получения настроек каналов, поэтому ChiefCfg готов к использованию
	HeartbeatCntrl.curHeartbeatMsg.CntrlID = chief_configurator.ChiefCfg.CntrlID
	HeartbeatCntrl.curHeartbeatMsg.IPAddr = chief_configurator.ChiefCfg.IPAddr
	HeartbeatCntrl.curHeartbeatMsg.ControllerVersion = version.Release
	HeartbeatCntrl.curHeartbeatMsg.DockerVersion = "???"

	HeartbeatCntrl.curHeartbeatMsg.LoggerState = log_state.LoggerState{
		LoggerConnected:   commonStateError,
		LoggerDbConnected: commonStateError,
		LoggerVersion:     "???",
	}

	for _, val := range chief_configurator.ChiefCfg.ProvidersSetts {
		HeartbeatCntrl.curHeartbeatMsg.ProviderStates = append(HeartbeatCntrl.curHeartbeatMsg.ProviderStates,
			fdps.ProviderState{
				ProviderID:    val.ID,
				ProviderType:  val.DataType,
				ProviderIPs:   val.IPAddresses,
				ProviderState: commonStateError,
			})
	}

	for {
		select {

		// сработал таймер отправки собщения о состоянии
		case <-HeartbeatCntrl.sendTicker.C:
			HeartbeatCntrl.curHeartbeatMsg.CommonState = commonStateOk
		CHSTL:
			for _, it := range HeartbeatCntrl.curHeartbeatMsg.ChannelStates {
				if it.DaemonState != channel_state.ChannelStateOk {
					HeartbeatCntrl.curHeartbeatMsg.CommonState = commonStateError
					break CHSTL
				}
			}

			if HeartbeatCntrl.curHeartbeatMsg.LoggerState.LoggerConnected != log_state.LoggerStateOk ||
				HeartbeatCntrl.curHeartbeatMsg.LoggerState.LoggerDbConnected != log_state.LoggerStateOk {
				HeartbeatCntrl.curHeartbeatMsg.CommonState = commonStateError
			}

		PVSTL:
			for _, it := range HeartbeatCntrl.curHeartbeatMsg.ProviderStates {
				if it.ProviderState != fdps.ProviderStateOk {
					HeartbeatCntrl.curHeartbeatMsg.CommonState = commonStateError
					break PVSTL
				}
			}

			HeartbeatCntrl.HeartbeatChannel <- HeartbeatCntrl.curHeartbeatMsg
		}
	}
}
