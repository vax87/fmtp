package configurator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"fmtp/chief/chief_settings"
	"fmtp/chief/chief_state"
	"fmtp/chief/chief_web"

	"lemz.com/fdps/logger"
	"lemz.com/fdps/utils"
)

const dbgChannelVersions = "Версии FMTP каналов"

// ChiefCfg настройки контроллера каналов(chief)
var ChiefCfg chief_settings.ChiefSettings = chief_settings.ChiefSettings{CntrlID: -1, IPAddr: "127.0.0.1", IsInitialised: false}

// ChannelImageName имя docker образа приложения FMTP канал
const ChannelImageName = "fmtp_channel"

// клиент контроллера для подключения к конфигуратору
type ChiefConfiguratorClient struct {
	ConfiguratorClient
	ChiefSettChangedChan   chan struct{}
	readLocalSettingsTimer *time.Timer
	sendHeartbeatTicker    *time.Ticker
	withDocker             bool
	channelVersions        []string // список версий приложений/docker бразов FMTP канала
}

func NewChiefClient(workWithDocker bool) *ChiefConfiguratorClient {
	return &ChiefConfiguratorClient{
		ConfiguratorClient:     ConfiguratorClient{postResultChan: make(chan httpResult, 1)},
		ChiefSettChangedChan:   make(chan struct{}, 1),
		readLocalSettingsTimer: time.NewTimer(time.Minute),
		sendHeartbeatTicker:    time.NewTicker(time.Second),
		withDocker:             workWithDocker,
	}
}

func (c *ChiefConfiguratorClient) Work() {
	if c.withDocker {
		// останавливаем и удаляем ранее запущенные контейнеры fmtp каналов
		if stopErr := utils.StopContainers(ChannelImageName); stopErr != nil {
			logger.PrintfErr("Ошибка остановки docker контейнеров приложения FMTP канал. Ошибка: %s.", stopErr.Error())
		}
	}

	for {
		select {

		// результат выполнения POST запроса
		case postRes := <-c.postResultChan:
			if postRes.err == nil {
				var msgHeader ChiefHdr
				if unmErr := json.Unmarshal(postRes.result, &msgHeader); unmErr != nil {
					logger.PrintfErr("Ошибка разбора (unmarshall) сообщения от конфигуратора. Сообщение: %s. Ошибка: %s.", string(postRes.result), unmErr.Error())
				} else {
					switch msgHeader.Header {
					// ответ на запрос настроек
					case ChiefSettsAnswerHdr:
						ChiefCfg = chief_settings.ChiefSettings{CntrlID: -1, IPAddr: "", IsInitialised: false}
						if unmErr = json.Unmarshal(postRes.result, &ChiefCfg); unmErr != nil {
							logger.PrintfErr("Ошибка разбора (unmarshall) ответа на запрос настроек. Сообщение: %s. Ошибка: %s.", string(postRes.result), unmErr.Error())
						} else {
							ChiefCfg.IsInitialised = true
							logger.PrintfDebug("Получены настройки от конфигуратора. %+v.", ChiefCfg)

							c.readLocalSettingsTimer.Stop()

							ChiefCfg.SaveToFile()

							c.initAfterGetSettings()

							// отправляем настройки
							c.sendSettings()
						}

					// ответ на сообщение о состоянии контроллера
					case ChiefHbtAnswerHdr:
						var curMsg ChiefHbtAnswerMsg
						if unmErr := json.Unmarshal(postRes.result, &curMsg); unmErr != nil {
							logger.PrintfErr("Ошибка разбора (unmarshall) ответа на сообщение о состоянии контроллера. Сообщение: %s. Ошибка: %s.",
								string(postRes.result), unmErr.Error())
						} else {
							if curMsg.ConfigTimestamp != ChiefCfg.Timestamp {
								go c.postToConfigurator(c.configUrls.SettingsURLStr, CreateChiefSettsRequestMsg(c.channelVersions))
							}
						}

					default:
						logger.PrintfErr("Получено сообщение с неизвестным заголовком от конфигуратора. Заголовок: %s.", msgHeader.Header)
					}
				}
			} else {
				logger.PrintfErr("%v", postRes.err)
				ChiefCfg.IsInitialised = false
				time.AfterFunc(time.Minute, func() {
					go c.postToConfigurator(c.configUrls.SettingsURLStr, CreateChiefSettsRequestMsg(c.channelVersions))
				})
			}

		// сработал таймер отправки состояния (heartbeat)
		case <-c.sendHeartbeatTicker.C:
			if ChiefCfg.IsInitialised {
				go c.postToConfigurator(c.configUrls.HeartbeatURLStr,
					ChiefHbtMsg{
						ChiefHdr:   ChiefHdr{Header: ChiefHbtHdr},
						ChiefState: chief_state.CommonChiefState,
					})
			}

		// сработал таймер считывания настроек из файла
		case <-c.readLocalSettingsTimer.C:
			if fileErr := ChiefCfg.ReadFromFile(); fileErr != nil {
				logger.PrintfErr("Ошибка чтения настроек контроллера из файла. Ошибка: %s.", fileErr.Error())
			} else {
				logger.PrintfWarn("Настройки контроллера считаны из файла.")
				// отправляем настройки
				c.sendSettings()
			}

		// получены настроки URL из web
		case c.configUrls = <-chief_web.UrlConfigChan:
			c.setUrls()
		}
	}
}

// Start запуск взаимодействия с конфигуратором
func (c *ChiefConfiguratorClient) Start() {
	c.getUrls()
	chief_web.SetUrlConfig(c.configUrls)

	c.initBeforeGetSettings()
	logger.PrintfDebug("Собственные IP адреса: %v", utils.GetLocalIpv4List())
	go c.postToConfigurator(c.configUrls.SettingsURLStr, CreateChiefSettsRequestMsg(c.channelVersions))
}

// инициализация после получений настроек
func (cc *ChiefConfiguratorClient) initBeforeGetSettings() {
	logger.SetDebugParam(dbgChannelVersions, "-", logger.StateDefaultColor)

	var versErr error

	if !cc.withDocker {
		if cc.channelVersions, versErr = utils.GetChannelVersions(); versErr != nil {
			logger.PrintfErr("Ошибка получения списка версий приложения FMTP канал. Ошибка: %v", versErr)
		} else {
			logger.SetDebugParam(dbgChannelVersions, fmt.Sprintf("%v", cc.channelVersions), logger.StateDefaultColor)
			logger.PrintfDebug("Получены версии приложения FMTP канал. Версии: %s.", cc.channelVersions)
		}
	} else {
		if cc.channelVersions, versErr = utils.GetDockerImageVersions(ChannelImageName); versErr != nil {
			logger.PrintfErr(
				"Ошибка получения списка версий docker образов FMTP канал. Ошибка: %v", versErr)
		} else {
			logger.SetDebugParam(dbgChannelVersions, fmt.Sprintf("%v", cc.channelVersions), logger.StateDefaultColor)
			logger.PrintfDebug("Получены версии doсker образов приложения FMTP канал. Версии: %s", cc.channelVersions)
		}
	}
}

// инициализация после получений настроек
func (cc *ChiefConfiguratorClient) initAfterGetSettings() {

	if cc.withDocker {
		// останавливаем и удаляем ранее запущенные контейнеры fmtp каналов
		// if stopErr := utils.StopContainers(ChannelImageName); stopErr != nil {
		// 	logger.PrintfErr("Ошибка остановки docker контейнеров приложения FMTP канал. Ошибка: %s.", stopErr.Error())
		// }

		if len(ChiefCfg.DockerRegistry) > 0 {
			// выкачиваем docker образы из репозитория
			if pullErr := utils.DockerPullImages(ChiefCfg.DockerRegistry + "/" + ChannelImageName); pullErr != nil {
				logger.PrintfErr("Ошибка загрузки образов (pull) docker приложения FMTP канал. Ошибка: %s", pullErr.Error())
			}
		}

		// поучаем список текущих версий
		if newVersions, versErr := utils.GetDockerImageVersions(ChannelImageName); versErr != nil {
			logger.PrintfErr("Ошибка получения списка версий docker образов приложения FMTP канал (Повторный запрос). Ошибка: %s.", versErr.Error())
		} else {
			// если появились новые версии, заново запрашиваем настройки
			if !reflect.DeepEqual(cc.channelVersions, newVersions) {
				cc.channelVersions = newVersions
				go cc.postToConfigurator(cc.configUrls.SettingsURLStr, CreateChiefSettsRequestMsg(cc.channelVersions))
			}
		}
	}
}

// отправляе настройки каналам, провайдерам, клиенту логгера
func (cc *ChiefConfiguratorClient) sendSettings() {
	// добавляем в настройки URL
	for ind := range ChiefCfg.ChannelSetts {
		ChiefCfg.ChannelSetts[ind].URLAddress = ChiefCfg.IPAddr
		ChiefCfg.ChannelSetts[ind].URLPath = "channel"
		ChiefCfg.ChannelSetts[ind].URLPort = utils.FmtpChannelStartWebPort + ChiefCfg.ChannelSetts[ind].Id
	}

	// выставляем кодировку сообщений для OLDI провайдеров
	for idx, val := range ChiefCfg.ProvidersSetts {
		if val.DataType == chief_settings.OLDIProvider {
			ChiefCfg.ProvidersSetts[idx].ProviderEncoding = ChiefCfg.OldiProviderEncoding
		}
	}

	// отправляем настройки провайдеров
	for ind, val := range ChiefCfg.ProvidersSetts {
		if val.DataType == chief_settings.AODBProvider {
			ChiefCfg.ProvidersSetts[ind].LocalPort = ChiefCfg.AodbProviderPort
		} else if val.DataType == chief_settings.OLDIProvider {
			ChiefCfg.ProvidersSetts[ind].LocalPort = ChiefCfg.OldiProviderPort
		}
	}
	cc.ChiefSettChangedChan <- struct{}{}

	chief_state.CommonChiefState.CntrlID = ChiefCfg.CntrlID
	chief_state.CommonChiefState.IPAddr = ChiefCfg.IPAddr
}
