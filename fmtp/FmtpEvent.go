package fmtp

type FmtpEvent int

// типы событий контроллера состояний
const (
	// A command is given to establish a FMTP connection (MT-Connect-Req service primitive mapped to a T-Connect-Req).
	LSetup FmtpEvent = iota
	// A TCP transport connection establishment indication has been received by the TCP client or server (T-Connect-Ind)
	RSetup
	// Event (e.g. explicit command, application request) which causes a T-Connect Request primitive
	TcSetup
	// A command is given to stop the FMTP association, FMTP connection and the underlying TCP transport layer (MT-Disconnect-Req service primitive mapped to a T-Disconnect-Req)
	LDisconnect
	// A TCP transport connection release indication has been received (TDisconnect-Ind) or the TCP connection establishment fails
	RDisconnect
	// A Transport connection release indication has been received (T-Disconnect Indication)
	TcDisconnect
	// Indication that data (Operational, Operator, or Status message) is to be sent from the local to the remote user (MT-Data-Req mapped to a TData-Req)
	LData
	// A command is given to stop the FMTP association by transmitting a SHUTDOWN message (MT-Stop-Req service primitive mapped to TData-Req, TYP = 'System')
	LShutdown
	// A command is given to progress the FMTP association establishment by exchanging STARTUP messages (MT-Associate-Req service primitive mapped to a T-Data-Req, TYP = 'System')
	LStartup
	// Indicates that data has been received from the remote user (T-Data-Ind, TYP != 'System', TYP != 'Identification') mapped to an MT-Data-Ind
	RData
	// An ACCEPT identification message is received (T-Data-Ind, TYP = 'Identification', contents = ACCEPT)
	RAccept
	// A REJECT identification message is received (T-Data-Ind, TYP = 'Identification', contents = REJECT)
	RReject
	// An identification message is received (T-Data-Ind, TYP = 'Identification', contents != ACCEPT , contents != REJECT)
	RIdValid
	RIdInvalid
	// A HEARTBEAT message is received from the remote system (TData-Ind, TYP = 'System', Message code = HEARTBEAT mapped to an MT-Data-Ind)
	RHeartbeat
	// A SHUTDOWN message is received from the remote system (TData-Ind, TYP = 'System', Message code = SHUTDOWN mapped to an MT-Stop-Ind)
	RShutdown
	// A STARTUP message is received from the remote system (T-Data-Ind, TYP = 'System', Message code = STARTUP)
	RStartup
	// Expiry of timer Ts
	TsTimeout
	// Expiry of timer Tr
	TrTimeout
	// Expiry of timer Ti
	TiTimeout
	// останавливаем контроллер
	Disable
	// запускаем контроллер
	Enable
	/// для логов используется
	None = -1
)

var fmtpEventMap = map[FmtpEvent]string{
	LSetup:       "l_setup",
	RSetup:       "r_setup",
	TcSetup:      "tc_setup",
	LDisconnect:  "l_disconnect",
	RDisconnect:  "r_disconnect",
	TcDisconnect: "tc_disconnect",
	LData:        "l_data",
	LShutdown:    "l_shutdown",
	LStartup:     "l_startup",
	RData:        "r_data",
	RAccept:      "r_accept",
	RReject:      "r_reject",
	RIdValid:     "r_id_valid",
	RIdInvalid:   "r_id_invalid",
	RHeartbeat:   "r_heartbeat",
	RShutdown:    "r_shutdown",
	RStartup:     "r_startup",
	TsTimeout:    "ts_timeout",
	TrTimeout:    "tr_timeout",
	TiTimeout:    "ti_timeout",
	Disable:      "disable",
	Enable:       "enable",
}

func (fe *FmtpEvent) ToString() string {
	if curEventString, ok := fmtpEventMap[*fe]; ok {
		return curEventString
	}
	return "none"
}

func (fe *FmtpEvent) FromString(curEventString string) {
	for curKey, curEvent := range fmtpEventMap {
		if curEvent == curEventString {
			*fe = curKey
			return
		}
	}
	*fe = None
}
