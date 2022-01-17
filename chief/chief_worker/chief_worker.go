package chief_worker

import (
	"fdps/fmtp/channel/channel_settings"
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

	// TCP сервер для подключения OLDI провайдеров
	//var oldiTcpCntrl = oldi.NewOldiTcpController()
	var oldiGrpcCntrl = oldi.NewOldiGrpcController()

	// контроллер FMTP каналов
	var channelCntrl = chief_channel.NewChiefChannelServer(done, withDocker)

	chiefConfClient = chief_configurator.NewChiefClient(withDocker)

	go chiefConfClient.Work()
	// отправляем запрос настроек контроллера
	go chiefConfClient.Start()

	//go oldiTcpCntrl.Work()
	go oldiGrpcCntrl.Work()
	go channelCntrl.Work()

	go tky.Work()

	chief_state.CommonChiefState.ControllerVersion = version.Release
	chief_state.CommonChiefState.DockerVersion = dockerVersion

	for {
		select {

		// настройки контроллера изменены
		case <-chiefConfClient.ChiefSettChangedChan:
			channelCntrl.ChannelSettsChan <- channel_settings.ChannelSettingsWithPort{
				ChSettings: chief_configurator.ChiefCfg.ChannelSetts,
				ChPort:     chief_configurator.ChiefCfg.ChannelsPort,
			}

			//oldiTcpCntrl.SettsChan <- chief_configurator.ChiefCfg.ProviderSettings(chief_settings.OLDIProvider)
			oldiGrpcCntrl.SettsChan <- chief_configurator.ChiefCfg.ProviderSettings(chief_settings.OLDIProvider)

			chief_logger.ChiefLog.SettsChan <- chief_configurator.ChiefCfg.LoggerSetts

		case saveStates := <-chiefConfClient.SaveStatesToDbChan:
			chief_logger.ChiefLog.SetWriteStatesToDb(saveStates)

		// получены данные от провайдера OLDI
		//case oldiData := <-oldiTcpCntrl.FromOldiDataChan:
		case fdpsData := <-oldiGrpcCntrl.FromFdpsChan:
			channelCntrl.FromFdpsPacketChan <- fdpsData

		// OLDI пакет от контроллера каналов
		case oldiData := <-channelCntrl.ToFdpsPacketChan:
			//oldiTcpCntrl.ToOldiDataChan <- oldiData
			oldiGrpcCntrl.ToFdpsChan <- oldiData

		case <-done:
			wg.Done()
			return
		}
	}
}
