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
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/logger/common"
	"fdps/fmtp/web"
	"fdps/utils"
	"fdps/utils/logger"
)

const dbgChannelVersions = "Версии FMTP каналов"

// ChiefCfg настройки контроллера каналов(chief)
var ChiefCfg chief_settings.ChiefSettings = chief_settings.ChiefSettings{CntrlID: -1, IPAddr: "127.0.0.1", IsInitialised: false}

// ChannelVersions список версий приложений/docker бразов FTP канала
var ChannelVersions []string

// ChannelImageName имя docker образа приложения 'FMTP канал'
const ChannelImageName = "fmtp_channel"

// клиент контроллера для подключения к конфигуратору
type ChiefConfigutarorClient struct {
	configUrls           ConfiguratorUrls
	HeartbeatChan        chan HeartbeatMsg                             // канал для приема сообщений о состоянии контроллера
	FmtpChannelSettsChan chan channel_settings.ChannelSettingsWithPort // канал для передачи настроек FMTP каналов
	LoggerSettsChan      chan common.LoggerSettings                    // канал для передачи настроек логгера
	ProviderSettsChan    chan []fdps.ProviderSettings                  // канал для передачи натроек провайдеров

	postResultChan chan json.RawMessage

	readLocalSettingsTimer *time.Timer

	withDocker bool
}

// NewChiefClient конструктор клиента
func NewChiefClient(workWithDocker bool) *ChiefConfigutarorClient {
	return &ChiefConfigutarorClient{
		HeartbeatChan:          make(chan HeartbeatMsg, 10),
		FmtpChannelSettsChan:   make(chan channel_settings.ChannelSettingsWithPort),
		LoggerSettsChan:        make(chan common.LoggerSettings),
		ProviderSettsChan:      make(chan []fdps.ProviderSettings),
		postResultChan:         make(chan json.RawMessage, 10),
		readLocalSettingsTimer: time.NewTimer(time.Minute),
		withDocker:             workWithDocker,
	}
}

// Work рабочий цикл
func (cc *ChiefConfigutarorClient) Work() {
	for {
		select {

		// результат выполнения POST запроса
		case postRes := <-cc.postResultChan:
			var msgHeader MessageHeader
			if unmErr := json.Unmarshal(postRes, &msgHeader); unmErr != nil {
				logger.PrintfErr("Ошибка разбора (unmarshall) сообщения от конфигуратора. Сообщение: %s. Ошибка: %s.", string(postRes), unmErr.Error())
			} else {
				switch msgHeader.Header {
				// ответ на запрос настроек
				case AnswerSettingsHeader:
					if unmErr = json.Unmarshal(postRes, &ChiefCfg); unmErr != nil {
						logger.PrintfErr("Ошибка разбора (unmarshall) ответа на запрос настроек. Сообщение: %s. Ошибка: %s.", string(postRes), unmErr.Error())
					} else {
						//chiefCfg = &curMsg
						ChiefCfg.IsInitialised = true
						cc.readLocalSettingsTimer.Stop()

						// todo
						if len(ChiefCfg.DockerRegistry) == 0 {
							ChiefCfg.DockerRegistry = "di.topaz-atcs.com"
						}
						// сохраняем настройки в файл
						ChiefCfg.SaveToFile()

						cc.initAfterGetSettings()

						// отправляем настройки
						go cc.sendSettings()
					}

				// ответ на сообщение о состоянии контроллера
				case HeartbeatAnswerHeader:
					var curMsg HeartbeatAnswerMsg
					if unmErr := json.Unmarshal(postRes, &curMsg); unmErr != nil {
						logger.PrintfErr("Ошибка разбора (unmarshall) ответа на сообщение о состоянии контроллера. Сообщение: %s. Ошибка: %s.", string(postRes), unmErr.Error())
					} else {
						if curMsg.ConfigTimestamp != ChiefCfg.Timestamp {
							go cc.postSettingsRequest()
						}
					}

				default:
					logger.PrintfErr("Получено сообщение с неизвестнм заголовком от конфигуратора. Загогловок: %s.", msgHeader.Header)
				}
			}

		// получено собщение о состоянии контроллера
		case hbMgs := <-cc.HeartbeatChan:
			go cc.postToConfigurator(cc.configUrls.HeartbeatURLStr, hbMgs)

		// сработал таймер считывания настроек из файла
		case <-cc.readLocalSettingsTimer.C:
			if fileErr := ChiefCfg.ReadFromFile(); fileErr != nil {
				logger.PrintfErr("Ошибка чтения настроек контроллера з файла. Ошибка: %s.", fileErr.Error())
			} else {
				logger.PrintfErr("Настройки контроллера считаны из файла.")
				// отправляем настройки
				go cc.sendSettings()
			}
		}
	}
}

// Start запуск взаимодействия с конфигуратором
func (cc *ChiefConfigutarorClient) Start() {
	cc.configUrls.ReadFromFile()

	cc.initBeforeGetSettings()
	cc.postSettingsRequest()
}

// отправка запроса настроек конфигуратору
func (cc *ChiefConfigutarorClient) postSettingsRequest() {

	postErr := cc.postToConfigurator(cc.configUrls.SettingsURLStr, CreateSettingsRequestMsg(ChannelVersions))
	if postErr != nil {
		logger.PrintfErr("Ошибка запроса настроек у конфигуратора. Ошибка: %s", postErr.Error())

		// если настройки не инициализированы, снова их запрашиваем
		// при считывании из файла настроки не инициализируются (только при получении от конфигуратора)
		if !ChiefCfg.IsInitialised {
			time.AfterFunc(5*time.Second, func() { cc.postSettingsRequest() })
		}
	}
}

// отправка POST сообщения конфигуратору и возврат ответа
func (cc *ChiefConfigutarorClient) postToConfigurator(url string, msg interface{}) error {
	jsonValue, _ := json.Marshal(msg)
	resp, postErr := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	if postErr == nil {
		defer resp.Body.Close()
		if strings.Contains(resp.Status, "200") {
			if body, readErr := ioutil.ReadAll(resp.Body); readErr == nil {
				if bytes.Contains(body, []byte("error")) { //пишем в логи только ошибки
					web.SetConfigConn(false)
					return fmt.Errorf("Не валидное тело http пакета. Тело пакета: %s", string(body))
				}
				if ind := strings.Index(string(body), "{"); ind >= 0 {
					cc.postResultChan <- body[ind:]
				} else {
					web.SetConfigConn(false)
					return fmt.Errorf("Не валидное тело http пакета. Тело пакета: %s", string(body))
				}
			} else {
				web.SetConfigConn(false)
				return fmt.Errorf("Ошибка чтения ответа http запроса. Тело пакета: %s. Ошибка: %s", string(body), readErr.Error())
			}
		} else {
			web.SetConfigConn(false)
			return fmt.Errorf("HTPP запрос выполнен с ошибкой. Запрос: %s. URL: %s. Статус ответа: %s",
				jsonValue, url, resp.Status)
		}
	} else {
		web.SetConfigConn(false)
		return fmt.Errorf("Ошибка выполнения HTPP запроса. Запрос: %s. URL: %s. Ошибка: %s",
			jsonValue, url, postErr.Error())
	}

	web.SetConfigConn(true)
	return nil
}

// инициализация после получений настроек
func (cc *ChiefConfigutarorClient) initBeforeGetSettings() {
	logger.SetDebugParam(dbgChannelVersions, "-", logger.StateDefaultColor)

	if cc.withDocker == false {

		if ChannelVersions, versErr := utils.GetChannelVersions(); versErr != nil {
			logger.PrintfErr("Ошибка получения списка версий приложения <FMTP канал>. Ошибка: %v", versErr)
		} else {
			logger.SetDebugParam(dbgChannelVersions, fmt.Sprintf("%v", ChannelVersions), logger.StateDefaultColor)
			logger.PrintfInfo("Получены версии приложения <FMTP канал>. Версии: %s.", ChannelVersions)
		}
	} else {
		if ChannelVersions, versErr := utils.GetDockerImageVersions(ChannelImageName); versErr != nil {
			logger.PrintfErr(
				"Ошибка получения списка версий docker образов 'FMTP канал'. Ошибка: %v", versErr)
		} else {
			logger.SetDebugParam(dbgChannelVersions, fmt.Sprintf("%v", ChannelVersions), logger.StateDefaultColor)
			logger.PrintfInfo("Получены версии doсker образов приложения 'FMTP канал'. Версии: %s", ChannelVersions)
		}
	}
}

// инициализация после получений настроек
func (cc *ChiefConfigutarorClient) initAfterGetSettings() {

	if cc.withDocker {

		// останавливаем и удаляем ранее запущенные контейнеры fmtp каналов
		if stopErr := utils.StopAndRmContainers(ChannelImageName); stopErr != nil {
			logger.PrintfErr("Ошибка остановки и удаления docker контейнеров приложения 'FMTP канал'. Ошибка: %s.", stopErr.Error())
		}

		// выкачиваем docker образы из репозитория
		if pullErr := utils.DockerPullImages(ChiefCfg.DockerRegistry + ""); pullErr != nil {
			logger.PrintfErr("Ошибка загрузки образов (pull) docker приложения 'FMTP канал'. Ошибка: %s", pullErr.Error())
		}

		// поучаем список текущих версий
		if newVersions, versErr := utils.GetChannelVersions(); versErr != nil {
			logger.PrintfErr("Ошибка получения списка версий docker образов приложения 'FMTP канал' (Повторный запрос). Ошибка: %s.", versErr.Error())
		} else {
			// если пойвились новые версии, заново запрашиваем настройки
			if !reflect.DeepEqual(ChannelVersions, newVersions) {
				ChannelVersions = newVersions
				cc.postSettingsRequest()
			}
		}
	}
}

// отправляе настройки каналам, провайдерам, клиенту логгера
func (cc *ChiefConfigutarorClient) sendSettings() {
	// добавляем в настройки URL
	for ind, _ := range ChiefCfg.ChannelSetts {
		ChiefCfg.ChannelSetts[ind].URLAddress = ChiefCfg.IPAddr
		ChiefCfg.ChannelSetts[ind].URLPath = "channel"
		ChiefCfg.ChannelSetts[ind].URLPort = 13100 + ChiefCfg.ChannelSetts[ind].Id
	}

	// отправляем настройки каналов
	cc.FmtpChannelSettsChan <- channel_settings.ChannelSettingsWithPort{
		ChSettings: ChiefCfg.ChannelSetts,
		ChPort:     ChiefCfg.ChannelsPort,
	}

	// отправляем настройки логгера
	cc.LoggerSettsChan <- ChiefCfg.LoggerSetts
	// отправляем настройки провайдеров
	for ind, val := range ChiefCfg.ProvidersSetts {
		if val.DataType == fdps.AODBProvider {
			ChiefCfg.ProvidersSetts[ind].LocalPort = ChiefCfg.AodbProviderPort
		} else if val.DataType == fdps.OLDIProvider {
			ChiefCfg.ProvidersSetts[ind].LocalPort = ChiefCfg.OldiProviderPort
		}
	}
	cc.ProviderSettsChan <- ChiefCfg.ProvidersSetts
}
