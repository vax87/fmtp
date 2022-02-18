package configurator

import (
	"encoding/json"
	"time"

	"fmtp/ora_logger/logger_settings"
	"fmtp/ora_logger/logger_state"

	"lemz.com/fdps/logger"
	"lemz.com/fdps/utils"
)

var LoggerCfg logger_settings.LoggerSettings = logger_settings.LoggerSettings{IsInitialised: false}

type LoggerConfiguratorClient struct {
	ConfiguratorClient
	LoggerSettChangedChan  chan struct{}
	readLocalSettingsTimer *time.Timer
	sendHeartbeatTicker    *time.Ticker
}

func NewLoggerClient() *LoggerConfiguratorClient {
	return &LoggerConfiguratorClient{
		ConfiguratorClient:    ConfiguratorClient{postResultChan: make(chan httpResult, 1)},
		LoggerSettChangedChan: make(chan struct{}, 1),
		//readLocalSettingsTimer: time.NewTimer(time.Minute),
		readLocalSettingsTimer: time.NewTimer(time.Second * 5),
		sendHeartbeatTicker:    time.NewTicker(time.Second),
	}
}

func (c *LoggerConfiguratorClient) Work() {

	for {
		select {

		// результат выполнения POST запроса
		case postRes := <-c.postResultChan:
			if postRes.err == nil {
				var msgHeader LoggerHdr
				if unmErr := json.Unmarshal(postRes.result, &msgHeader); unmErr != nil {
					logger.PrintfErr("Ошибка разбора (unmarshall) сообщения от конфигуратора. Сообщение: %s. Ошибка: %s.", string(postRes.result), unmErr.Error())
				} else {
					switch msgHeader.Header {
					// ответ на запрос настроек
					case LogggerSettsAnswerHdr:
						LoggerCfg.IsInitialised = false
						if unmErr = json.Unmarshal(postRes.result, &LoggerCfg); unmErr != nil {
							logger.PrintfErr("Ошибка разбора (unmarshall) ответа на запрос настроек. Сообщение: %s. Ошибка: %s.", string(postRes.result), unmErr.Error())
						} else {
							LoggerCfg.IsInitialised = true
							c.readLocalSettingsTimer.Stop()

							LoggerCfg.SaveToFile()

							// отправляем настройки
							c.sendSettings()
						}

					// ответ на сообщение о состоянии контроллера
					case LoggerHbtAnswerHdr:
						var curMsg LoggerHbtAnswerMsg
						if unmErr := json.Unmarshal(postRes.result, &curMsg); unmErr != nil {
							logger.PrintfErr("Ошибка разбора (unmarshall) ответа на сообщение о состоянии контроллера. Сообщение: %s. Ошибка: %s.",
								string(postRes.result), unmErr.Error())
						} else {
							if curMsg.ConfigTimestamp != LoggerCfg.Timestamp {
								go c.postToConfigurator(c.configUrls.SettingsURLStr, CreateLoggerSettsRequestMsg())
							}
						}

					default:
						logger.PrintfErr("Получено сообщение с неизвестным заголовком от конфигуратора. Заголовок: %s.", msgHeader.Header)
					}
				}
			} else {
				logger.PrintfErr("%v", postRes.err)
				LoggerCfg.IsInitialised = false
				time.AfterFunc(time.Minute, func() {
					go c.postToConfigurator(c.configUrls.SettingsURLStr, CreateLoggerSettsRequestMsg())
				})
			}

		// сработал таймер отправки состояния (heartbeat)
		case <-c.sendHeartbeatTicker.C:
			if LoggerCfg.IsInitialised {
				go c.postToConfigurator(c.configUrls.HeartbeatURLStr,
					LoggerHbtMsg{
						LoggerHdr:   LoggerHdr{Header: LoggerHbtHdr},
						LoggerState: logger_state.CommonLoggerState,
					})
			}

		// сработал таймер считывания настроек из файла
		case <-c.readLocalSettingsTimer.C:
			if fileErr := LoggerCfg.ReadFromFile(); fileErr != nil {
				logger.PrintfErr("Ошибка чтения настроек логгера из файла. Ошибка: %s.", fileErr.Error())
			} else {
				logger.PrintfWarn("Настройки логгера считаны из файла.")
				// отправляем настройки
				c.sendSettings()
			}
		}
	}
}

// Start запуск взаимодействия с конфигуратором
func (c *LoggerConfiguratorClient) Start() {
	c.getUrls()
	logger.PrintfDebug("Собственные IP адреса: %v", utils.GetLocalIpv4List())
	go c.postToConfigurator(c.configUrls.SettingsURLStr, CreateLoggerSettsRequestMsg())
}

func (c *LoggerConfiguratorClient) sendSettings() {
	c.LoggerSettChangedChan <- struct{}{}

	logger_state.CommonLoggerState.LoggerID = LoggerCfg.LoggerID
	logger_state.CommonLoggerState.IPAddr = LoggerCfg.IPAddr
}
