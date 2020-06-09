package chief_configurator

import (
	"time"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/fdps"
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
	LoggerStateChan  chan common.LoggerState           // канал приема состояния логгера
	LoggerErrChan    chan error                        // канал приема ошибки подключения к логгеру
	ChannelStateChan chan []channel_state.ChannelState // список состояний fmtp каналов
	AODBStateChan    chan []fdps.ProviderState         // список состояний AODB fdps провайдеров
	HeartbeatChannel chan HeartbeatMsg

	sendTicker      *time.Ticker
	curHeartbeatMsg HeartbeatMsg
}

// NewHeartbeatController конструктор
func NewHeartbeatController() *HeartbeatController {
	return &HeartbeatController{
		LoggerStateChan:  make(chan common.LoggerState, 10),
		LoggerErrChan:    make(chan error, 10),
		ChannelStateChan: make(chan []channel_state.ChannelState, 10),
		AODBStateChan:    make(chan []fdps.ProviderState, 10),
		HeartbeatChannel: make(chan HeartbeatMsg, 10),
		sendTicker:       time.NewTicker(time.Second),

		curHeartbeatMsg: HeartbeatMsg{MessageHeader: MessageHeader{Header: HeartbeatHeader},
			CommonState: CommonStateError,
		},
	}
}

// Work рабочий цикл контроллера
func (hc *HeartbeatController) Work() {
	// work вызывается после получения настроек каналов, поэтому ChiefCfg готов к использованию
	hc.curHeartbeatMsg.CntrlID = ChiefCfg.CntrlID
	hc.curHeartbeatMsg.IPAddr = ChiefCfg.IPAddr
	hc.curHeartbeatMsg.ControllerVersion = "???"
	hc.curHeartbeatMsg.DockerVersion = DockerVersion

	hc.curHeartbeatMsg.LoggerState = common.LoggerState{
		LoggerConnected:   CommonStateError,
		LoggerDbConnected: CommonStateError,
		LoggerVersion:     "???",
	}

	for _, val := range ChiefCfg.ProvidersSetts {
		hc.curHeartbeatMsg.ProviderStates = append(hc.curHeartbeatMsg.ProviderStates,
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
		case curLogStateMgs := <-hc.LoggerStateChan:
			hc.curHeartbeatMsg.LoggerState = curLogStateMgs

		// принята ошибка подключения к логгеру
		case curErr := <-hc.LoggerErrChan:
			hc.curHeartbeatMsg.LoggerState.LoggerConnected = common.LoggerStateError
			hc.curHeartbeatMsg.LoggerState.LoggerDbError = curErr.Error()

		// принято состояние FMTP канала
		case channelMgs := <-hc.ChannelStateChan:
			hc.curHeartbeatMsg.ChannelStates = channelMgs

		// принято состояние FDPS провайдеров
		case aodbStates := <-hc.AODBStateChan:
			var resStates []fdps.ProviderState
			for _, val := range hc.curHeartbeatMsg.ProviderStates {
				if val.ProviderType == fdps.OLDIProvider {
					resStates = append(resStates, val)
				}
			}
			resStates = append(resStates, aodbStates...)
			hc.curHeartbeatMsg.ProviderStates = resStates

		// сработал таймер отправки собщения о состоянии
		case <-hc.sendTicker.C:
			hc.curHeartbeatMsg.CommonState = CommonStateOk
		CHSTL:
			for _, it := range hc.curHeartbeatMsg.ChannelStates {
				if it.FmtpState != channel_state.ChannelStateOk {
					hc.curHeartbeatMsg.CommonState = CommonStateError
					break CHSTL
				}
			}

			if hc.curHeartbeatMsg.LoggerState.LoggerConnected != common.LoggerStateOk ||
				hc.curHeartbeatMsg.LoggerState.LoggerDbConnected != common.LoggerStateOk {
				hc.curHeartbeatMsg.CommonState = CommonStateError
			}

		PVSTL:
			for _, it := range hc.curHeartbeatMsg.ProviderStates {
				if it.ProviderState != fdps.ProviderStateOk {
					hc.curHeartbeatMsg.CommonState = CommonStateError
					break PVSTL
				}
			}

			hc.HeartbeatChannel <- hc.curHeartbeatMsg
		}
	}
}
