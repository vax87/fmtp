package chief_worker

import (
	"fmtp/channel/channel_settings"
	"fmtp/chief/chief_logger"
	"fmtp/chief/chief_metrics"
	"fmtp/chief/chief_state"
	"fmtp/chief/oldi"
	"fmtp/chief/tky"
	"fmtp/chief/version"
	"fmtp/chief_channel"
	"fmtp/configurator"
	"sync"

	"lemz.com/fdps/prom_metrics"
)

func Start(withDocker bool, dockerVersion string, done chan struct{}, wg *sync.WaitGroup) {

	// клиент для связи с конфигуратором
	var chiefConfClient *configurator.ChiefConfiguratorClient

	// grpc сервер для подключения OLDI провайдеров
	var oldiGrpcCntrl = oldi.NewOldiGrpcController()

	// контроллер FMTP каналов
	var channelCntrl = chief_channel.NewChiefChannelServer(done, withDocker)

	chiefConfClient = configurator.NewChiefClient(withDocker)

	go chiefConfClient.Work()
	// отправляем запрос настроек контроллера
	go chiefConfClient.Start()

	go oldiGrpcCntrl.Work()
	go channelCntrl.Work()

	go tky.Work()

	chief_state.CommonChiefState.ControllerVersion = version.Release
	chief_state.CommonChiefState.DockerVersion = dockerVersion

	var metricsCntrl = chief_metrics.NewChiefMetricsCntrl()
	go metricsCntrl.Run()

	//!!! только один раз можно вызвать
	metricsCntrl.SettsChan <- prom_metrics.PusherSettings{
		PusherIntervalSec: 1,
		GatewayUrl:        "http://192.168.1.24:9100", // from lemz
		//GatewayUrl:       "http://127.0.0.1:9100",	// from home
		GatewayJob:       "fmtp",
		CollectNamespace: "fmtp",
		CollectSubsystem: "controller",
		CollectLabels:    map[string]string{"host": "192.168.10.219"},
	}
	//!!!

	for {
		select {

		// настройки контроллера изменены
		case <-chiefConfClient.ChiefSettChangedChan:
			channelCntrl.ChannelSettsChan <- channel_settings.ChannelSettingsWithPort{
				ChSettings: configurator.ChiefCfg.ChannelSetts,
				ChPort:     configurator.ChiefCfg.ChannelsPort,
			}

			oldiGrpcCntrl.SettsChangedChan <- struct{}{}

			chief_logger.ChiefLog.SettsChangedChan <- struct{}{}

		// получены данные от провайдера OLDI
		case fdpsData := <-oldiGrpcCntrl.FromFdpsChan:
			channelCntrl.FromFdpsPacketChan <- fdpsData

		// OLDI пакет от контроллера каналов
		case oldiData := <-channelCntrl.ToFdpsPacketChan:
			oldiGrpcCntrl.ToFdpsChan <- oldiData

		case <-done:
			wg.Done()
			return
		}
	}
}
