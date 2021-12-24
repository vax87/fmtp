package chief_channel

import (
	"encoding/json"
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/fmtp_logger"

	"fdps/go_utils/logger"
	"fdps/go_utils/web_sock"
	"fdps/utils"
	"fmt"
	"time"
)

// ClientSettings настройки клиента контроллера каналов
type ClientSettings struct {
	ChiefAddress string // адрес контроллера каналов
	ChiefPort    int    // порт приема данных контроллера каналов
	ChannelID    int    // идентификатор канала
}

// Client клиент взаимодействия контроллера и канала
type Client struct {
	LogChan chan fmtp_logger.LogMessage // канал для передачи сообщний для журнала

	ReceiveChan chan []byte // канал для приема данных от контроллера каналов
	SendChan    chan []byte // канал для отправки данных контроллеру каналов

	SettChan            chan ClientSettings
	setts               ClientSettings // текущие настройки канала
	ws                  *web_sock.WebSockClient
	sendSettingsRequest bool      // был отправлен запрос настроек. При потере связи с сконтроллером и ее восстановлении настройки не перезапрашиваются
	disconnTime         time.Time // время дисконнекта
	CloseChan           chan struct{}
}

// NewChiefChannelClient конструктор
func NewChiefChannelClient(done chan struct{}) *Client {
	return &Client{
		LogChan:     make(chan fmtp_logger.LogMessage, 100),
		ReceiveChan: make(chan []byte, 1024),
		SendChan:    make(chan []byte, 1024),
		SettChan:    make(chan ClientSettings),
		ws:          web_sock.NewWebSockClient(done),
		CloseChan:   make(chan struct{}),
	}
}

// Work реализация работы клиента
func (c *Client) Work() {
	for {
		select {
		// новые настройки
		case newSetts := <-c.SettChan:
			if c.setts != newSetts {
				c.setts = newSetts
				go c.ws.Work(web_sock.WebSockClientSettings{
					ServerAddress:     newSetts.ChiefAddress,
					ServerPort:        newSetts.ChiefPort,
					UrlPath:           utils.FmtpChannelWsUrlPath,
					ReconnectInterval: 3,
				})
			}

		// данные для отправки
		//case _ = <-c.SendChan:
		case toChiefData := <-c.SendChan:
			c.ws.SendDataChan <- toChiefData

		// полученные данные от chief
		case fromChiefData := <-c.ws.ReceiveDataChan:
			c.ReceiveChan <- fromChiefData

		// изменено сотсояние подключения
		case wsState := <-c.ws.StateChan:
			var connStateStr string

			if wsState == web_sock.ClientConnected {

				c.disconnTime = time.Time{}

				if c.sendSettingsRequest == false {
					c.LogChan <- fmtp_logger.LogChannelST(fmtp_logger.SeverityDebug,
						fmt.Sprintf("Запущен FMTP канал id = %d.", c.setts.ChannelID))

					if dataToSend, err := json.Marshal(CreateSettingsRequestMsg(c.setts.ChannelID)); err == nil {
						c.LogChan <- fmtp_logger.LogChannelST(fmtp_logger.SeverityDebug,
							fmt.Sprintf("Запрос настроек от FMTP канала id = %d.", c.setts.ChannelID))

						c.SendChan <- dataToSend
						c.sendSettingsRequest = true
					}
				}
				connStateStr = "Подключен"
				logger.SetDebugParam("Подключение к контролеру:", connStateStr, channel_state.WebOkColor)
			} else if wsState == web_sock.ClientDisconnected {
				connStateStr = "Не подключен"
				logger.SetDebugParam("Подключение к контролеру:", connStateStr, channel_state.WebOkColor)

				// не было подключения к серверу
				if c.disconnTime.IsZero() {
					c.disconnTime = time.Now()
				} else {
					if c.disconnTime.Add(time.Minute).Before(time.Now()) {
						fmt.Println(" disconnTime c.CloseChan <- struct{}{}")
						c.CloseChan <- struct{}{}
					}
				}
			}

			c.LogChan <- fmtp_logger.LogChannelST(fmtp_logger.SeverityDebug,
				fmt.Sprintf("Изменено состояние подключения FMTP канала в контроллеру. Id канала = %d. Состояние: %s", c.setts.ChannelID, connStateStr))

		// ошибка WS
		case wsErr := <-c.ws.ErrorChan:

			c.LogChan <- fmtp_logger.LogChannelST(fmtp_logger.SeverityError,
				fmt.Sprintf("Ошибка сетевого взаимодействия FMTP канала и контроллера. Ошибка: %s.", wsErr.Error()))

		}
	}
}
