package main

import (
	"log"
	"sort"

	"fdps/fmtp/chief/aodb"
	"fdps/fmtp/chief/chief_web"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief/heartbeat"
	"fdps/fmtp/chief/oldi"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/chief_logger"
	"fdps/fmtp/web"

	"fdps/utils"
	"fdps/utils/logger"
	"fdps/utils/logger/log_ljack"
	"fdps/utils/logger/log_std"
	"fdps/utils/logger/log_web"
	"fdps/utils/web_sock"
)

const (
	appName    = "fdps-fmtp-chief"
	appVersion = "2020-06-10 15:29"
)

var workWithDocker bool
var dockerVersion string

var done = make(chan struct{}, 1)

func initLoggers() {
	// логгер с web страничкой
	log_web.Initialize(log_web.LogWebSettings{
		StartHttp:    false,
		LogURLPath:   utils.FmtpChiefWebLogPath,
		Title:        appName,
		ShowSetts:    true,
		SettsURLPath: utils.FmtpChiefWebConfigPath,
	})
	logger.AppendLogger(log_web.WebLogger)
	utils.AppendHandler(log_web.WebLogger)
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

	logger.AppendLogger(chief_logger.ChiefLog)
	go chief_logger.ChiefLog.Work()
}

func initDockerInfo() bool {
	if dockerVersion, dockErr := utils.GetDockerVersion(); dockErr != nil {
		log_web.SetDockerVersion("???")
		logger.PrintfErr(dockErr.Error())
		return false
	} else {
		workWithDocker = true
		log_web.SetDockerVersion(dockerVersion)
	}
	return true
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
	if initDockerInfo() == false {
		utils.InitFileBinUtils(
			utils.AppPath()+"/versions",
			utils.AppPath()+"/runningChannels",
			".exe",
			"fmtp_channel",
			"FMTP канал",
		)
	}

	web.Initialize("chief", 13002, new(web.ChiefPage))
	go web.Start()

	done := make(chan struct{})
	chief_web.Start(done)

	// клиент для связи с конфигуратором
	var chiefConfClient *chief_configurator.ChiefConfiguratorClient

	// сервер WS для подключения AODB провайдеров
	var aodbCntrl = aodb.NewController(done)

	// TCP сервер для подключения OLDI провайдеров
	var oldiCntrl = oldi.NewOldiController()

	// контроллер FMTP каналов
	var channelCntrl = chief_channel.NewChiefChannelServer(done, workWithDocker)

	chiefConfClient = chief_configurator.NewChiefClient(workWithDocker)

	go chiefConfClient.Work()
	// отправляем запрос настроек контроллера
	go chiefConfClient.Start()

	go aodbCntrl.Work()
	go oldiCntrl.Work()
	go channelCntrl.Work()

	for {
		select {
		// получены настройки каналов
		case channelSetts := <-chiefConfClient.FmtpChannelSettsChan:
			heartbeat.SetDockerVersion(dockerVersion)
			go heartbeat.Work()
			channelCntrl.ChannelSettsChan <- channelSetts

		// получены настройки логгера
		case loggerSetts := <-chiefConfClient.LoggerSettsChan:
			chief_logger.ChiefLog.SettsChan <- web_sock.WebSockClientSettings{
				ServerAddress:     "127.0.0.1",
				ServerPort:        loggerSetts.LoggerPort,
				UrlPath:           utils.FmtpLoggerWsPath,
				ReconnectInterval: 3,
			}

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

			oldiCntrl.ProviderSettsChan <- oldiSettings

		// получены данные от провайдера AODB
		case aodbData := <-aodbCntrl.FromAODBDataChan:
			channelCntrl.IncomeAodbPacketChan <- aodbData

		// AODB пакет от контроллера каналов
		case aodbData := <-channelCntrl.OutAodbPacketChan:
			aodbCntrl.ToAODBDataChan <- aodbData

		// состояние провайдера AODB
		case curAodbState := <-aodbCntrl.ProviderStatesChan:
			heartbeat.SetAodbProviderState(curAodbState)
			chief_web.SetAodbProviderStates(curAodbState)

		// получены данные от провайдера OLDI
		case oldiData := <-oldiCntrl.FromOldiDataChan:
			channelCntrl.IncomeOldiPacketChan <- oldiData

		// OLDI пакет от контроллера каналов
		case oldiData := <-channelCntrl.OutOldiPacketChan:
			oldiCntrl.ToOldiDataChan <- oldiData

		// состояние провайдера OLDI
		case curOldiState := <-oldiCntrl.ProviderStatesChan:
			heartbeat.SetOldiProviderState(curOldiState)
			chief_web.SetOldiProviderStates(curOldiState)

		// сообщение о состоянии контроллера
		case curMsg := <-heartbeat.HeartbeatCntrl.HeartbeatChannel:
			chiefConfClient.HeartbeatChan <- curMsg

		// получено состояние каналов
		case curChannelStates := <-channelCntrl.ChannelStates:
			heartbeat.SetChannelsState(curChannelStates)
			chief_web.SetChannelStates(curChannelStates)
		}
	}
}
