package heartbeat

import (
	"time"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/logger/common"
)

const (
	CommonStateOk    = "ok"
	CommonStateError = "error"
)

// HeartbeatController контроллер отвечает за прием сообщений о состоянии от
// логгера, провайдеров, каналов. Аккумулирует состояния и отправляет раз в секунду
// сообщение с общим состоянием контроллера/
type HeartbeatController struct {
	HeartbeatChannel chan chief_configurator.HeartbeatMsg
	sendTicker       *time.Ticker
	curHeartbeatMsg  chief_configurator.HeartbeatMsg
	dockerVersion    string
}

var HeartbeatCntrl = newHeartbeatController()

func SetDockerVersion(dockerVers string) {
	HeartbeatCntrl.dockerVersion = dockerVers
}

func SetLoggerState(loggerState common.LoggerState) {
	HeartbeatCntrl.curHeartbeatMsg.LoggerState = loggerState
}

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

func SetChannelsState(channelsState []channel_state.ChannelState) {
	HeartbeatCntrl.curHeartbeatMsg.ChannelStates = channelsState
}

// NewHeartbeatController конструктор
func newHeartbeatController() *HeartbeatController {
	return &HeartbeatController{
		HeartbeatChannel: make(chan chief_configurator.HeartbeatMsg, 10),
		sendTicker:       time.NewTicker(time.Second),
		curHeartbeatMsg: chief_configurator.HeartbeatMsg{MessageHeader: chief_configurator.MessageHeader{Header: chief_configurator.HeartbeatHeader},
			CommonState: CommonStateError,
		},
	}
}

// Work рабочий цикл контроллера
func Work() {

	// work вызывается после получения настроек каналов, поэтому ChiefCfg готов к использованию
	HeartbeatCntrl.curHeartbeatMsg.CntrlID = chief_configurator.ChiefCfg.CntrlID
	HeartbeatCntrl.curHeartbeatMsg.IPAddr = chief_configurator.ChiefCfg.IPAddr
	HeartbeatCntrl.curHeartbeatMsg.ControllerVersion = "???"
	HeartbeatCntrl.curHeartbeatMsg.DockerVersion = HeartbeatCntrl.dockerVersion

	HeartbeatCntrl.curHeartbeatMsg.LoggerState = common.LoggerState{
		LoggerConnected:   CommonStateError,
		LoggerDbConnected: CommonStateError,
		LoggerVersion:     "???",
	}

	for _, val := range chief_configurator.ChiefCfg.ProvidersSetts {
		HeartbeatCntrl.curHeartbeatMsg.ProviderStates = append(HeartbeatCntrl.curHeartbeatMsg.ProviderStates,
			fdps.ProviderState{
				ProviderID:    val.ID,
				ProviderType:  val.DataType,
				ProviderIPs:   val.IPAddresses,
				ProviderState: CommonStateError,
			})
	}

	for {
		select {
		// принято состояние логгера
		// case curLogStateMgs := <-HeartbeatCntrl.LoggerStateChan:
		// 	HeartbeatCntrl.curHeartbeatMsg.LoggerState = curLogStateMgs

		// принята ошибка подключения к логгеру
		// case curErr := <-HeartbeatCntrl.LoggerErrChan:
		// 	HeartbeatCntrl.curHeartbeatMsg.LoggerState.LoggerConnected = common.LoggerStateError
		// 	HeartbeatCntrl.curHeartbeatMsg.LoggerState.LoggerDbError = curErr.Error()

		// принято состояние FMTP канала
		// case channelMgs := <-HeartbeatCntrl.ChannelStateChan:
		// 	HeartbeatCntrl.curHeartbeatMsg.ChannelStates = channelMgs

		// принято состояние FDPS провайдеров
		// case aodbStates := <-HeartbeatCntrl.AODBStateChan:
		// 	var resStates []fdps.ProviderState
		// 	for _, val := range HeartbeatCntrl.curHeartbeatMsg.ProviderStates {
		// 		if val.ProviderType == fdps.OLDIProvider {
		// 			resStates = append(resStates, val)
		// 		}
		// 	}
		// 	resStates = append(resStates, aodbStates...)
		// 	HeartbeatCntrl.curHeartbeatMsg.ProviderStates = resStates

		// сработал таймер отправки собщения о состоянии
		case <-HeartbeatCntrl.sendTicker.C:
			HeartbeatCntrl.curHeartbeatMsg.CommonState = CommonStateOk
		CHSTL:
			for _, it := range HeartbeatCntrl.curHeartbeatMsg.ChannelStates {
				if it.FmtpState != channel_state.ChannelStateOk {
					HeartbeatCntrl.curHeartbeatMsg.CommonState = CommonStateError
					break CHSTL
				}
			}

			if HeartbeatCntrl.curHeartbeatMsg.LoggerState.LoggerConnected != common.LoggerStateOk ||
				HeartbeatCntrl.curHeartbeatMsg.LoggerState.LoggerDbConnected != common.LoggerStateOk {
				HeartbeatCntrl.curHeartbeatMsg.CommonState = CommonStateError
			}

		PVSTL:
			for _, it := range HeartbeatCntrl.curHeartbeatMsg.ProviderStates {
				if it.ProviderState != fdps.ProviderStateOk {
					HeartbeatCntrl.curHeartbeatMsg.CommonState = CommonStateError
					break PVSTL
				}
			}

			HeartbeatCntrl.HeartbeatChannel <- HeartbeatCntrl.curHeartbeatMsg
		}
	}
}
