package tcp_transport

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/fmtp"
)

// клиентское TCP подключение
type TcpTransportClient struct {
	sync.Mutex

	settChan chan TcpTransportSettings // канал приема новых настроек канала
	curSett  TcpTransportSettings      // текущие настройки канала

	receivedDataChan   chan []byte         // канал для принятых данных
	toSendDataChan     chan DataAndEvent   // канал для отправки данных по TCP
	eventAfterSendChan chan fmtp.FmtpEvent // событие, генерируемое после отправки (кроме None)

	tcpClient      net.Conn      // клиентское подключение по TCPv4
	cancelWorkChan chan struct{} // канал для сигнала прекращения отправки, чтения данных

	logMessageChan chan common.LogMessage // канал для передачи сообщний для журнала
	connStateChan  chan bool              // канал для передачи успешности подключения по TCP
	reconnectChan  chan struct{}          // канал для сообщения TCP клиенту о необходимости подключитья к серверу

	lastConnectError   error // последняя возникшая ошибка при установке соединения (чтоб не отправлять в лог одно и то же)
	lastKeepaliveError error // последняя возникшая ошибка при установке keepalive (чтоб не отправлять в лог одно и то же)
	errorChan          chan error
}

// конструктор
func NewFmtpTcpClient() *TcpTransportClient {
	return &TcpTransportClient{
		settChan:           make(chan TcpTransportSettings),
		receivedDataChan:   make(chan []byte, 1024),
		toSendDataChan:     make(chan DataAndEvent, 1024),
		eventAfterSendChan: make(chan fmtp.FmtpEvent),
		cancelWorkChan:     make(chan struct{}),
		logMessageChan:     make(chan common.LogMessage, 10),
		connStateChan:      make(chan bool),
		reconnectChan:      make(chan struct{}),
		errorChan:          make(chan error),
		lastConnectError:   errors.New(""),
		lastKeepaliveError: errors.New(""),
	}
}

func (ftc *TcpTransportClient) SettChan() chan TcpTransportSettings {
	return ftc.settChan
}

func (ftc *TcpTransportClient) ReceivedChan() chan []byte {
	return ftc.receivedDataChan
}

func (ftc *TcpTransportClient) SendChan() chan DataAndEvent {
	return ftc.toSendDataChan
}

func (ftc *TcpTransportClient) EventChan() chan fmtp.FmtpEvent {
	return ftc.eventAfterSendChan
}

func (ftc *TcpTransportClient) LogChan() chan common.LogMessage {
	return ftc.logMessageChan
}

func (ftc *TcpTransportClient) ConnStateChan() chan bool {
	return ftc.connStateChan
}

func (ftc *TcpTransportClient) ReconnectChan() chan struct{} {
	return ftc.reconnectChan
}

// запуск работы контроллера
func (ftc *TcpTransportClient) Work() {
	for {
		select {
		// получены новые настройки
		case newSettings := <-ftc.settChan:
			if ftc.curSett != newSettings {
				ftc.curSett = newSettings
				ftc.startClient()
			}
		case <-ftc.errorChan:
			ftc.stopClient()

		case <-ftc.reconnectChan:
			ftc.startClient()
		}
	}
}

//
func (ftc *TcpTransportClient) startClient() {
	var err error
	if ftc.tcpClient, err = net.Dial("tcp", ftc.curSett.ServerAddr+":"+strconv.Itoa(ftc.curSett.ServerPort)); err != nil {
		if err.Error() != ftc.lastConnectError.Error() {
			ftc.lastConnectError = err
			ftc.logMessageChan <- common.LogChannelST(common.SeverityError,
				fmt.Sprintf("При установке TCP соединения возникла ошибка. Ошибка:<%s>", err.Error()))
		}
		ftc.connStateChan <- false
		return
	}
	ftc.logMessageChan <- common.LogChannelST(common.SeverityInfo, "Установлено TCP соединение FMTP канала.")
	ftc.connStateChan <- true

	if err = ftc.tcpClient.(*net.TCPConn).SetKeepAlive(false); err != nil {
		if err.Error() != ftc.lastKeepaliveError.Error() {
			ftc.lastKeepaliveError = err
			ftc.logMessageChan <- common.LogChannelST(common.SeverityError,
				fmt.Sprintf("При установке флага keep_alive TCP соединения возникла ошибка. Ошибка:<%s>", err.Error()))
		}
		ftc.connStateChan <- false
		return
	}

	go ftc.receiveLoop()
	go ftc.sendLoop()
}

//
func (ftc *TcpTransportClient) stopClient() {
	ftc.Lock()
	ftc.cancelWorkChan <- struct{}{}
	ftc.Unlock()

	ftc.connStateChan <- false

	if err := ftc.tcpClient.Close(); err != nil {
		ftc.logMessageChan <- common.LogChannelST(common.SeverityError,
			fmt.Sprintf("Ошибка при закрытии TCP соединение FMTP канала. Ошибка: <%s>.", err.Error()))
	} else {
		ftc.logMessageChan <- common.LogChannelST(common.SeverityError, "Закрыто TCP соединение FMTP канала.")
	}
}

// обработчик получения данных
func (ftc *TcpTransportClient) receiveLoop() {
	for {
		select {
		// отмена приема данных
		case <-ftc.cancelWorkChan:
			return
		// прием данных
		default:
			buffer := make([]byte, 1024)
			if readBytes, err := ftc.tcpClient.Read(buffer); err != nil {
				if err != io.EOF {
					ftc.logMessageChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
						fmt.Sprintf("Ошибка чтения данных из FMTP канала. Ошибка: <%s>.", err.Error()))
				}
				ftc.errorChan <- err
				return
			} else {
				fmt.Println("Read bytes", readBytes, "  ", string(buffer[:readBytes]))
				ftc.receivedDataChan <- buffer[:readBytes]
			}
		}
	}
}

// обработчик отправки данных
func (ftc *TcpTransportClient) sendLoop() {
	for {
		select {
		// отмена отправки данных
		case <-ftc.cancelWorkChan:
			return

		// получены данные для отправки
		case curData := <-ftc.toSendDataChan:
			if _, err := ftc.tcpClient.Write(curData.DataToSend); err != nil {
				ftc.logMessageChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
					fmt.Sprintf("Ошибка отправки данных в FMTP канала. Ошибка: <%s>.", err.Error()))
				ftc.errorChan <- err
				return
			} else if curData.EventAfterSend != fmtp.None {
				ftc.eventAfterSendChan <- curData.EventAfterSend
			}
		}
	}
}
