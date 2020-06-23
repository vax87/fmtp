package oldi

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief_logger"
	"fdps/fmtp/logger/common"
	"fdps/utils"
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
	cancelWorkChan chan struct{} // канал для сигнала прекращения отправки/приема данных
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
	tcpListener     net.Listener
	tcpListenerWork bool
	tcpLocalPort    int

	closeTcpListenerFunc func()

	providerEncoding string
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

func (c *OldiController) startServer(ctx context.Context, localPort int) {
	var errListen error
	if c.tcpListener, errListen = net.Listen("tcp", string(":"+strconv.Itoa(localPort))); errListen != nil {
		logger.PrintfErr("Ошибка запуска TCP сервера OLDI провайдера. Ошибка: %v.", errListen)
		logger.SetDebugParam(srvStateKey, srvStateErrorValue, logger.StateErrorColor)
		c.tcpListenerWork = false
		return
	} else {
		c.tcpListenerWork = true
		logger.PrintfInfo("Запущен TCP сервер для работы с OLDI провайдером. Порт: %d", localPort)
		logger.SetDebugParam(srvStateKey, srvStateOkValue+" Порт: "+strconv.Itoa(localPort), logger.StateOkColor)
		//defer tcpListener.Close()

		for {
			select {

			case <-ctx.Done():
				return

			default:
				if curConn, err := c.tcpListener.Accept(); err != nil {
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
			}
		}
	}
}

func (c *OldiController) stopServer() {

	for key := range c.providerClients {
		c.closeClient(key)
	}

	if errClose := c.tcpListener.Close(); errClose != nil {
		logger.PrintfErr("Ошибка закрытия TCP сервера для подключения OLDI провайдеров. Ошибка: %v")
	}
	c.closeTcpListenerFunc()
}

func (c *OldiController) closeClient(conn net.Conn) {
	if val, ok := c.providerClients[conn]; ok == true {

		conn.SetDeadline(time.Now().Add(time.Second))
		// останавливаем передачу / прием
		utils.ChanSafeClose(val.cancelWorkChan)
		logger.PrintfErr("Отключен клиент OLDI провайдера. Адрес: %s", conn.RemoteAddr().String())
		delete(c.providerClients, conn)
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
				chief_logger.ChiefLog.FmtpLogChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
					fmt.Sprintf("Ошибка чтения данных из FMTP канала. Ошибка: %v.", err))

				c.closeClient(clntConn)
			} else {
				logger.PrintfDebug("Приняты данные от OLDI провайдера: %v", string(buffer[:readBytes]))

				if c.providerEncoding == channel_settings.Encode1251 {
					c.FromOldiDataChan <- utils.Win1251toUtf8(buffer[:readBytes])
				} else {
					c.FromOldiDataChan <- buffer[:readBytes]
				}
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
			var dataToSend []byte
			if c.providerEncoding == channel_settings.Encode1251 {
				dataToSend = utils.Utf8toWin1251(curData)
			}

			if _, err := clntConn.Write(dataToSend); err != nil {
				chief_logger.ChiefLog.FmtpLogChan <- common.LogChannelSTDT(common.SeverityError, common.NoneFmtpType, common.DirectionIncoming,
					fmt.Sprintf("Ошибка отправки данных в FMTP канала. Ошибка: %v.", err))
				c.closeClient(clntConn)
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
				c.providerEncoding = val.ProviderEncoding
			}

			if c.tcpLocalPort != localPort {
				c.tcpLocalPort = localPort

				if c.tcpListenerWork == true {
					c.stopServer()
				}
				var ctx context.Context
				ctx, c.closeTcpListenerFunc = context.WithCancel(context.Background())

				go c.startServer(ctx, c.tcpLocalPort)
			}

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

				for _, ipVal := range val.IPAddresses {
					for key := range c.providerClients {
						if strings.HasPrefix(key.RemoteAddr().String(), ipVal) {
							curState.ProviderState = fdps.ProviderStateOk
							curState.ClientAddresses += " " + key.RemoteAddr().String()
						}
					}
				}
				states = append(states, curState)
			}
			c.ProviderStatesChan <- states
		}
	}
}
