package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/phf/go-queue/queue"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/aodb"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/chief_logger"
	"fdps/fmtp/logger/common"
	"fdps/fmtp/utils"
	"fdps/fmtp/web"
	"fdps/fmtp/web_sock"
	ut2 "fdps/utils"
	"fdps/utils/logger"
	"fdps/utils/logger/log_ljack"
	"fdps/utils/logger/log_std"
	"fdps/utils/logger/log_web"
)

const (
	// максимальное кол-во логов в logCache
	maxLogQueueSize = 30000

	// максимальное кол-во отправляемых сообщений по сети
	maxLogSendCount = 20
)

// клиент для связи с конфигуратором
var chiefConfClient *chief_configurator.ChiefConfigutarorClient

// клиент для связи с логгером
var chiefLoggerClient = web_sock.NewWebSockClient()

// контроллер сообщений состояния
var chiefHeartbeatCntrl = chief_configurator.NewHeartbeatController()

// очередь сообщений для отправки
var logQueue queue.Queue

// тикер отправки сообщений логгеру
var loggerTicker *time.Ticker

// сервер WS для подключения AODB провайдера
var aodbCntrl = aodb.NewController()

// контроллер FMTP каналов
var channelCntrl = chief_channel.NewChiefChannelServer()

var logQueueLocker sync.Mutex

const (
	appName    = "fdps-parking"
	appVersion = "2020-01-14 19:26"
)

var workWithDocker bool
var dockerVersion string

func initLoggers() {
	// логгер с web страничкой
	log_web.Initialize(log_web.LogWebSettings{
		StartHttp: false,
		//NetPort:      utils.ParkingWebDefaultPort,
		LogURLPath:   ut2.FmtpWebLogPath,
		Title:        appName,
		ShowSetts:    true,
		SettsURLPath: ut2.FmtpWebConfigPath,
	})
	logger.AppendLogger(log_web.WebLogger)
	ut2.AppendHandler(log_web.WebLogger)
	log_web.SetVersion(appVersion)

	// логгер с сохранением в файлы
	var ljackSetts log_ljack.LjackSettings
	ljackSetts.GenDefault()
	ljackSetts.FilesName = appName + ".log"
	if err := log_ljack.Initialize(ljackSetts); err != nil {
		logger.PrintfErr("Ошибка инициализации логгера lumberjack. Ошибка: %s", err.Error())
	}
	logger.AppendLogger(log_ljack.LogLjack)

	logger.AppendLogger(log_std.LogStd)

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC | log.Llongfile)
}

func initDockerInfo() {
	if dockerVersion, dockErr := utils.GetDockerVersion(); dockErr != nil {
		log_web.SetDockerVersion("???")
		logger.PrintfErr(dockErr.Error())
	} else {
		workWithDocker = true
		log_web.SetDockerVersion(dockerVersion)
	}
}

// постановка сообщения в очередь.
func processNewLogMsg(logMsg common.LogMessage) {
	logQueueLocker.Lock()
	defer logQueueLocker.Unlock()

	if logQueue.Len() >= maxLogQueueSize {
		logQueue.PopFront()
	}
	logQueue.PushBack(logMsg)

	web.AppendLog(logMsg)
}

// отправка состояния каналов web страничке
func sendChannelStatesToWeb(states []channel_state.ChannelState) {
	var chWebStates []web.ChannelStateWeb
	for _, nn := range states {
		var curColor string

		switch nn.DaemonState {
		case channel_state.ChannelStateStopped:
			curColor = web.StopColor
		case channel_state.ChannelStateError:
			curColor = web.ErrorColor
		case channel_state.ChannelStateOk:
			curColor = web.OkColor
		default:
			curColor = web.DefaultColor
		}

		chWebStates = append(chWebStates, web.ChannelStateWeb{
			ChannelID:   nn.ChannelID,
			ChannelURL:  nn.ChannelURL,
			LocalName:   nn.LocalName,
			RemoteName:  nn.RemoteName,
			DaemonState: nn.DaemonState,
			FmtpState:   nn.FmtpState,
			StateColor:  curColor,
		})
	}
	sort.Slice(chWebStates, func(i, j int) bool {
		return chWebStates[i].ChannelID < chWebStates[j].ChannelID
	})
	web.SetChannelStates(chWebStates)
}

// отправка состояния провайдеров web страничке
func sendProviderStatesToWeb(states []fdps.ProviderState) {
	var prvdWebStates []web.ProviderStateWeb
	for _, nn := range states {
		var curColor string

		switch nn.ProviderState {
		case fdps.ProviderStateOk:
			curColor = web.OkColor
		case fdps.ProviderStateError:
			curColor = web.ErrorColor
		default:
			curColor = web.DefaultColor
		}

		prvdWebStates = append(prvdWebStates, web.ProviderStateWeb{
			ProviderID:    nn.ProviderID,
			ProviderURL:   nn.ProviderURL,
			ProviderType:  nn.ProviderType,
			ProviderState: nn.ProviderState,
			StateColor:    curColor,
		})
	}

	sort.Slice(prvdWebStates, func(i, j int) bool {
		return prvdWebStates[i].ProviderID < prvdWebStates[j].ProviderID
	})
	web.SetProviderStates(prvdWebStates)
}

func main() {
	initLoggers()
	initDockerInfo()

	web.Initialize(utils.ChiefWebPath, 13002, new(web.ChiefPage))
	go web.Start()

	// запускаем таймер отправки логов
	loggerTicker = time.NewTicker(time.Second)

	chiefConfClient = chief_configurator.NewChiefClient()
	go chiefConfClient.Work()
	// отправляем запрос настроек контроллера
	go chiefConfClient.Start()

	go aodbCntrl.Work()
	go channelCntrl.Work()

	for {
		select {
		// получены настройки каналов
		case channelSetts := <-chiefConfClient.FmtpChannelSettsChan:
			go chiefHeartbeatCntrl.Work()
			channelCntrl.ChannelSettsChan <- channelSetts

		// получены настройки логгера
		case loggerSetts := <-chiefConfClient.LoggerSettsChan:
			go chiefLoggerClient.Work(web_sock.WebSockClientSettings{ServerAddress: "127.0.0.1", ServerPort: loggerSetts.LoggerPort, UrlPath: "/logger"})

		// получены настройки провайера AODB
		case providerSetts := <-chiefConfClient.ProviderSettsChan:
			var aodbSettings, oldiSettings []fdps.ProviderSettings

			for _, val := range providerSetts {
				if val.DataType == fdps.AODBProvider {
					aodbSettings = append(aodbSettings, val)
				} else if val.DataType == fdps.OLDIProvider {
					oldiSettings = append(oldiSettings, val)
				}
			}
			aodbCntrl.ProviderSettsChan <- aodbSettings

		// получено сообщение для журнала
		case chiefConfLog := <-chiefConfClient.LogChan:
			processNewLogMsg(chiefConfLog)

		// получены данные от провайдера AODB
		case aodbData := <-aodbCntrl.FromAODBDataChan:
			channelCntrl.IncomeAodbPacketChan <- aodbData

		// ошибка в работе клиента логгера
		case curErr := <-chiefLoggerClient.ErrorChan:
			var curLogMsg common.LogMessage
			if curErr == nil {
				web.SetLoggerState(true)
				curLogMsg = common.LogCntrlST(common.SeverityInfo, "Установлено сетевое соединение с сервисом записи событий журнала.")
			} else {
				web.SetLoggerState(false)
				curLogMsg = common.LogCntrlST(common.SeverityError,
					fmt.Sprintf("Возникла ошибка при работе с сервисом записи событий журнала. Ошибка: <%s>.", curErr.Error()))
				chiefHeartbeatCntrl.LoggerErrChan <- curErr
			}
			processNewLogMsg(curLogMsg)

		// считанные данные из клиента логгера
		case curData := <-chiefLoggerClient.ReceiveDataChan:
			var stateMsg chief_logger.LoggerStateMessage
			if unmErr := json.Unmarshal(curData, &stateMsg); unmErr == nil {
				chiefHeartbeatCntrl.LoggerStateChan <- stateMsg.LoggerState
			}

		// сработал тикер отправки логов
		case <-loggerTicker.C:
			if chiefLoggerClient.IsConnected() && logQueue.Len() > 0 {

				logMsg := chief_logger.LoggerMsgSlice{MessageHeader: chief_logger.MessageHeader{Header: chief_logger.LogMessagesHeader}}

				for nn := 0; nn < maxLogSendCount; nn++ {
					if logQueue.Len() > 0 {
						curLogMsg := common.LogMessage(logQueue.PopFront().(common.LogMessage))
						curLogMsg.ControllerIP = chief_configurator.ChiefCfg.IPAddr
						logMsg.Messages = append(logMsg.Messages, curLogMsg)
					} else {
						break
					}
				}

				if len(logMsg.Messages) > 0 {
					dataToSend, err := json.Marshal(logMsg)
					if err == nil {
						chiefLoggerClient.SendDataChan <- dataToSend
					} else {
						log.Println("Ошибка маршаллинга сообщения логов. Ошибка:", err.Error())
					}
				}
			}

			// сообщение о состоянии контроллера
		case curMsg := <-chiefHeartbeatCntrl.HeartbeatChannel:
			chiefConfClient.HeartbeatChan <- curMsg

		// AODB пакет от контроллера каналов
		case aodbData := <-channelCntrl.OutAodbPacketChan:
			aodbCntrl.ToAODBDataChan <- aodbData

		// сообщение журнала от контроллера AODB
		case curLog := <-aodbCntrl.LogChan:
			processNewLogMsg(curLog)

		// состояние провайдера AODB
		case curAodbState := <-aodbCntrl.ProviderStatesChan:
			chiefHeartbeatCntrl.AODBStateChan <- curAodbState

			sendProviderStatesToWeb(curAodbState)

		// сообщение журнала от контроллера каналов
		case curLog := <-channelCntrl.LogChan:
			processNewLogMsg(curLog)

		// получено состояние каналов
		case curStates := <-channelCntrl.ChannelStates:
			chiefHeartbeatCntrl.ChannelStateChan <- curStates

			sendChannelStatesToWeb(curStates)
		}
	}
}
