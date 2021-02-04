package tcp_transport

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/fmtp"
)

// серверное TCP подключение
type TcpTransportServer struct {
	sync.Mutex

	settChan chan TcpTransportSettings // канал приема новых настроек канала
	curSett  TcpTransportSettings      // текущие настройки канала

	receivedDataChan chan []byte         // канал для принятых данных
	toSendDataChan   chan DataAndEvent   // канал для отправки данных по TCP
	fmtpEventChan    chan fmtp.FmtpEvent // событие, передаваемое контроллеру состояний

	tcpClient      net.Conn      // клиентское подключение по TCPv4
	cancelWorkChan chan struct{} // канал для сигнала прекращения отправки, чтения данных

	logMessageChan chan common.LogMessage // канал для передачи сообщний для журнала
	connStateChan  chan bool              // канал для передачи успешности подключения по TCP
	reconnectChan  chan struct{}          // канал для сообщения TCP клиенту о необходимости подключитья к серверу (не используется)

	errorChan chan error
}

// конструктор
func NewFmtpTcpServer() *TcpTransportServer {
	return &TcpTransportServer{
		settChan:         make(chan TcpTransportSettings),
		receivedDataChan: make(chan []byte, 1024),
		toSendDataChan:   make(chan DataAndEvent, 1024),
		fmtpEventChan:    make(chan fmtp.FmtpEvent),
		cancelWorkChan:   make(chan struct{}),
		logMessageChan:   make(chan common.LogMessage, 10),
		connStateChan:    make(chan bool),
		errorChan:        make(chan error),
		reconnectChan:    make(chan struct{}),
	}
}

func (fts *TcpTransportServer) SettChan() chan TcpTransportSettings {
	return fts.settChan
}

func (fts *TcpTransportServer) ReceivedChan() chan []byte {
	return fts.receivedDataChan
}

func (fts *TcpTransportServer) SendChan() chan DataAndEvent {
	return fts.toSendDataChan
}

func (fts *TcpTransportServer) EventChan() chan fmtp.FmtpEvent {
	return fts.fmtpEventChan
}

func (fts *TcpTransportServer) LogChan() chan common.LogMessage {
	return fts.logMessageChan
}

func (fts *TcpTransportServer) ConnStateChan() chan bool {
	return fts.connStateChan
}

func (fts *TcpTransportServer) ReconnectChan() chan struct{} {
	return fts.reconnectChan
}

// запуск работы контроллера
func (fts *TcpTransportServer) Work() {
	for {
		select {
		// получены новые настройки
		case newSettings := <-fts.settChan:
			if fts.curSett != newSettings {
				fts.curSett = newSettings
				go fts.startServer()
			}
		case <-fts.errorChan:
			fmt.Println("fts.stopClient() 90 line")
			fts.stopClient()

		case <-fts.reconnectChan:
		}
	}
}

//
func (fts *TcpTransportServer) startServer() {
	var listener net.Listener
	listener, err := net.Listen("tcp", string(":"+strconv.Itoa(fts.curSett.LocalPort)))
	if err != nil {
		fts.logMessageChan <- common.LogChannelST(common.SeverityError,
			fmt.Sprintf("Ошибка запуска TCP сервера FMTP канала. Ошибка: <%s>.", err.Error()))

		fts.connStateChan <- false
		return
	}
	fts.connStateChan <- true
	fts.logMessageChan <- common.LogChannelST(common.SeverityInfo, "Запущен TCP сервер FMTP канала.")
	defer listener.Close()

	for {
		var curConn net.Conn
		curConn, err := listener.Accept()
		if err != nil {
			fts.logMessageChan <- common.LogChannelST(common.SeverityError,
				fmt.Sprintf("Ошибка подключения клиента к TCP серверу FMTP канала. Ошибка: <%s>.", err.Error()))
			continue
		}
		remoteAddr, _ := curConn.RemoteAddr().(*net.TCPAddr)

		if fts.tcpClient != nil {
			fmt.Println("fts.stopClient() 124 line")
			fts.stopClient()
			// fts.logMessageChan <- common.LogChannelST(common.SeverityWarning,
			// 	fmt.Sprintf("Отклонено входящее подключение к TCP серверу FMTP канала. "+
			// 		"Клиент уже подключен. Адрес отклоненного клиента: <%s>", remoteAddr.IP.String()))
			// continue
		} else {
			if fts.curSett.ClientAddr != "" && remoteAddr.IP.String() != fts.curSett.ClientAddr {
				fts.logMessageChan <- common.LogChannelST(common.SeverityWarning,
					fmt.Sprintf("Отклонено входящее подключение к TCP серверу FMTP канала. "+
						"Адрес клиента не соответствует. Адрес отклоненного клиента: <%s>", remoteAddr.IP.String()))
			} else {
				fts.logMessageChan <- common.LogChannelST(common.SeverityInfo,
					fmt.Sprintf("Успешное подключение клиента к TCP серверу FMTP канала. "+
						"Адрес подключенного клиента: <%s>", remoteAddr.IP.String()))

				fmt.Println("fts.tcpClient = curConn 140 line")
				fts.tcpClient = curConn
				fts.connStateChan <- true
				fts.fmtpEventChan <- fmtp.RSetup

				fmt.Println("receiveLoop() sendLoop() 145 line")
				go fts.receiveLoop()
				go fts.sendLoop()
			}
		}
	}
}

//
func (fts *TcpTransportServer) stopClient() {
	//fts.Lock()
	//utils.ChanSafeClose(fts.cancelWorkChan)
	fts.cancelWorkChan <- struct{}{}

	fts.connStateChan <- false

	if err := fts.tcpClient.Close(); err != nil {
		fmt.Println("Ошибка при закрытии клиентского TCP подключения" + err.Error())
		fts.logMessageChan <- common.LogChannelST(common.SeverityError,
			fmt.Sprintf("Ошибка при закрытии клиентского TCP подключения FMTP канала. Ошибка: <%s>.", err.Error()))
	} else {
		fmt.Println("fts.tcpClient = nil line 166")
		fts.logMessageChan <- common.LogChannelST(common.SeverityInfo, "Закрыто клиентское TCP соединение FMTP канала.")
		fts.tcpClient = nil
	}
	//fts.Unlock()
}

// обработчик получения данных
func (fts *TcpTransportServer) receiveLoop() {
	for {
		select {
		// отмена приема данных
		case <-fts.cancelWorkChan:
			fmt.Println("cancelWorkChan in receiveLoop() line 179")
			return
		// прием данных
		default:
			if fts.tcpClient != nil {
				buffer := make([]byte, 8192)
				if readBytes, err := fts.tcpClient.Read(buffer); err != nil {
					if err != io.EOF {
						fts.logMessageChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
							fmt.Sprintf("Ошибка чтения данных из FMTP канала. Ошибка: <%s>.", err.Error()))
					}
					fmt.Println("fts.errorChan in receiveLoop() line 190")
					fts.errorChan <- err
					//fts.stopClient()
					return
				} else {
					fmt.Println("Read bytes", readBytes)
					fts.receivedDataChan <- buffer[:readBytes]
				}
			}
		}
	}
}

// обработчик отправки данных
func (fts *TcpTransportServer) sendLoop() {
	for {
		select {
		// отмена отправки данных
		case <-fts.cancelWorkChan:
			fmt.Println("cancelWorkChan in sendLoop() line 209")
			return

		// получены данные для отправки
		case curData := <-fts.toSendDataChan:
			if writeBytes, err := fts.tcpClient.Write(curData.DataToSend); err != nil {
				fmt.Println("Read bytes", writeBytes)
				fts.logMessageChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
					fmt.Sprintf("Ошибка отправки данных в FMTP канала. Ошибка: <%s>.", err.Error()))
				fmt.Println("fts.errorChan in sendLoop() line 218")
				fts.errorChan <- err
				//fts.stopClient()
				return
			} else if curData.EventAfterSend != fmtp.None {
				fts.fmtpEventChan <- curData.EventAfterSend
			}
		}
	}
}
