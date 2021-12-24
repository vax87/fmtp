package fmtp_states

import (
	"time"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/fmtp"
	"fdps/fmtp/fmtp_logger"
)

// ----------------------------состояния IDLE----------------------------
//	No protocol instance exists. TCP server is bound to the TCP listening port;
//	this corresponds to the TCP LISTEN state. TCP client can launch a TCP transport connection request

func IdleStateEnter(fsc *StateController, eventType fmtp.FmtpEvent) {
	//stateCntrl_->getTransport()->stop();
	//reconnectTimer_->start(reconnectTimeout_);
	time.AfterFunc(time.Duration(fsc.curSet.ReconnectTimeout)*time.Second, func() {
		fsc.tcpTransport.ReconnectChan() <- struct{}{}
	})
}

func IdleStateExit(fsc *StateController, eventType fmtp.FmtpEvent) {

	//reconnectTimer_->stop();
}

// ----------------------------состояния SYSTEM_ID_PENDING----------------------------
//	TCP transport connection established on TCP server and awaiting remote system identification message.
//	This state is applicable to the MT-Responder system only.
func SystemIdPendingStateEnter(fsc *StateController, eventType fmtp.FmtpEvent) {
	fsc.tiTimer.restartTimer()
}

func SystemIdPendingStateExit(fsc *StateController, eventType fmtp.FmtpEvent) {
	if eventType == fmtp.RIdValid {
		if fsc.curSet.NetRole == channel_settings.TcpServerText {
			fsc.sendPacket(fsc.ownIdentificationMsg, fmtp.None, fmtp_logger.SeverityInfo)
		}
		fsc.tiTimer.restartTimer()
	} else {
		fsc.tiTimer.stopTimer()
	}
	if eventType == fmtp.TiTimeout {
		//stateCntrl_->getTransport()->stopWaitIdent();
	}
}

// ----------------------------состояния CONNECTION_PENDING----------------------------
//  The TCP client has launched the TCP 3-way handshake and is waiting for
//	establishment of an FMTP connection. This state is applicable to the MT-Initiator system only.
func ConnectionPendingStateEnter(fsc *StateController, eventType fmtp.FmtpEvent) {
	fsc.sendPacket(fsc.ownIdentificationMsg, fmtp.RSetup, fmtp_logger.SeverityInfo)
}

// ----------------------------состояния ID_PENDING----------------------------
//	TCP transport connection is established, awaiting a response for the transmitted system identification message.
func IdPendingStateEnter(fsc *StateController, eventType fmtp.FmtpEvent) {
	fsc.tiTimer.restartTimer()
}

func IdPendingStateExit(fsc *StateController, eventType fmtp.FmtpEvent) {
	if eventType == fmtp.RIdValid {
		fsc.sendPacket(fmtp.AcceptMessage, fmtp.None, fmtp_logger.SeverityInfo)
	} else if eventType == fmtp.RIdInvalid {
		fsc.sendPacket(fmtp.RejectMessage, fmtp.None, fmtp_logger.SeverityInfo)
	}
	fsc.tiTimer.stopTimer()
}

// ----------------------------состояния READY----------------------------
//	TCP transport connection is established, system identification completed,
//	FMTP Association ready to be established by local user.
func ReadyStateEnter(fsc *StateController, eventType fmtp.FmtpEvent) {
	fsc.sendPacket(fmtp.StartupMessage, fmtp.LStartup, fmtp_logger.SeverityInfo)
}

func ReadyStateExit(fsc *StateController, eventType fmtp.FmtpEvent) {
	if eventType == fmtp.LStartup {
		fsc.trTimer.restartTimer()
	}
}

// ----------------------------ASSOTIATION_PENDING----------------------------
//	Waiting  for  remote  STARTUP  to  enter	DATA_READY.
func AssosiationPendingStateExit(fsc *StateController, eventType fmtp.FmtpEvent) {

	if eventType == fmtp.LDisconnect || eventType == fmtp.LShutdown {
		fsc.sendPacket(fmtp.ShutdownMessage, fmtp.None, fmtp_logger.SeverityInfo)
		fsc.trTimer.stopTimer()
	} else if eventType == fmtp.TrTimeout {
		fsc.sendPacket(fmtp.StartupMessage, fmtp.LStartup, fmtp_logger.SeverityInfo)
		fsc.trTimer.restartTimer()
	} else if eventType == fmtp.RStartup {
		fsc.sendPacket(fmtp.StartupMessage, fmtp.LStartup, fmtp_logger.SeverityInfo)
		fsc.trTimer.restartTimer()
		fsc.tsTimer.restartTimer()
	} else {
		fsc.trTimer.stopTimer()
	}
}

// ----------------------------DATA_READY----------------------------
//	Ready to exchange operational messages.
func DataReadyStateExit(fsc *StateController, eventType fmtp.FmtpEvent) {
	if eventType == fmtp.LDisconnect || eventType == fmtp.LShutdown {
		fsc.sendPacket(fmtp.ShutdownMessage, fmtp.None, fmtp_logger.SeverityInfo)
		fsc.trTimer.stopTimer()
		fsc.tsTimer.stopTimer()
	} else if eventType == fmtp.RDisconnect || eventType == fmtp.TrTimeout {
		fsc.trTimer.stopTimer()
		fsc.tsTimer.stopTimer()
	} else if eventType == fmtp.LData {
		fsc.tsTimer.restartTimer()
	} else if eventType == fmtp.TsTimeout {
		fsc.sendPacket(fmtp.HeartbeatMessage, fmtp.None, fmtp_logger.SeverityDebug)
		fsc.tsTimer.restartTimer()
	} else if eventType == fmtp.RData || eventType == fmtp.RHeartbeat {
		fsc.trTimer.restartTimer()
	} else if eventType == fmtp.RShutdown {
		fsc.tsTimer.stopTimer()
		fsc.trTimer.restartTimer()
	}
}

// ----------------------------DISABLED----------------------------
func DisabledStateEnter(fsc *StateController, eventType fmtp.FmtpEvent) {
	//stateCntrl_->getTransport()->stop();
	fsc.tiTimer.stopTimer()
	fsc.trTimer.stopTimer()
	fsc.tsTimer.stopTimer()

}
