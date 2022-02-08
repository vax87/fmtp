package fmtp_states

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/channel/tcp_transport"
	"fdps/fmtp/fmtp"
	"fdps/fmtp/fmtp_log"
	"fdps/go_utils/logger"

	"fdps/utils"
)

// контроллер переходов в FMTP состояния
type StateController struct {
	tcpTransport tcp_transport.TcpTransport // TPC транспорт

	tiTimer *Timer // таймер Ti (для идентификации)
	tsTimer *Timer // таймер Ts (для отправки)
	trTimer *Timer // таймер Tr (для приема)

	currentState  fmtp.FmtpState                  // текущее FMTP состояние
	FmtpStateChan chan channel_state.ChannelState // канал для отправки текущего состояния
	stateTick     *time.Ticker                    // тикер для отправки сообщений текущего состояния

	curSet channel_settings.ChannelSettings // настройки FMTP канала

	stateMachine fmtp.FmtpStateMachine // правила перехода из состояния в состояние

	stateEnterFuncMap StateTransitionFuncMap // функции, выполняемые при входе в состояние
	stateExitFuncMap  StateTransitionFuncMap //функции, выполняемые при выходе из состояние

	ownIdentificationMsg    fmtp.FmtpMessage // собственное идентификационное сообщение
	remoteIdentificationMsg fmtp.FmtpMessage // ожидаемое идентификационное сообщение

	LogMessageChan      chan fmtp_log.LogMessage // канал для передачи сообщений для журнала
	FmtpDataReceiveChan chan fmtp.FmtpMessage    // канал для отправки данных полученных поверх FMTP
	FmtpDataSendChan    chan fmtp.FmtpMessage    // канал для приема данных полученных поверх FMTP

	receivedBuffer bytes.Buffer // буфер полученных из TCP транспорта данных
}

// конструктор
func NewStateController() *StateController {
	return &StateController{
		currentState:        fmtp.Idle,
		FmtpStateChan:       make(chan channel_state.ChannelState),
		stateTick:           time.NewTicker(channel_state.StateSendInterval),
		LogMessageChan:      make(chan fmtp_log.LogMessage, 100),
		FmtpDataReceiveChan: make(chan fmtp.FmtpMessage, 1024),
		FmtpDataSendChan:    make(chan fmtp.FmtpMessage, 1024),
	}
}

// запуск работы контроллера
func (fsc *StateController) Work(settings channel_settings.ChannelSettings) {
	fsc.curSet = settings

	if fsc.curSet.NetRole == channel_settings.TcpClientText {
		fsc.tcpTransport = tcp_transport.NewFmtpTcpClient()
	} else {
		fsc.tcpTransport = tcp_transport.NewFmtpTcpServer()
	}

	fsc.tiTimer = newFmtpTimer(time.Duration(fsc.curSet.IntervalTi)*time.Second, fmtp.TiTimeout)
	fsc.tsTimer = newFmtpTimer(time.Duration(fsc.curSet.IntervalTs)*time.Second, fmtp.TsTimeout)
	fsc.trTimer = newFmtpTimer(time.Duration(fsc.curSet.IntervalTr)*time.Second, fmtp.TrTimeout)

	fsc.stateMachine = fmtp.InitStateMachine(fsc.curSet.NetRole)
	fsc.stateEnterFuncMap = initEnterTransFuncMap(fsc.curSet.NetRole)
	fsc.stateExitFuncMap = initExitTransFuncMap(fsc.curSet.NetRole)
	fsc.ownIdentificationMsg = fmtp.CreateIdentificationMessage(fsc.curSet.LocalATC, fsc.curSet.RemoteATC, true)
	fsc.remoteIdentificationMsg = fmtp.CreateIdentificationMessage(fsc.curSet.LocalATC, fsc.curSet.RemoteATC, false)

	go fsc.tcpTransport.Work()
	if fsc.curSet.NetRole == channel_settings.TcpClientText {
		fsc.tcpTransport.SettChan() <- tcp_transport.TcpTransportSettings{ServerAddr: fsc.curSet.RemoteAddress, ServerPort: fsc.curSet.RemotePort}
	} else {
		fsc.tcpTransport.SettChan() <- tcp_transport.TcpTransportSettings{ClientAddr: fsc.curSet.RemoteAddress, LocalPort: fsc.curSet.LocalPort}
	}

	for {
		select {
		// изменено состояние подключения по TCP
		case tcpConnected := <-fsc.tcpTransport.ConnStateChan():
			var webState, webStateColor string

			if tcpConnected {
				fsc.forceNewEvent(fmtp.LSetup)
				webState = "OK"
				webStateColor = channel_state.WebOkColor
			} else {
				fsc.forceNewEvent(fmtp.RDisconnect)
				webState = "Ошибка"
				webStateColor = channel_state.WebErrorColor
			}

			if fsc.curSet.NetRole == "server" {
				logger.SetDebugParam("Состояние TCP сервера:", webState, webStateColor)
			} else {
				logger.SetDebugParam("Подключение к удаленному TCP серверу:", webState, webStateColor)
			}

		// получены данные от контроллера (chief) поверх FMTP
		case fmtpMsgFromChief := <-fsc.FmtpDataSendChan:
			fsc.sendPacket(fmtpMsgFromChief, fmtp.LData, fmtp_log.SeverityInfo)

		// полученные по TCP данные
		case receivedData := <-fsc.tcpTransport.ReceivedChan():
			if _, err := fsc.receivedBuffer.Write(receivedData); err == nil {
				for {
					headerBytes := make([]byte, fmtp.FmtpHeaderLen)
					if _, err := fsc.receivedBuffer.Read(headerBytes); err == nil {
						curHeader := fmtp.ParceFmtpPacketHeader(headerBytes)
						if curHeader.IsValid {
							// читаем тело сообщения
							bodyBytes := make([]byte, curHeader.BodyLen())
							if _, err := fsc.receivedBuffer.Read(bodyBytes); err == nil {
								fsc.processFmtpMessage(fmtp.FmtpMessage{Type: curHeader.PkgType, Text: string(bodyBytes)})

							} else {
								if err != io.EOF {
									fsc.LogMessageChan <- fmtp_log.LogChannelST(fmtp_log.SeverityError,
										fmt.Sprintf("Ошибка чтения из буфера полученных данных данных. Ошибка: <%s>", err.Error()))
								}
								break
							}
						}
					} else {
						if err != io.EOF {
							fsc.LogMessageChan <- fmtp_log.LogChannelST(fmtp_log.SeverityError,
								fmt.Sprintf("Ошибка чтения из буфера полученных данных данных. Ошибка: <%s>", err.Error()))
						}
						break
					}
				}
			} else {
				fmt.Println("Buffer error ", err.Error())
				fsc.LogMessageChan <- fmtp_log.LogChannelST(fmtp_log.SeverityError,
					fmt.Sprintf("Ошибка записи в буфер полученных данных данных. Ошибка: <%s>", err.Error()))
			}

		// полученно сообщение для журнала
		case curLogMsg := <-fsc.tcpTransport.LogChan():
			fsc.LogMessageChan <- curLogMsg

		// получение события после отправки данных по TCP
		case curEvent := <-fsc.tcpTransport.EventChan():
			fsc.forceNewEvent(curEvent)

		// сработал таймер Ti
		case curEvent := <-fsc.tiTimer.eventChan:
			fsc.forceNewEvent(curEvent)

		// сработал таймер Ts
		case curEvent := <-fsc.tsTimer.eventChan:
			fsc.forceNewEvent(curEvent)

		// сработал таймер Tr
		case curEvent := <-fsc.trTimer.eventChan:
			fsc.forceNewEvent(curEvent)

			// сработал таймер отправки FMTP состояния канала
		case <-fsc.stateTick.C:
			curChannelState := channel_state.ChannelStateOk
			if fsc.currentState != fmtp.DataReady {
				curChannelState = channel_state.ChannelStateError
			}

			fsc.FmtpStateChan <- channel_state.ChannelState{
				ChannelID:   fsc.curSet.Id,
				LocalName:   fsc.curSet.LocalATC,
				RemoteName:  fsc.curSet.RemoteName,
				DaemonState: curChannelState,
				FmtpState:   fsc.currentState.ToString(),
				ChannelURL:  fmt.Sprintf("http://%s:%d/%s", fsc.curSet.URLAddress, fsc.curSet.URLPort, fsc.curSet.URLPath),
			}
		}
	}
}

// обработать новое полученное сообщение
func (fsc *StateController) processFmtpMessage(fmtpMsg fmtp.FmtpMessage) {

	switch fmtpMsg.Type {

	case fmtp.Identification:
		switch string(fmtpMsg.Text) {
		case string(fmtp.RejectMessage.Text):
			fsc.processEventMessage(fmtp.RReject, fmtpMsg, fmtp_log.SeverityInfo)
		case string(fmtp.AcceptMessage.Text):
			fsc.processEventMessage(fmtp.RAccept, fmtpMsg, fmtp_log.SeverityInfo)
		default:
			if string(fmtpMsg.Text) == string(fsc.remoteIdentificationMsg.Text) {

				fsc.processEventMessage(fmtp.RIdValid, fmtpMsg, fmtp_log.SeverityInfo)
			} else {
				fsc.LogMessageChan <- fmtp_log.LogChannelSTDT(fmtp_log.SeverityWarning, fmtpMsg.Type.ToString(), fmtp_log.DirectionIncoming,
					fmt.Sprintf("Несовпадение идентификационных сообщений. Ожидаемый идентификатор - <%s>, полученный - <%s>.",
						string(fsc.remoteIdentificationMsg.Text), string(fmtpMsg.Text)))

				fsc.processEventMessage(fmtp.RIdInvalid, fmtpMsg, fmtp_log.SeverityInfo)
			}
		}

	case fmtp.System:
		switch string(fmtpMsg.Text) {
		case string(fmtp.StartupMessage.Text):
			fsc.processEventMessage(fmtp.RStartup, fmtpMsg, fmtp_log.SeverityInfo)
		case string(fmtp.ShutdownMessage.Text):
			fsc.processEventMessage(fmtp.RShutdown, fmtpMsg, fmtp_log.SeverityInfo)
		case string(fmtp.HeartbeatMessage.Text):
			if fsc.curSet.LogDebug {
				fsc.processEventMessage(fmtp.RHeartbeat, fmtpMsg, fmtp_log.SeverityInfo)
			} else {
				fsc.processEventMessage(fmtp.RHeartbeat, fmtpMsg, fmtp_log.SeverityDebug)
			}
		}
	case fmtp.Operational:
		fsc.processEventMessage(fmtp.RData, fmtpMsg, fmtp_log.SeverityDebug)
		fsc.processDataMessage(fmtpMsg)

	default:
		fsc.processDataMessage(fmtpMsg)
	}
}

// оработать новое событие
func (fsc *StateController) forceNewEvent(curEvent fmtp.FmtpEvent) {
	if nextState := fsc.stateMachine.GetNextState(fsc.currentState, curEvent); nextState != fmtp.Empt {
		if fsc.currentState != nextState {
			fsc.LogMessageChan <- fmtp_log.LogChannelST(fmtp_log.SeverityInfo,
				fmt.Sprintf("Смена FMTP состояния: <%s> -> <%s> по событию <%s>.",
					fsc.currentState.ToString(), nextState.ToString(), curEvent.ToString()))
		}

		if exitFunc, exitFunkOk := fsc.stateExitFuncMap[fsc.currentState]; exitFunkOk == true {
			exitFunc(fsc, curEvent)
		}
		fmt.Println("STATE changed: ", fsc.currentState.ToString(), " -> ", nextState.ToString(), " by event ", curEvent.ToString())
		fsc.currentState = nextState

		if enterFunc, enterFunkOk := fsc.stateEnterFuncMap[fsc.currentState]; enterFunkOk == true {
			enterFunc(fsc, curEvent)
		}
	}
}

func (fsc *StateController) sendPacket(messageToSend fmtp.FmtpMessage, fmtpEvent fmtp.FmtpEvent, logSeverity string) {
	utfTextToLog := messageToSend.Text

	if fsc.curSet.DataEncoding == channel_settings.Encode1251 {
		messageToSend.Text = string(utils.Utf8toWin1251([]byte(messageToSend.Text)))
	}

	fsc.tcpTransport.SendChan() <- tcp_transport.DataAndEvent{DataToSend: fmtp.MakeFmtpPacket(messageToSend), EventAfterSend: fmtpEvent}

	if (logSeverity == fmtp_log.SeverityDebug && fsc.curSet.LogDebug) || logSeverity != fmtp_log.SeverityDebug {
		fsc.LogMessageChan <- fmtp_log.LogChannelSTDT(logSeverity, messageToSend.Type.ToString(), fmtp_log.DirectionOutcoming,
			fmt.Sprintf("Содержание: <%s>.", utfTextToLog))
	}
}

func (fsc *StateController) processEventMessage(curEvent fmtp.FmtpEvent, fmtpMsg fmtp.FmtpMessage, logSeverity string) {
	if (logSeverity == fmtp_log.SeverityDebug && fsc.curSet.LogDebug) || logSeverity != fmtp_log.SeverityDebug {

		if fsc.curSet.DataEncoding == channel_settings.Encode1251 && curEvent == fmtp.RData {
			fmtpMsg.Text = string(utils.Win1251toUtf8([]byte(fmtpMsg.Text)))
		}

		fsc.LogMessageChan <- fmtp_log.LogChannelSTDT(logSeverity, fmtpMsg.Type.ToString(), fmtp_log.DirectionIncoming,
			fmt.Sprintf("Содержание: <%s>. Указание обработать событие <%s>",
				fmtpMsg.Text, curEvent.ToString()))
	}
	fsc.forceNewEvent(curEvent)
}

func (fsc *StateController) processDataMessage(fmtpMsg fmtp.FmtpMessage) {
	if fsc.curSet.DataEncoding == channel_settings.Encode1251 {
		fmtpMsg.Text = string(utils.Win1251toUtf8([]byte(fmtpMsg.Text)))
	}

	fsc.LogMessageChan <- fmtp_log.LogChannelSTDT(fmtp_log.SeverityInfo, fmtpMsg.Type.ToString(), fmtp_log.DirectionIncoming,
		fmt.Sprintf("Содержание: <%s>.",
			fmtpMsg.Text))

	fsc.FmtpDataReceiveChan <- fmtpMsg
}
