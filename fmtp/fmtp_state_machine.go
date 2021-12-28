package fmtp

// описание переходов из состояния в состояние при получении событий
type FmtpStateMachine map[FmtpState]map[FmtpState][]FmtpEvent

// инициализация StateMachine (общая для клиента и сервера часть)
func commonStateMachine(curMachine FmtpStateMachine) {
	curMachine[Ready] = map[FmtpState][]FmtpEvent{AssPending: {LStartup}}
	curMachine[Ready][Idle] = []FmtpEvent{LDisconnect, RDisconnect}
	curMachine[Ready][Ready] = []FmtpEvent{LShutdown} // from old

	curMachine[AssPending] = map[FmtpState][]FmtpEvent{Ready: {LShutdown}}
	curMachine[AssPending][DataReady] = []FmtpEvent{RStartup}
	curMachine[AssPending][Idle] = []FmtpEvent{LDisconnect, RDisconnect}
	curMachine[AssPending][AssPending] = []FmtpEvent{TiTimeout} // from old

	curMachine[DataReady] = map[FmtpState][]FmtpEvent{AssPending: {RShutdown}}
	curMachine[DataReady][Ready] = []FmtpEvent{LShutdown}
	curMachine[DataReady][Idle] = []FmtpEvent{LDisconnect, RDisconnect, TrTimeout}
	curMachine[DataReady][DataReady] = []FmtpEvent{LData, RData, RHeartbeat, TsTimeout} // from old
}

// инициализация StateMachine для клиентского соединения
func InitStateMachine(tcpRole string) FmtpStateMachine {
	var retValue FmtpStateMachine = make(map[FmtpState]map[FmtpState][]FmtpEvent)

	if tcpRole == "client" {
		retValue[Idle] = map[FmtpState][]FmtpEvent{ConPending: {LSetup}}
		retValue[Idle][Idle] = []FmtpEvent{LDisconnect, RDisconnect} // from old

		retValue[ConPending] = map[FmtpState][]FmtpEvent{IdPending: {RSetup}}
		retValue[ConPending][Idle] = []FmtpEvent{LDisconnect, RDisconnect,
			LData, LShutdown, LStartup, RData, RAccept, RReject, RHeartbeat, RShutdown, RStartup, TiTimeout} // from old

		retValue[IdPending] = map[FmtpState][]FmtpEvent{Ready: {RIdValid}}
		retValue[IdPending][Idle] = []FmtpEvent{LDisconnect, RDisconnect, RReject, RAccept,
			RIdInvalid, RData, RHeartbeat, RShutdown, RStartup, TiTimeout,
			LData, LShutdown, LStartup} //from old
	} else {
		retValue[Idle] = map[FmtpState][]FmtpEvent{SysIdPending: {RSetup}}
		retValue[Idle][Idle] = []FmtpEvent{LDisconnect, RDisconnect} // from old

		retValue[SysIdPending] = map[FmtpState][]FmtpEvent{IdPending: {RIdValid}}
		retValue[SysIdPending][Idle] = []FmtpEvent{LDisconnect, RDisconnect, RAccept,
			RReject, TiTimeout, RData, RIdInvalid, RHeartbeat, RShutdown, RStartup}
		retValue[SysIdPending][SysIdPending] = []FmtpEvent{RSetup} // from old

		retValue[IdPending] = map[FmtpState][]FmtpEvent{Ready: {RAccept}}
		retValue[IdPending][Idle] = []FmtpEvent{LDisconnect, RDisconnect, RData, RReject,
			RHeartbeat, RShutdown, RStartup, TiTimeout}
	}

	commonStateMachine(retValue)
	return retValue
}

// поиск состояние, в которое необходимо перейти по событию curEvent из состояния curState
func (sfm *FmtpStateMachine) GetNextState(curState FmtpState, curEvent FmtpEvent) FmtpState {
	if curStateVal, ok := (*sfm)[curState]; ok {
		for nextStateIt := range curStateVal {
			for _, eventIt := range curStateVal[nextStateIt] {
				if eventIt == curEvent {
					return nextStateIt
				}
			}
		}
	}
	return Empt
}
