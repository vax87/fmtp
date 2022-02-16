package fmtp_states

import (
	"fmtp/channel/channel_settings"
	"fmtp/fmtp"
)

// функции, выполняемые при переходе в / (выходе из) состояния
type StateTransitionFuncMap map[fmtp.FmtpState]func(fsc *StateController, curEvent fmtp.FmtpEvent)

// функции, вызываемые при входе в состояние
func initEnterTransFuncMap(tcpRole string) StateTransitionFuncMap {
	var retValue = StateTransitionFuncMap{
		fmtp.Idle:      IdleStateEnter,
		fmtp.IdPending: IdPendingStateEnter,
		fmtp.Ready:     ReadyStateEnter,
		fmtp.Disabled:  DisabledStateEnter,
	}

	if tcpRole == channel_settings.TcpClientText {
		retValue[fmtp.ConPending] = ConnectionPendingStateEnter
	} else {
		retValue[fmtp.SysIdPending] = SystemIdPendingStateEnter
	}
	return retValue
}

// функции, вызываемые при выходе из состояние
func initExitTransFuncMap(tcpRole string) StateTransitionFuncMap {
	var retValue = StateTransitionFuncMap{
		fmtp.Idle:       IdleStateExit,
		fmtp.IdPending:  IdPendingStateExit,
		fmtp.Ready:      ReadyStateExit,
		fmtp.AssPending: AssosiationPendingStateExit,
		fmtp.DataReady:  DataReadyStateExit,
	}
	if tcpRole == channel_settings.TcpServerText {
		retValue[fmtp.SysIdPending] = SystemIdPendingStateExit
	}
	return retValue
}
