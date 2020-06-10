package chief_channel

import (
	"fdps/fmtp/logger/common"
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

	SettChan chan ClientSettings
	setts    ClientSettings // текущие настройки канала
	//ws       *web_sock.WebSockClient
}

// NewChiefChannelClient конструктор
func NewChiefChannelClient() *Client {
	return &Client{
		LogChan:     make(chan common.LogMessage, 100),
		ReceiveChan: make(chan []byte, 1024),
		SendChan:    make(chan []byte, 1024),
		SettChan:    make(chan ClientSettings),
		//ws:          web_sock.NewWebSockClient(),
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
				//go c.ws.Work(web_sock.WebSockClientSettings{ServerAddress: newSetts.ChiefAddress, ServerPort: newSetts.ChiefPort, UrlPath: utils.ChannelURLPath})
			}

		// данные для отправки
		case _ = <-c.SendChan:
			//case toChiefData := <-c.SendChan:
			//c.ws.SendDataChan <- toChiefData

			// полученные данные от chief
			//case fromChiefData := <-c.ws.ReceiveDataChan:
			//	c.ReceiveChan <- fromChiefData

			// ошибка WS
			// case wsErr := <-c.ws.ErrorChan:
			// 	if wsErr == nil {
			// 		web.SetChiefConn(true)
			// 		c.LogChan <- common.LogChannelST(common.SeverityInfo,
			// 			fmt.Sprintf("Запущен FMTP канал id = <%d>.", c.setts.ChannelID))

			// 		if dataToSend, err := json.Marshal(CreateSettingsRequestMsg(c.setts.ChannelID)); err == nil {
			// 			c.LogChan <- common.LogChannelST(common.SeverityInfo,
			// 				fmt.Sprintf("Запрос настроек от FMTP канала id = <%d>.", c.setts.ChannelID))

			// 			c.SendChan <- dataToSend
			// 		}
			// 	} else {
			// 		web.SetChiefConn(false)
			// 		c.LogChan <- common.LogChannelST(common.SeverityError,
			// 			fmt.Sprintf("Ошибка сетевого взаимодействия FMTP канала и контроллера. Ошибка: <%s>.", wsErr.Error()))
			// 	}

		}
	}
}
