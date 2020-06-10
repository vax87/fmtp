package main

import (
	"log"
	"sort"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/aodb"
	"fdps/fmtp/chief/chief_web"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief/heartbeat"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/chief_logger"
	"fdps/fmtp/utils"
	"fdps/fmtp/web"

	ut2 "fdps/utils"
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
var chiefLogger = chief_logger.NewChiefLogger(done)

func initLoggers() {
	// логгер с web страничкой
	log_web.Initialize(log_web.LogWebSettings{
		StartHttp:    false,
		LogURLPath:   ut2.FmtpChiefWebLogPath,
		Title:        appName,
		ShowSetts:    true,
		SettsURLPath: ut2.FmtpChiefWebConfigPath,
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

	logger.AppendLogger(chiefLogger)
	go chiefLogger.Work()

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
	//web.SetChannelStates(chWebStates)
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

	done := make(chan struct{})
	chief_web.Start(done)

	// клиент для связи с конфигуратором
	var chiefConfClient *chief_configurator.ChiefConfigutarorClient

	// сервер WS для подключения AODB провайдера
	var aodbCntrl = aodb.NewController(done)

	// контроллер FMTP каналов
	var channelCntrl = chief_channel.NewChiefChannelServer(workWithDocker)

	chiefConfClient = chief_configurator.NewChiefClient(workWithDocker)

	go chiefConfClient.Work()
	// отправляем запрос настроек контроллера
	go chiefConfClient.Start()

	go aodbCntrl.Work()
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
			chiefLogger.SettsChan <- web_sock.WebSockClientSettings{
				ServerAddress:     "127.0.0.1",
				ServerPort:        loggerSetts.LoggerPort,
				UrlPath:           ut2.FmtpLoggerWsPath,
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

		// получены данные от провайдера AODB
		case aodbData := <-aodbCntrl.FromAODBDataChan:
			channelCntrl.IncomeAodbPacketChan <- aodbData

		// сообщение о состоянии контроллера
		case curMsg := <-heartbeat.HeartbeatCntrl.HeartbeatChannel:
			chiefConfClient.HeartbeatChan <- curMsg

		// AODB пакет от контроллера каналов
		case aodbData := <-channelCntrl.OutAodbPacketChan:
			aodbCntrl.ToAODBDataChan <- aodbData

		// состояние провайдера AODB
		case curAodbState := <-aodbCntrl.ProviderStatesChan:
			heartbeat.SetAodbProviderState(curAodbState)

		// сообщение журнала от контроллера каналов
		case _ = <-channelCntrl.LogChan:
			//processNewLogMsg(curLog)

		// получено состояние каналов
		case curChannelStates := <-channelCntrl.ChannelStates:
			heartbeat.SetChannelsState(curChannelStates)
		}
	}
}
