package chief_configurator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
	"fdps/fmtp/chief/chief_web"

	"fdps/fmtp/chief_configurator/configurator_urls"
	"fdps/go_utils/logger"
	"fdps/utils"
)

const dbgChannelVersions = "Версии FMTP каналов"

// ChiefCfg настройки контроллера каналов(chief)
var ChiefCfg chief_settings.ChiefSettings = chief_settings.ChiefSettings{CntrlID: -1, IPAddr: "127.0.0.1", IsInitialised: false}

// ChannelImageName имя docker образа приложения FMTP канал
const ChannelImageName = "fmtp_channel"

type httpResult struct {
	result json.RawMessage
	err    error
}

// клиент контроллера для подключения к конфигуратору
type ChiefConfiguratorClient struct {
	configUrls           configurator_urls.ConfiguratorUrls
	FmtpChannelSettsChan chan channel_settings.ChannelSettingsWithPort // канал для передачи настроек FMTP каналов
	LoggerSettsChan      chan chief_settings.LoggerSettings            // канал для передачи настроек логгера
	ProviderSettsChan    chan []chief_settings.ProviderSettings        // канал для передачи натроек провайдеров

	postResultChan chan httpResult

	readLocalSettingsTimer *time.Timer

	withDocker bool

	channelVersions []string // список версий приложений/docker бразов FMTP канала

	sendHeartbeatTicker *time.Ticker
}

// NewChiefClient конструктор клиента
func NewChiefClient(workWithDocker bool) *ChiefConfiguratorClient {
	return &ChiefConfiguratorClient{
		FmtpChannelSettsChan:   make(chan channel_settings.ChannelSettingsWithPort, 1),
		LoggerSettsChan:        make(chan chief_settings.LoggerSettings, 1),
		ProviderSettsChan:      make(chan []chief_settings.ProviderSettings, 1),
		postResultChan:         make(chan httpResult, 1),
		readLocalSettingsTimer: time.NewTimer(time.Minute),
		withDocker:             workWithDocker,
		sendHeartbeatTicker:    time.NewTicker(time.Second),
	}
}

// Work рабочий цикл
func (cc *ChiefConfiguratorClient) Work() {

	if cc.withDocker {
		// останавливаем и удаляем ранее запущенные контейнеры fmtp каналов
		if stopErr := utils.StopContainers(ChannelImageName); stopErr != nil {
			logger.PrintfErr("Ошибка остановки docker контейнеров приложения FMTP канал. Ошибка: %s.", stopErr.Error())
		}
	}

	for {
		select {

		// результат выполнения POST запроса
		case postRes := <-cc.postResultChan:
			if postRes.err == nil {
				var msgHeader MessageHeader
				if unmErr := json.Unmarshal(postRes.result, &msgHeader); unmErr != nil {
					logger.PrintfErr("Ошибка разбора (unmarshall) сообщения от конфигуратора. Сообщение: %s. Ошибка: %s.", string(postRes.result), unmErr.Error())
				} else {
					switch msgHeader.Header {
					// ответ на запрос настроек
					case AnswerSettingsHeader:
						ChiefCfg = chief_settings.ChiefSettings{CntrlID: -1, IPAddr: "", IsInitialised: false}
						if unmErr = json.Unmarshal(postRes.result, &ChiefCfg); unmErr != nil {
							logger.PrintfErr("Ошибка разбора (unmarshall) ответа на запрос настроек. Сообщение: %s. Ошибка: %s.", string(postRes.result), unmErr.Error())
						} else {
							//logger.PrintfWarn("POST RESULT: %s", string(postRes.result))

							ChiefCfg.IsInitialised = true
							logger.PrintfDebug("Получены настройки от конфигуратора. %+v.", ChiefCfg)

							cc.readLocalSettingsTimer.Stop()

							ChiefCfg.SaveToFile()

							cc.initAfterGetSettings()

							// отправляем настройки
							cc.sendSettings()
						}

					// ответ на сообщение о состоянии контроллера
					case HeartbeatAnswerHeader:
						var curMsg HeartbeatAnswerMsg
						if unmErr := json.Unmarshal(postRes.result, &curMsg); unmErr != nil {
							logger.PrintfErr("Ошибка разбора (unmarshall) ответа на сообщение о состоянии контроллера. Сообщение: %s. Ошибка: %s.",
								string(postRes.result), unmErr.Error())
						} else {
							if curMsg.ConfigTimestamp != ChiefCfg.Timestamp {
								go cc.postToConfigurator(cc.configUrls.SettingsURLStr, CreateSettingsRequestMsg(cc.channelVersions))
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
					go cc.postToConfigurator(cc.configUrls.SettingsURLStr, CreateSettingsRequestMsg(cc.channelVersions))
				})
			}

		// сработал таймер отправки состояния (heartbeat)
		case <-cc.sendHeartbeatTicker.C:
			if ChiefCfg.IsInitialised {
				go cc.postToConfigurator(cc.configUrls.HeartbeatURLStr,
					HeartbeatMsg{
						MessageHeader: MessageHeader{Header: HeartbeatHeader},
						ChiefState:    chief_state.CommonChiefState,
					})
			}

		// сработал таймер считывания настроек из файла
		case <-cc.readLocalSettingsTimer.C:
			if fileErr := ChiefCfg.ReadFromFile(); fileErr != nil {
				logger.PrintfErr("Ошибка чтения настроек контроллера из файла. Ошибка: %s.", fileErr.Error())
			} else {
				logger.PrintfErr("Настройки контроллера считаны из файла.")
				// отправляем настройки
				cc.sendSettings()
			}

		// получены настроки URL из web
		case cc.configUrls = <-chief_web.UrlConfigChan:
			cc.configUrls.SaveToFile()
		}
	}
}

// Start запуск взаимодействия с конфигуратором
func (cc *ChiefConfiguratorClient) Start() {
	cc.configUrls.ReadFromFile()
	chief_web.SetUrlConfig(cc.configUrls)

	cc.initBeforeGetSettings()
	logger.PrintfDebug("Собственные IP адреса: %v", utils.GetLocalIpv4List())
	go cc.postToConfigurator(cc.configUrls.SettingsURLStr, CreateSettingsRequestMsg(cc.channelVersions))
}

// отправка POST сообщения конфигуратору и возврат ответа
func (cc *ChiefConfiguratorClient) postToConfigurator(url string, msg interface{}) {
	jsonValue, _ := json.Marshal(msg)
	//logger.PrintfWarn("POST: %s", string(jsonValue))
	resp, postErr := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	if postErr == nil {
		defer resp.Body.Close()
		if strings.Contains(resp.Status, "200") {
			if body, readErr := ioutil.ReadAll(resp.Body); readErr == nil {
				if bytes.Contains(body, []byte("error")) { //пишем в логи только ошибки

					cc.postResultChan <- httpResult{err: fmt.Errorf("Не валидное тело http пакета. Тело пакета: %s", string(body))}
				}
				if ind := strings.Index(string(body), "{"); ind >= 0 {
					cc.postResultChan <- httpResult{result: body[ind:], err: nil}
				} else {
					cc.postResultChan <- httpResult{err: fmt.Errorf("Не валидное тело http пакета. Тело пакета: %s", string(body))}
				}
			} else {
				cc.postResultChan <- httpResult{err: fmt.Errorf("Ошибка чтения ответа http запроса. Тело пакета: %s. Ошибка: %s", string(body), readErr.Error())}
			}
		} else {
			cc.postResultChan <- httpResult{err: fmt.Errorf("HTPP запрос выполнен с ошибкой. Запрос: %s. URL: %s. Статус ответа: %s",
				jsonValue, url, resp.Status)}
		}
	} else {
		cc.postResultChan <- httpResult{err: fmt.Errorf("Ошибка выполнения HTPP запроса. Запрос: %s. URL: %s. Ошибка: %s",
			jsonValue, url, postErr.Error())}
	}
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
				go cc.postToConfigurator(cc.configUrls.SettingsURLStr, CreateSettingsRequestMsg(cc.channelVersions))
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

	// отправляем настройки каналов
	cc.FmtpChannelSettsChan <- channel_settings.ChannelSettingsWithPort{
		ChSettings: ChiefCfg.ChannelSetts,
		ChPort:     ChiefCfg.ChannelsPort,
	}

	// отправляем настройки логгера
	cc.LoggerSettsChan <- ChiefCfg.LoggerSetts

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
	cc.ProviderSettsChan <- ChiefCfg.ProvidersSetts

	chief_state.CommonChiefState.CntrlID = ChiefCfg.CntrlID
	chief_state.CommonChiefState.IPAddr = ChiefCfg.IPAddr
}
