package aodb

import (
	"fmt"
	"time"

	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/logger/common"
	"fdps/fmtp/utils"
	"fdps/fmtp/web"
	"fdps/fmtp/web_sock"
)

// интервал проверки состояния контроллера
const stateTickerInt = time.Second

// Controller контроллер для работы с провайдером AODB
type Controller struct {
	ProviderSettsChan  chan []fdps.ProviderSettings // канал для приема настроек провайдеров
	ProviderSetts      []fdps.ProviderSettings      // текущие настройки провайдеров
	ProviderStatesChan chan []fdps.ProviderState    // канал для передачи состояний провайдеров
	ProviderStates     []fdps.ProviderState         // текущее состояние провайдеров

	FromAODBDataChan chan []byte // канал для приема сообщений от провайдера AODB
	ToAODBDataChan   chan []byte // канал для отправки сообщений провайдеру AODB

	LogChan chan common.LogMessage // канал для передачи сообщений журнала

	ws               *web_sock.WebSockServer
	checkStateTicker *time.Ticker // тикер для проверки состояния контроллера
}

// NewController конструктор
func NewController() *Controller {
	return &Controller{
		ProviderSettsChan:  make(chan []fdps.ProviderSettings, 10),
		ProviderStatesChan: make(chan []fdps.ProviderState, 10),
		FromAODBDataChan:   make(chan []byte, 1024),
		ToAODBDataChan:     make(chan []byte, 1024),
		LogChan:            make(chan common.LogMessage, 10),
		ws:                 web_sock.NewWebSockServer(),
		checkStateTicker:   time.NewTicker(stateTickerInt),
	}
}

// Work реализация работы
func (c *Controller) Work() {
	go c.ws.Work(utils.AodbURLPath)

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

			c.ws.SettingsChan <- web_sock.WebSockServerSettings{NeedWork: true,
				LocalPort: localPort, PermitClientIps: permitIPs}

		// получен новый пакет для отправки провайдеру
		case incomeData := <-c.ToAODBDataChan:
			c.ws.SendDataChan <- web_sock.WsPackage{Data: incomeData}

		// получены данные от WS сервера
		case curWsPkg := <-c.ws.ReceiveDataChan:
			c.FromAODBDataChan <- curWsPkg.Data

		// получена ошибка от ws сервера
		case wsErr := <-c.ws.ErrorChan:
			var curLogMsg common.LogMessage
			if wsErr == nil {
				web.SetAODBState(true)
				curLogMsg = common.LogCntrlST(common.SeverityInfo, "Запущен WS сервер для взаимодействия с AODB провайдером.")
			} else {
				web.SetAODBState(false)
				curLogMsg = common.LogCntrlST(common.SeverityError,
					fmt.Sprintf("Возникла ошибка при работе WS сервера взаимодействия с AODB провайдером. Ошибка: <%s>.", wsErr.Error()))
			}
			c.LogChan <- curLogMsg

		// получено уведомление от ws сервера
		case wsInfo := <-c.ws.InfoChan:
			c.LogChan <- common.LogCntrlST(common.SeverityInfo, "Сервер WS для взаимодействия с AODB провайдером. "+wsInfo)

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
					if c.ws.IsIPConnected(ipVal) {
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
