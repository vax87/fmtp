package chief_worker

import (
	"fdps/fmtp/chief/aodb"
	"fdps/fmtp/chief/chief_logger"
	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
	"fdps/fmtp/chief/oldi"
	"fdps/fmtp/chief/tky"
	"fdps/fmtp/chief/version"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/chief_configurator"
	"sync"
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

	go tky.Work()

	chief_state.CommonChiefState.ControllerVersion = version.Release
	chief_state.CommonChiefState.DockerVersion = dockerVersion

	for {
		select {
		// получены настройки каналов
		case channelSetts := <-chiefConfClient.FmtpChannelSettsChan:
			channelCntrl.ChannelSettsChan <- channelSetts

		// получены настройки логгера
		case loggerSetts := <-chiefConfClient.LoggerSettsChan:
			chief_logger.ChiefLog.SettsChan <- loggerSetts

		// получены настройки провайера AODB
		case providerSetts := <-chiefConfClient.ProviderSettsChan:
			var aodbSettings, oldiSettings []chief_settings.ProviderSettings

			for _, val := range providerSetts {
				if val.DataType == chief_settings.AODBProvider {
					aodbSettings = append(aodbSettings, val)
				} else if val.DataType == chief_settings.OLDIProvider {
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

		// получены данные от провайдера OLDI
		case oldiData := <-oldiCntrl.FromOldiDataChan:
			channelCntrl.IncomeOldiPacketChan <- oldiData

		// OLDI пакет от контроллера каналов
		case oldiData := <-channelCntrl.OutOldiPacketChan:
			oldiCntrl.ToOldiDataChan <- oldiData

		case <-done:
			wg.Done()
			return
		}
	}
}
