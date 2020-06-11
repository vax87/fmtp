package aodb

import (
	"fmt"
	"strconv"
	"time"

	"fdps/fmtp/chief/fdps"
	"fdps/utils"
	"fdps/utils/logger"
	"fdps/utils/web_sock"
)

// интервал проверки состояния контроллера
const stateTickerInt = time.Second

const (
	srvStateKey = "AODB WS. Состояние:"
	// srvLastConnKey    = "AODB WS. Последнее клиентское подключение:"
	// srvLastDisconnKey = "AODB WS. Последнее клиентское отключение:"
	// srvLastErrKey     = "AODB WS. Последняя ошибка:"
	//srvClntListKey    = "AODB WS. Список клиентов:"

	srvStateOkValue    = "Запущен."
	srvStateErrorValue = "Не запущен."

	timeFormat = "2006-01-02 15:04:05"
)

// Controller контроллер для работы с провайдером AODB
type Controller struct {
	ProviderSettsChan  chan []fdps.ProviderSettings // канал для приема настроек провайдеров
	ProviderSetts      []fdps.ProviderSettings      // текущие настройки провайдеров
	ProviderStatesChan chan []fdps.ProviderState    // канал для передачи состояний провайдеров
	ProviderStates     []fdps.ProviderState         // текущее состояние провайдеров

	FromAODBDataChan chan []byte // канал для приема сообщений от провайдера AODB
	ToAODBDataChan   chan []byte // канал для отправки сообщений провайдеру AODB

	wsServer         *web_sock.WebSockServer
	wsServerSetts    web_sock.WebSockServerSettings
	checkStateTicker *time.Ticker // тикер для проверки состояния контроллера
}

// NewController конструктор
func NewController(done chan struct{}) *Controller {
	return &Controller{
		ProviderSettsChan:  make(chan []fdps.ProviderSettings, 10),
		ProviderStatesChan: make(chan []fdps.ProviderState, 10),
		FromAODBDataChan:   make(chan []byte, 1024),
		ToAODBDataChan:     make(chan []byte, 1024),
		wsServer:           web_sock.NewWebSockServer(done),
		checkStateTicker:   time.NewTicker(stateTickerInt),
	}
}

// Work реализация работы
func (c *Controller) Work() {
	go c.wsServer.Work("/" + utils.FmtpAodbWsUrlPath)

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

			c.wsServer.SettingsChan <- web_sock.WebSockServerSettings{
				Port: localPort, PermitClientIps: permitIPs}

		// получен новый пакет для отправки провайдеру
		case incomeData := <-c.ToAODBDataChan:
			c.wsServer.SendDataChan <- web_sock.WsPackage{Data: incomeData}

		// получен подключенный клиент от WS сервера
		case curClnt := <-c.wsServer.ClntConnChan:
			logger.PrintfInfo("Подключен клиент AODB с адресом: %s.", curClnt.RemoteAddr().String())
			//logger.SetDebugParam(srvLastConnKey, curClnt.RemoteAddr().String()+" "+time.Now().Format(timeFormat), logger.StateDefaultColor)
			//logger.SetDebugParam(srvClntListKey, fmt.Sprintf("%v", c.wsServer.ClientList()), logger.StateDefaultColor)

		// получен отключенный клиент от WS сервера
		case curClnt := <-c.wsServer.ClntDisconnChan:
			logger.PrintfInfo("Отключен клиент AODB с адресом: %s.", curClnt.RemoteAddr().String())
			//logger.SetDebugParam(srvLastDisconnKey, curClnt.RemoteAddr().String()+" "+time.Now().Format(timeFormat), logger.StateDefaultColor)
			//logger.SetDebugParam(srvClntListKey, fmt.Sprintf("%v", c.wsServer.ClientList()), logger.StateDefaultColor)

		// получен отклоненный клиент от WS сервера
		case curClnt := <-c.wsServer.ClntRejectChan:
			logger.PrintfInfo("Отклонен клиент AODB с адресом: %s.", curClnt.RemoteAddr().String())

		// получена ошибка от WS сервера
		case wsErr := <-c.wsServer.ErrorChan:
			logger.PrintfErr("Возникла ошибка при работе WS сервера AODB. Ошибка: %s.", wsErr.Error())
			//logger.SetDebugParam(srvLastErrKey, wsErr.Error(), logger.StateErrorColor)

		// получены данные от WS сервера
		case curWsPkg := <-c.wsServer.ReceiveDataChan:
			c.FromAODBDataChan <- curWsPkg.Data

		// получено состояние работоспособности WS сервера для связи с AFTN каналами
		case connState := <-c.wsServer.StateChan:
			switch connState {
			case web_sock.ServerTryToStart:
				logger.PrintfInfo("Запускаем WS сервер для взаимодействия c AODB. Порт: %d Path: %s.", c.wsServerSetts.Port, utils.ParkingClientsPath)
				logger.SetDebugParam(srvStateKey, srvStateOkValue+" Порт: "+strconv.Itoa(c.wsServerSetts.Port)+" Path: "+utils.StatisticsClientsPath, logger.StateOkColor)
			case web_sock.ServerError:
				logger.SetDebugParam(srvStateKey, srvStateErrorValue, logger.StateErrorColor)
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
					if c.wsServer.IsIPConnected(ipVal) {
						curState.ProviderState = fdps.ProviderStateOk
						curState.ProviderURL = fmt.Sprintf("http://%s:8888", ipVal)
						break IPLBL
					}
				}
				states = append(states, curState)
			}
			c.ProviderStatesChan <- states
		}
	}
}
