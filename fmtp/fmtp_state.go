package fmtp

type FmtpState int

// состояния контроллера состояний
const (
	// No protocol instance. TCP server is bound to the TCP listening port, this corresponds to the TCP LISTEN state. TCP client can launch a TCP transport connection request
	Idle FmtpState = iota
	// The TCP client has launched the TCP 3-way handshake and waiting on FMTP connection establishment. This state only exists for outgoing FMTP association establishment.
	ConPending
	// TCP transport connection established on TCP server and awaiting remote system identification message. This state only exists for incoming FMTP association establishment.
	SysIdPending
	// TCP transport connection is established, awaiting a response for the transmitted system identification message. Actions in this state depend on whether it is an incoming/outgoing FMTP connection request.
	IdPending
	// TCP transport connection established, system identification completed, FMTP association ready to be established by local user
	Ready
	// Waiting on remote STARTUP to enter DATA_READY
	AssPending
	// Ready to exchange operational messages
	DataReady
	// выключен
	Disabled
	// не делаем перехода
	Empt = -1
)

var fmtpStateMap = map[FmtpState]string{
	Idle:         "idle",
	ConPending:   "connection_pending",
	SysIdPending: "system_id_pending",
	IdPending:    "id_pending",
	Ready:        "ready",
	AssPending:   "ass_pending",
	DataReady:    "data_ready",
	Disabled:     "disabled",
	Empt:         "empt",
}

func (fs *FmtpState) ToString() string {
	if curStateString, ok := fmtpStateMap[*fs]; ok {
		return curStateString
	}
	return ""
}

func (fs *FmtpState) FromString(curStateString string) {
	for curKey, curState := range fmtpStateMap {
		if curState == curStateString {
			*fs = curKey
			return
		}
	}
	*fs = Empt
}
