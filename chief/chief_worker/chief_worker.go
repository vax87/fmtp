package chief_worker

import (
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/aodb"
	"fdps/fmtp/chief/chief_logger"
	"fdps/fmtp/chief/chief_web"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief/heartbeat"
	"fdps/fmtp/chief/oldi"
	"fdps/fmtp/chief/tky"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/chief_configurator"
	"sync"
)

var (
	akeyForSave string
	akeyForSend string
)

func Start(withDocker bool, dockerVersion string, done chan struct{}, wg *sync.WaitGroup) {

	// клиент для связи с конфигуратором
	var chiefConfClient *chief_configurator.ChiefConfiguratorClient

	// сервер WS для подключения AODB провайдеров
	var aodbCntrl = aodb.NewController(done)

	// TCP сервер для подключения OLDI провайдеров
	var oldiCntrl = oldi.NewOldiController()

	// контроллер FMTP каналов
	var channelCntrl = chief_channel.NewChiefChannelServer(done, withDocker)

	var channelStates []channel_state.ChannelState

	chiefConfClient = chief_configurator.NewChiefClient(withDocker)

	go chiefConfClient.Work()
	// отправляем запрос настроек контроллера
	go chiefConfClient.Start()

	go aodbCntrl.Work()
	go oldiCntrl.Work()
	go channelCntrl.Work()

	go tky.Work()

	for {
		select {
		// получены настройки каналов
		case channelSetts := <-chiefConfClient.FmtpChannelSettsChan:
			heartbeat.SetDockerVersion(dockerVersion)
			go heartbeat.Work()
			channelCntrl.ChannelSettsChan <- channelSetts

		// получены настройки логгера
		case loggerSetts := <-chiefConfClient.LoggerSettsChan:
			chief_logger.ChiefLog.SettsChan <- loggerSetts

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
			tky.SetAodbProviderStates(curAodbState)

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
			tky.SetOldiProviderState(curOldiState)

		// сообщение о состоянии контроллера
		case curMsg := <-heartbeat.HeartbeatCntrl.HeartbeatChannel:
			chiefConfClient.HeartbeatChan <- curMsg

		// получено состояние каналов
		case curChannelStates := <-channelCntrl.ChannelStates:
			heartbeat.SetChannelsState(curChannelStates)
			chief_web.SetChannelStates(curChannelStates)
			tky.SetChannelsState(curChannelStates)

			for _, curVal := range curChannelStates {

				foundState := false
				for chIdx, chVal := range channelStates {
					if chVal.LocalName == curVal.LocalName && chVal.RemoteName == curVal.RemoteName {

						foundState = true

						if chVal.DaemonState != curVal.DaemonState ||
							chVal.FmtpState != curVal.FmtpState {

							channelStates[chIdx] = curVal
							chief_logger.ChiefLog.ChannelStatesChan <- curVal
						}
					}
				}
				if foundState == false {
					channelStates = append(channelStates, curVal)
					chief_logger.ChiefLog.ChannelStatesChan <- curVal
				}
			}

		// получено сообщение журнала
		// case curLogMsg := <-channelCntrl.LogChan:
		// 	//chief_logger.ChiefLog.FmtpLogChan <- curLogMsg

		// 	switch curLogMsg.Severity {
		// 	case common.SeverityDebug:
		// 		logger.PrintfDebug("FMTP FORMAT", curLogMsg)
		// 	case common.SeverityInfo:
		// 		logger.PrintfInfo("FMTP FORMAT", curLogMsg)
		// 	case common.SeverityWarning:
		// 		logger.PrintfWarn("FMTP FORMAT", curLogMsg)
		// 	case common.SeverityError:
		// 		logger.PrintfErr("FMTP FORMAT", curLogMsg)
		// 	}

		case <-done:
			wg.Done()
			return
		}
	}
}
