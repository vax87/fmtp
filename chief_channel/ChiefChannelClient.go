package chief_channel

import (
	"encoding/json"
	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/web"
	"fdps/utils"
	"fdps/utils/web_sock"
	"fmt"
)

// ClientSettings настройки клиента контроллера каналов
type ClientSettings struct {
	ChiefAddress string // адрес контроллера каналов
	ChiefPort    int    // порт приема данных контроллера каналов
	ChannelID    int    // идентификатор канала
}

// Client клиент взаимодействия контроллера и канала
type Client struct {
	LogChan chan common.LogMessage // канал для передачи сообщний для журнала

	ReceiveChan chan []byte // канал для приема данных от контроллера каналов
	SendChan    chan []byte // канал для отправки данных контроллеру каналов

	SettChan            chan ClientSettings
	setts               ClientSettings // текущие настройки канала
	ws                  *web_sock.WebSockClient
	sendSettingsRequest bool // был отправлен запрос настроек. При потере связи с сконтроллером и ее восстановлении настройки не перезапрашиваются
}

// NewChiefChannelClient конструктор
func NewChiefChannelClient(done chan struct{}) *Client {
	return &Client{
		LogChan:     make(chan common.LogMessage, 100),
		ReceiveChan: make(chan []byte, 1024),
		SendChan:    make(chan []byte, 1024),
		SettChan:    make(chan ClientSettings),
		ws:          web_sock.NewWebSockClient(done),
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
			if wsState == web_sock.ClientConnected {
				web.SetChiefConn(true)
				if c.sendSettingsRequest == false {
					c.LogChan <- common.LogChannelST(common.SeverityDebug,
						fmt.Sprintf("Запущен FMTP канал id = %d.", c.setts.ChannelID))

					if dataToSend, err := json.Marshal(CreateSettingsRequestMsg(c.setts.ChannelID)); err == nil {
						c.LogChan <- common.LogChannelST(common.SeverityDebug,
							fmt.Sprintf("Запрос настроек от FMTP канала id = %d.", c.setts.ChannelID))

						c.SendChan <- dataToSend
						c.sendSettingsRequest = true
					}
				}
			} else if wsState == web_sock.ClientDisconnected {
				web.SetChiefConn(false)
			}

			c.LogChan <- common.LogChannelST(common.SeverityDebug,
				fmt.Sprintf("Изменено состояние подключения FMTP канала в контроллеру. Id канала = %d. Состояние: %v", c.setts.ChannelID, wsState))

		// ошибка WS
		case wsErr := <-c.ws.ErrorChan:

			c.LogChan <- common.LogChannelST(common.SeverityError,
				fmt.Sprintf("Ошибка сетевого взаимодействия FMTP канала и контроллера. Ошибка: %s.", wsErr.Error()))

		}
	}
}
