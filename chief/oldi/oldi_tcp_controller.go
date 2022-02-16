package oldi

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
	"fdps/fmtp/fmtp_log"

	//"fdps/go_utils/logger"
	"fdps/utils"

	"lemz.com/fdps/logger"
)

// интервал проверки состояния контроллера
const stateTickerInt = time.Second

const (
	srvStateKey = "OLDI TCP. Состояние:"

	srvStateOkValue    = "Запущен."
	srvStateErrorValue = "Не запущен."
)

type oldiClnt struct {
	cancelWorkChan chan struct{} // канал для сигнала прекращения отправки/приема данных
	toSendDataChan chan []byte   // канал для отправки данных
}

// OldiTcpController контроллер для работы с провайдером OLDI по TCP
type OldiTcpController struct {
	SettsChan chan []chief_settings.ProviderSettings // канал для приема настроек провайдеров
	setts     []chief_settings.ProviderSettings      // текущие настройки провайдеров

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
func NewOldiTcpController() *OldiTcpController {
	return &OldiTcpController{
		SettsChan:        make(chan []chief_settings.ProviderSettings, 10),
		FromOldiDataChan: make(chan []byte, 1024),
		ToOldiDataChan:   make(chan []byte, 1024),
		checkStateTicker: time.NewTicker(stateTickerInt),
		providerClients:  make(map[net.Conn]oldiClnt),
	}
}

func (c *OldiTcpController) startServer(ctx context.Context, localPort int) {
	var errListen error
	if c.tcpListener, errListen = net.Listen("tcp", string(":"+strconv.Itoa(localPort))); errListen != nil {
		logger.PrintfErr("Ошибка запуска TCP сервера OLDI провайдера. Ошибка: %v.", errListen)
		logger.SetDebugParam(srvStateKey, srvStateErrorValue, logger.StateErrorColor)
		c.tcpListenerWork = false
		return
	} else {
		c.tcpListenerWork = true
		logger.PrintfDebug("Запущен TCP сервер для работы с OLDI провайдером. Порт: %d", localPort)
		logger.SetDebugParam(srvStateKey, srvStateOkValue+" Порт: "+strconv.Itoa(localPort), logger.StateOkColor)
		//defer tcpListener.Close()

		for {
			select {

			case <-ctx.Done():
				return

			default:
				if curConn, err := c.tcpListener.Accept(); err != nil {
					logger.PrintfErr("Ошибка подключения OLDI провайдера. Ошибка: %v.", err)
					continue
				} else {
					c.providerClients[curConn] = oldiClnt{
						cancelWorkChan: make(chan struct{}),
						toSendDataChan: make(chan []byte, 1024),
					}

					remoteAddr, _ := curConn.RemoteAddr().(*net.TCPAddr)
					logger.PrintfDebug("Успешное подключение OLDI провайдера. Адрес подключенного клиента: %s", remoteAddr.IP.String())

					go c.receiveLoop(curConn, c.providerClients[curConn])
					go c.sendLoop(curConn, c.providerClients[curConn])
				}
			}
		}
	}
}

func (c *OldiTcpController) stopServer() {

	for key := range c.providerClients {
		c.closeClient(key)
	}

	if errClose := c.tcpListener.Close(); errClose != nil {
		logger.PrintfErr("Ошибка закрытия TCP сервера для подключения OLDI провайдеров. Ошибка: %v")
	}
	c.closeTcpListenerFunc()
}

func (c *OldiTcpController) closeClient(conn net.Conn) {
	if val, ok := c.providerClients[conn]; ok {

		conn.SetDeadline(time.Now().Add(time.Second))
		// останавливаем передачу / прием
		utils.ChanSafeClose(val.cancelWorkChan)
		logger.PrintfDebug("Отключен OLDI провайдер. Адрес: %s", conn.RemoteAddr().String())
		delete(c.providerClients, conn)
	}
}

// обработчик получения данных
func (c *OldiTcpController) receiveLoop(clntConn net.Conn, clnt oldiClnt) {
	for {
		select {
		// отмена приема данных
		case <-clnt.cancelWorkChan:
			return
		// прием данных
		default:
			buffer := make([]byte, 8192)
			if readBytes, err := clntConn.Read(buffer); err != nil {
				logger.PrintfErr("FMTP FORMAT %#v", fmtp_log.LogChannelSTDT(fmtp_log.SeverityError, fmtp_log.NoneFmtpType, fmtp_log.DirectionIncoming,
					fmt.Sprintf("Ошибка чтения данных из FMTP канала. Ошибка: %v.", err)))

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
func (c *OldiTcpController) sendLoop(clntConn net.Conn, clnt oldiClnt) {
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
				logger.PrintfErr("FMTP FORMAT %#v", fmtp_log.LogChannelSTDT(fmtp_log.SeverityError, fmtp_log.NoneFmtpType, fmtp_log.DirectionIncoming,
					fmt.Sprintf("Ошибка отправки данных в FMTP канала. Ошибка: %v.", err)))
				c.closeClient(clntConn)
			} else {
				logger.PrintfDebug("Отправлены данные OLDI провайдеру: %v", string(curData))
			}
		}
	}
}

// Work реализация работы
func (c *OldiTcpController) Work() {

	for {
		select {
		// получены новые настройки каналов
		case c.setts = <-c.SettsChan:
			var localPort int
			for _, val := range c.setts {
				localPort = val.LocalPort
				c.providerEncoding = val.ProviderEncoding
			}

			if c.tcpLocalPort != localPort {
				c.tcpLocalPort = localPort

				if c.tcpListenerWork {
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
			var states []chief_state.ProviderState

			for _, val := range c.setts {
				curState := chief_state.ProviderState{
					ProviderID:    val.ID,
					ProviderType:  val.DataType,
					ProviderIPs:   val.IPAddresses,
					ProviderState: chief_state.StateError,
				}

				for _, ipVal := range val.IPAddresses {
					for key := range c.providerClients {
						if strings.HasPrefix(key.RemoteAddr().String(), ipVal) {
							curState.ProviderState = chief_state.StateOk
							curState.ClientAddresses += " " + key.RemoteAddr().String()
						}
					}
				}
				states = append(states, curState)
			}
			chief_state.SetOldiProviderState(states)
		}
	}
}
