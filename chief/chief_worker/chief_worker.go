package chief_worker

import (
	"fdps/fmtp/chief/aodb"
	"fdps/fmtp/chief/chief_web"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief/heartbeat"
	"fdps/fmtp/chief/oldi"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/chief_logger"
	"fdps/utils"
	"fdps/utils/web_sock"
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

	chiefConfClient = chief_configurator.NewChiefClient(withDocker)

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

		case <-done:
			wg.Done()
			return
		}
	}
}
