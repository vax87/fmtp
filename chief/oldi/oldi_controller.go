package oldi

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief_logger"
	"fdps/fmtp/logger/common"
	"fdps/utils/logger"
)

// интервал проверки состояния контроллера
const stateTickerInt = time.Second

const (
	srvStateKey = "OLDI TCP. Состояние:"

	srvStateOkValue    = "Запущен."
	srvStateErrorValue = "Не запущен."

	timeFormat = "2006-01-02 15:04:05"
)

type oldiClnt struct {
	cancelWorkChan chan struct{} // канал для сигнала прекращения отправки, чтения данных
	toSendDataChan chan []byte   // канал для отправки данных
}

// OldiController контроллер для работы с провайдером OLDI
type OldiController struct {
	ProviderSettsChan  chan []fdps.ProviderSettings // канал для приема настроек провайдеров
	ProviderSetts      []fdps.ProviderSettings      // текущие настройки провайдеров
	ProviderStatesChan chan []fdps.ProviderState    // канал для передачи состояний провайдеров
	ProviderStates     []fdps.ProviderState         // текущее состояние провайдеров

	FromOldiDataChan chan []byte // канал для приема сообщений от провайдера OLDI
	ToOldiDataChan   chan []byte // канал для отправки сообщений провайдеру OLDI

	checkStateTicker *time.Ticker // тикер для проверки состояния контроллера

	providerClients map[net.Conn]oldiClnt
}

// NewOldiController конструктор
func NewOldiController() *OldiController {
	return &OldiController{
		ProviderSettsChan:  make(chan []fdps.ProviderSettings, 10),
		ProviderStatesChan: make(chan []fdps.ProviderState, 10),
		FromOldiDataChan:   make(chan []byte, 1024),
		ToOldiDataChan:     make(chan []byte, 1024),
		checkStateTicker:   time.NewTicker(stateTickerInt),
		providerClients:    make(map[net.Conn]oldiClnt),
	}
}

//
func (c *OldiController) startServer(localPort int) {
	listener, err := net.Listen("tcp", string(":"+strconv.Itoa(localPort)))
	if err != nil {
		logger.PrintfErr("Ошибка запуска TCP сервера OLDI провайдера. Ошибка: %v.", err)
		logger.SetDebugParam(srvStateKey, srvStateErrorValue, logger.StateErrorColor)
		return
	}
	logger.PrintfInfo("Запущен TCP сервер FMTP канала.")
	logger.SetDebugParam(srvStateKey, srvStateOkValue, logger.StateOkColor)
	defer listener.Close()

	for {

		if curConn, err := listener.Accept(); err != nil {
			logger.PrintfErr("Ошибка подключения клиента к TCP серверу OLDI провайдера. Ошибка: %v.", err)
			continue
		} else {
			c.providerClients[curConn] = oldiClnt{
				cancelWorkChan: make(chan struct{}),
				toSendDataChan: make(chan []byte, 1024),
			}

			remoteAddr, _ := curConn.RemoteAddr().(*net.TCPAddr)
			logger.PrintfInfo("Успешное подключение клиента к TCP серверу OLDI провайдера. "+
				"Адрес подключенного клиента: %s", remoteAddr.IP.String())

			go c.receiveLoop(curConn, c.providerClients[curConn])
			go c.sendLoop(curConn, c.providerClients[curConn])
		}

		// if fts.tcpClient != nil {
		// 	fts.logMessageChan <- common.LogChannelST(common.SeverityWarning,
		// 		fmt.Sprintf("Отклонено входящее подключение к TCP серверу FMTP канала. "+
		// 			"Клиент уже подключен. Адрес отклоненного клиента: <%s>", remoteAddr.IP.String()))
		// 	continue
		// } else {
		// if fts.curSett.ClientAddr != "" && remoteAddr.IP.String() != fts.curSett.ClientAddr {
		// 	fts.logMessageChan <- common.LogChannelST(common.SeverityWarning,
		// 		fmt.Sprintf("Отклонено входящее подключение к TCP серверу FMTP канала. "+
		// 			"Адрес клиента не соответствует. Адрес отклоненного клиента: <%s>", remoteAddr.IP.String()))
		// } else {

		//fts.tcpClient = curConn
		//fts.connStateChan <- true
		//fts.fmtpEventChan <- fmtp.RSetup

		//}
		//}
	}
}

// обработчик получения данных
func (c *OldiController) receiveLoop(clntConn net.Conn, clnt oldiClnt) {
	for {
		select {
		// отмена приема данных
		case <-clnt.cancelWorkChan:
			return
		// прием данных
		default:
			buffer := make([]byte, 8192)
			if readBytes, err := clntConn.Read(buffer); err != nil {
				if err != io.EOF {
					chief_logger.ChiefLog.FmtpLogChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
						fmt.Sprintf("Ошибка чтения данных из FMTP канала. Ошибка: <%s>.", err.Error()))
				}
				return
			} else {
				logger.PrintfDebug("Приняты данные от OLDI провайдера: %v", string(buffer[:readBytes]))
				c.FromOldiDataChan <- buffer[:readBytes]
			}
		}
	}
}

// обработчик отправки данных
func (c *OldiController) sendLoop(clntConn net.Conn, clnt oldiClnt) {
	for {
		select {
		// отмена отправки данных
		case <-clnt.cancelWorkChan:
			return

		// получены данные для отправки
		case curData := <-clnt.toSendDataChan:
			if _, err := clntConn.Write(curData); err != nil {
				chief_logger.ChiefLog.FmtpLogChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
					fmt.Sprintf("Ошибка отправки данных в FMTP канала. Ошибка: <%s>.", err.Error()))
				return
			} else {
				logger.PrintfDebug("Отправлены данные OLDI провайдеру: %v", string(curData))
			}
		}
	}
}

// Work реализация работы
func (c *OldiController) Work() {

	for {
		select {
		// получены новые настройки каналов
		case curSetts := <-c.ProviderSettsChan:
			c.ProviderSetts = curSetts

			var permitIPs []string
			var localPort int
			for _, val := range c.ProviderSetts {
				localPort = val.LocalPort
				for _, ipVal := range val.IPAddresses {
					permitIPs = append(permitIPs, ipVal)
				}
			}

			// c.wsServer.SettingsChan <- web_sock.WebSockServerSettings{
			// 	Port: localPort, PermitClientIps: permitIPs}
			go c.startServer(localPort)

		// получен новый пакет для отправки провайдеру
		case incomeData := <-c.ToOldiDataChan:
			// отправляем всем клиентам
			for _, v := range c.providerClients {
				v.toSendDataChan <- incomeData
			}

		// сработал тикер проверки состояния контроллера
		case <-c.checkStateTicker.C:
			var states []fdps.ProviderState

			for _, val := range c.ProviderSetts {
				curState := fdps.ProviderState{
					ProviderID:    val.ID,
					ProviderType:  val.DataType,
					ProviderIPs:   val.IPAddresses,
					ProviderState: fdps.ProviderStateError,
				}

			IPLBL:
				for _, ipVal := range val.IPAddresses {

					for key := range c.providerClients {
						if strings.HasPrefix(key.RemoteAddr().String(), ipVal) {
							curState.ProviderState = fdps.ProviderStateOk
							break IPLBL
						}
					}
				}
				states = append(states, curState)
			}
			c.ProviderStatesChan <- states
		}
	}
}
