package chief_channel

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
	pb "fdps/fmtp/chief/proto/fmtp"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/fmtp"
	"fdps/fmtp/fmtp_logger"
	"fdps/go_utils/logger"
	"fdps/go_utils/web_sock"
	"fdps/utils"
)

const (
	srvStateKey        = "WS FMTP каналов. Состояние:"
	srvStateOkValue    = "Запущен."
	srvStateErrorValue = "Не запущен."
	startOldiMsgTag    = "<msg>"
	endOldiMsgTag      = "</msg>"
	startOldiAccTag    = "<acc>"
	endOldiAccTag      = "</acc>"
)

// сведения об исполняемом файле канала
type channelBin struct {
	filePath string        // путь к исполняемому файлу
	killChan chan struct{} // канал, исользуемый для завершения выполнения
	//startPerform bool          // выполнене запуск канала
}

// состояния каналов FMTP с отметкой времени
type сhannelStateTime struct {
	channel_state.ChannelState           //состояние
	Time                       time.Time // время получения сообщений о состоянии
}

// ChiefChannelServer контроллер FMTP каналов
type ChiefChannelServer struct {
	ChannelSettsChan chan channel_settings.ChannelSettingsWithPort // канал для приема настроек каналов и порта для взаимодействия с каналами
	channelSetts     channel_settings.ChannelSettingsWithPort      // текущие настройки каналов и орт для связи с каналами
	statesSendTicker *time.Ticker                                  // тикер отправки состояния каналов

	FromFdpsPacketChan chan pb.Msg // канал для приема сообщений от провайдера OLDI
	ToFdpsPacketChan   chan pb.Msg // канал для отправки сообщений провайдеру OLDI

	ChannelBinMap *sync.Map // ключ - идентификатор каналаб значение типа сhannelBin

	killerChan chan struct{} // канал, по которому передается сигнал о завершение работы канала

	wsServer *web_sock.WebSockServer

	wsClients map[int]*websocket.Conn

	chStates map[int]сhannelStateTime // ключ - ID канала

	oldiIdent  int // идентификатор сообщения, отправляемого OLDI cервису
	withDocker bool
}

// состояние FMTP канала, при котором ему отправляем сообщений от AODB
var chValidSt = fmtp.DataReady
var chValidStStr = chValidSt.ToString()

// NewChiefChannelServer конструктор
func NewChiefChannelServer(done chan struct{}, workWithDocker bool) *ChiefChannelServer {
	return &ChiefChannelServer{
		ChannelSettsChan: make(chan channel_settings.ChannelSettingsWithPort, 10),
		channelSetts: channel_settings.ChannelSettingsWithPort{
			ChSettings: make([]channel_settings.ChannelSettings, 0),
			ChPort:     0,
		},
		statesSendTicker:   time.NewTicker(time.Second),
		FromFdpsPacketChan: make(chan pb.Msg, 1024),
		ToFdpsPacketChan:   make(chan pb.Msg, 1024),
		killerChan:         make(chan struct{}),
		ChannelBinMap:      new(sync.Map),
		wsServer:           web_sock.NewWebSockServer(done),
		wsClients:          make(map[int]*websocket.Conn),
		chStates:           make(map[int]сhannelStateTime),
		oldiIdent:          1,
		withDocker:         workWithDocker,
	}
}

// Work реализация работы
func (cc *ChiefChannelServer) Work() {
	go cc.wsServer.Work("/" + utils.FmtpChannelWsUrlPath)

	for {
		select {
		// получены новые настройки каналов
		case newSetts := <-cc.ChannelSettsChan:

			var needToStopIds, needToStartIds []int // идентификаторы каналов, которые необходимо остановить/запустить

			if cc.channelSetts.ChPort != newSetts.ChPort {
				cc.wsServer.SettingsChan <- web_sock.WebSockServerSettings{Port: newSetts.ChPort}

				//останавливаем и запускаем все каналы
				for _, val := range cc.channelSetts.ChSettings {
					if val.IsWorking {
						needToStopIds = append(needToStopIds, val.Id)
					}
				}

				//останавливаем и запускаем все каналы
				for _, val := range newSetts.ChSettings {
					cc.initChannelState(val)

					if val.IsWorking {
						needToStartIds = append(needToStartIds, val.Id)
					}
				}
			} else if !reflect.DeepEqual(cc.channelSetts.ChSettings, newSetts.ChSettings) {

				var oldIt, newIt channel_settings.ChannelSettings

				for _, newIt = range newSetts.ChSettings {
					newInOld := false
				NEWL:
					for _, oldIt = range cc.channelSetts.ChSettings {
						if oldIt.Id == newIt.Id {
							newInOld = true
							break NEWL
						}
					}

					// есть в новых, есть в старых
					if newInOld {
						if oldIt != newIt {
							if oldIt.IsWorking {
								needToStopIds = append(needToStopIds, oldIt.Id)
							}
							if newIt.IsWorking {
								needToStartIds = append(needToStartIds, newIt.Id)
							}
							cc.initChannelState(newIt)
						}
					} else { // есть в новых, нет в старых
						if newIt.IsWorking {
							needToStartIds = append(needToStartIds, newIt.Id)
						}
						cc.initChannelState(newIt)
					}
				}

				for _, oldIt = range cc.channelSetts.ChSettings {
					oldInNew := false
				OLDL:
					for _, newIt = range newSetts.ChSettings {
						if newIt.Id == oldIt.Id {
							oldInNew = true
							break OLDL
						}
					}
					// есть в старых, нет в новых
					if !oldInNew {
						if oldIt.IsWorking {
							needToStopIds = append(needToStopIds, oldIt.Id)
						}
					}
				}
			}
			// если просто cc.channelSetts = newSetts написать, то по приходу новых настроек cc.channelSetts. уже будет ссылаться на них
			cc.channelSetts.ChSettings = append([]channel_settings.ChannelSettings(nil), newSetts.ChSettings...)
			cc.channelSetts.ChPort = newSetts.ChPort

			// останавливаем каналы FMTP
			if len(needToStopIds) > 0 {
				cc.stopChannelsByIDs(needToStopIds)
			}
			// костыль - не успевает удалиться старый контейнер, при создании нового - конфликн имен
			time.AfterFunc(2*time.Second, func() {
				// запускаем каналы FMTP
				if len(needToStartIds) > 0 {
					cc.startChannelsByIDs(needToStartIds)
				}
			})

		// получен новый пакет от провайдера OLDI
		case oldiPkg := <-cc.FromFdpsPacketChan:
			cc.ProcessOldiPacket(oldiPkg)

		// получены данные от WS сервера
		case curWsPkg := <-cc.wsServer.ReceiveDataChan:
			var curHdr HeaderMsg
			var unmErr error
			if unmErr = json.Unmarshal(curWsPkg.Data, &curHdr); unmErr == nil {
				switch curHdr.Header {

				case RequestSettingsHeader:
					var reqSettsMsg SettingsRequestMsg
					if err := json.Unmarshal(curWsPkg.Data, &reqSettsMsg); err == nil {
						cc.wsClients[reqSettsMsg.ChannelID] = curWsPkg.Sock

						var channelSetts channel_settings.ChannelSettings
					SETTSL:
						for _, curSetts := range cc.channelSetts.ChSettings {
							if curSetts.Id == reqSettsMsg.ChannelID {
								channelSetts = curSetts
								break SETTSL
							}
						}

						if settsData, errMarsh := json.Marshal(CreateSettingsAnswerMsg(channelSetts)); errMarsh == nil {
							cc.wsServer.SendDataChan <- web_sock.WsPackage{Data: settsData, Sock: cc.wsClients[channelSetts.Id]}
						}
					}

				case ChannelHeartbeatHeader:
					var curHbtMsg ChannelHeartbeatMsg
					if err := json.Unmarshal(curWsPkg.Data, &curHbtMsg); err == nil {
						cc.chStates[curHbtMsg.ChannelID] = сhannelStateTime{ChannelState: curHbtMsg.ChannelState, Time: time.Now()}
					}

					// если каналы были запущены на момент старта контроллера, то
					// при получении Heartbeat проверяем клиентские сообщения
					if _, ok := cc.wsClients[curHbtMsg.ChannelID]; !ok {
						cc.wsClients[curHbtMsg.ChannelID] = curWsPkg.Sock
					}

				case ChannelLogHeader:
					var curLogMsg ChannelLogMsg
					if err := json.Unmarshal(curWsPkg.Data, &curLogMsg); err == nil {
						switch curLogMsg.LogMessage.Severity {
						case fmtp_logger.SeverityDebug:
							logger.PrintfDebug("FMTP FORMAT %#v", curLogMsg.LogMessage)
						case fmtp_logger.SeverityInfo:
							logger.PrintfInfo("FMTP FORMAT %#v", curLogMsg.LogMessage)
						case fmtp_logger.SeverityWarning:
							logger.PrintfWarn("FMTP FORMAT %#v", curLogMsg.LogMessage)
						case fmtp_logger.SeverityError:
							logger.PrintfErr("FMTP FORMAT %#v", curLogMsg.LogMessage)
						}
					}

				case ChannelMessageHeader:

					var dataMsg DataMsg
					if err := json.Unmarshal(curWsPkg.Data, &dataMsg); err == nil {
						var localAtc, remoteAtc string
						var channelType string
						for _, val := range cc.channelSetts.ChSettings {
							if val.Id == dataMsg.ChannelID {
								localAtc = val.LocalATC
								remoteAtc = val.RemoteATC
								channelType = val.DataType
								break
							}
						}

						if channelType == chief_settings.OLDIProvider {
							//var oldiPkg fdps.FdpsOldiPackage
							var oldiPkg pb.Msg
							oldiPkg.Id = strconv.Itoa(cc.oldiIdent)
							oldiPkg.Cid = localAtc
							oldiPkg.RemoteAtc = remoteAtc
							oldiPkg.Txt = dataMsg.Text

							cc.oldiIdent++
							if cc.oldiIdent > 1000 {
								cc.oldiIdent = 1
							}
							logger.PrintfInfo("FMTP FORMAT %#v", fmtp_logger.LogCntrlSDT(fmtp_logger.SeverityInfo, chief_settings.OLDIProvider,
								fmt.Sprintf("Плановой подсистеме отправлено сообщение: %s.", dataMsg.Text)))
							cc.ToFdpsPacketChan <- oldiPkg
						}
					}
				}

			} else {
				logger.PrintfErr("Ошибка разбора сообщения от приложения FMTP канала. Ошибка: %s.", unmErr)
			}

		// получен подключенный клиент от WS сервера
		case curClnt := <-cc.wsServer.ClntConnChan:
			logger.PrintfDebug("WS сервер для взаимодействия с FMTP каналами. Подключен клиент с адресом: %s.", curClnt.RemoteAddr().String())
			//userhub_web.ClientConn(userhub_web.FromWebSock(curClnt))

		// получен отключенный клиент от WS сервера
		case curClnt := <-cc.wsServer.ClntDisconnChan:
			logger.PrintfDebug("WS сервер для взаимодействия с FMTP каналами. Отключен клиент с адресом: %s.", curClnt.RemoteAddr().String())
			//userhub_web.ClientDisconn(userhub_web.FromWebSock(curClnt))

		// получен отклоненный клиент от WS сервера
		case curClnt := <-cc.wsServer.ClntRejectChan:
			logger.PrintfDebug("WS сервер для взаимодействия с FMTP каналами. Отклонен клиент с адресом: %s.", curClnt.RemoteAddr().String())

		// получена ошибка от WS сервера
		case wsErr := <-cc.wsServer.ErrorChan:
			logger.PrintfErr("Возникла ошибка при работе WS сервера для взаимодействия с FMTP каналами. Ошибка: %s.", wsErr.Error())

		// получено состояние работоспособности WS сервера для связи с FMTP каналами
		case connState := <-cc.wsServer.StateChan:
			switch connState {
			case web_sock.ServerTryToStart:
				logger.PrintfDebug("Запускаем WS сервер для взаимодействия с FMTP каналами. Порт: %d. Path: \"%s\".", cc.channelSetts.ChPort, utils.FmtpChannelWsUrlPath)
				logger.SetDebugParam(srvStateKey, fmt.Sprintf("%s Порт: %d. Path: \"%s\"", srvStateOkValue, cc.channelSetts.ChPort, utils.FmtpChannelWsUrlPath), logger.StateOkColor)
			case web_sock.ServerError:
				logger.SetDebugParam(srvStateKey, srvStateErrorValue, logger.StateErrorColor)
			}

		case <-cc.statesSendTicker.C:
			// удаляем из мэпки состояний старые состояния
			for channelId, val := range cc.chStates {
				if val.Time.Add(10 * channel_state.StateSendInterval).Before(time.Now()) {
					// если состояния удалили и в настройках канал должен быть запущен, то добавляем состояние
					for _, setts := range cc.channelSetts.ChSettings {
						if setts.Id == channelId && setts.IsWorking {
							delete(cc.chStates, channelId)
							cc.initChannelState(setts)
							break
						}
					}
				}
			}

			// отправляем heartbeat контроллеру
			var channelStates []channel_state.ChannelState
			for key := range cc.chStates {
				channelStates = append(channelStates, cc.chStates[key].ChannelState)
			}
			chief_state.SetChannelsState(channelStates)
		}
	}
}

// ProcessOldiPacket обработка пакета OLDI
func (cc *ChiefChannelServer) ProcessOldiPacket(pkg pb.Msg) {
	var channelID int = -1

	for _, val := range cc.channelSetts.ChSettings {

		// от OLDI только cid приходит
		if pkg.Cid == "" {
			if val.DataType == chief_settings.OLDIProvider && val.RemoteATC == pkg.RemoteAtc {
				channelID = val.Id
				break
			}
		} else { // данные от провайдера тренажера (все RemoteAtc одинаковые, LocalAtc не пустой)
			if val.DataType == chief_settings.OLDIProvider && val.LocalATC == pkg.Cid && val.RemoteATC == pkg.RemoteAtc {
				channelID = val.Id
				break
			}
		}
	}

	if channelID != -1 {
		if sock, ok := cc.wsClients[channelID]; ok {
			if oldiData, mrshErr := json.Marshal(CreateChiefDataMsg(channelID, pkg.Txt)); mrshErr != nil {
				logger.PrintfErr("Ошибка формирования сообщения для FMTP канала. Ошибка: %v.", mrshErr)
			} else {
				if cc.chStates[channelID].ChannelState.FmtpState == chValidStStr {
					cc.wsServer.SendDataChan <- web_sock.WsPackage{Data: oldiData, Sock: sock}
				}
			}
		} else {
			logger.PrintfErr("Не найдено WebSocket соединение канала для отправки сообщения.")
		}
	} else {
		logger.PrintfErr("Не найден FMTP канал для отправки сообщения. CID (remote ATC): %s.", pkg.RemoteAtc)
	}
}

// останавливаем каналы с указанным ID
func (cc *ChiefChannelServer) stopChannelsByIDs(idsToStop []int) {
STOPL:
	for _, stopID := range idsToStop {
		if val, ok := cc.ChannelBinMap.Load(stopID); ok {
			val.(channelBin).killChan <- struct{}{}
			<-cc.killerChan

			if !cc.withDocker {
				if err := utils.RemoveBinary(val.(channelBin).filePath); err != nil {
					logger.PrintfErr("%v", err)
					continue STOPL
				}
			}
			close(val.(channelBin).killChan)
			cc.ChannelBinMap.Delete(stopID)
		}
	}
}

// запускаем каналы с указанным ID
func (cc *ChiefChannelServer) startChannelsByIDs(idsToStart []int) {
STARTL:
	for _, startID := range idsToStart {
	STARTL2:
		for _, newIt := range cc.channelSetts.ChSettings {
			if startID == newIt.Id {
				if cc.withDocker {
					cc.ChannelBinMap.Store(startID, channelBin{killChan: make(chan struct{})})

					if val, ok := cc.ChannelBinMap.Load(startID); ok {
						go cc.startChannelContainer(newIt, val.(channelBin))
					}

				} else {
					channelFilePath, err := utils.CopyBinary(newIt.Version, newIt.Id, newIt.LocalATC, newIt.RemoteATC)
					if err != nil {
						logger.PrintfErr("%v", err)
						continue STARTL
					}
					cc.ChannelBinMap.Store(startID, channelBin{filePath: channelFilePath, killChan: make(chan struct{})})

					if val, ok := cc.ChannelBinMap.Load(startID); ok {
						go cc.startChannelProcess(channelFilePath, newIt, val.(channelBin))
					}
				}
				break STARTL2
			}
		}
	}
}

// запуск исполняемого файла fmtp канала
func (cc *ChiefChannelServer) startChannelProcess(channelFilePath string, chSett channel_settings.ChannelSettings, binInfo channelBin) {
	cmd := exec.Command(channelFilePath, strconv.Itoa(cc.channelSetts.ChPort), strconv.Itoa(chSett.Id), chSett.LocalATC,
		chSett.RemoteATC, chSett.DataType, chSett.URLPath, strconv.Itoa(chSett.URLPort))

	if err := cmd.Start(); err != nil {
		logger.PrintfErr("Ошибка запуска приложения FMTP канала. Исполняемый файл: %s. Иденификатор канала: %d. Ошибка: %v.",
			channelFilePath, chSett.Id, err)
		return
	}

	logger.PrintfDebug("Запущено приложения FMTP канала. Исполняемый файл: %s. Иденификатор канала: %d.",
		channelFilePath, chSett.Id)

	// канал, в который ошибка завершения отправкится
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.PrintfErr("Нештатное завершение приложения FMTP канала. Исполняемый файл: %s. Идентификатор канала: %d. Ошибка: %s.",
				channelFilePath, chSett.Id, err.Error())

			if val, ok := cc.ChannelBinMap.Load(chSett.Id); ok {
				close(val.(channelBin).killChan)
				cc.ChannelBinMap.Delete(chSett.Id)
			}

			if _, ok := cc.chStates[chSett.Id]; ok {
				curState := cc.chStates[chSett.Id]
				curState.ChannelState.DaemonState = channel_state.ChannelStateError
				cc.chStates[chSett.Id] = curState
			}
		}
		cc.checkChannelWorking(chSett.Id)
		return

	case <-binInfo.killChan:
		if err := cmd.Process.Kill(); err != nil {
			logger.PrintfErr("Ошибка завершения выполнения приложения FMTP канала. Ошибка: %v.", err)
		} else {
			logger.PrintfDebug("Штатное завершение приложения FMTP канала. Исполняемый файл: %s. Идентификатор канала: %d.",
				channelFilePath, chSett.Id)
		}
		time.Sleep(1 * time.Second)
		cc.killerChan <- struct{}{}
		return
	}
}

// запуск docker контейнера fmtp канала
func (cc *ChiefChannelServer) startChannelContainer(chSett channel_settings.ChannelSettings, binInfo channelBin) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.PrintfErr("Ошибка создания клиента сервиса docker. Ошибка: %v", err)
		if val, ok := cc.ChannelBinMap.Load(chSett.Id); ok {
			close(val.(channelBin).killChan)
			cc.ChannelBinMap.Delete(chSett.Id)
		}
		return
	} else {
		defer cli.Close()
		cli.NegotiateAPIVersion(ctx)
	}

	curContainerName := "fmtp_channel_" + chSett.LocalATC + "_" + chSett.RemoteATC + "_" + strconv.Itoa(chSett.Id)
	var imageName string

	if len(chief_configurator.ChiefCfg.DockerRegistry) != 0 {
		imageName = chief_configurator.ChiefCfg.DockerRegistry + "/"
	}
	imageName += chief_configurator.ChannelImageName + ":" + chSett.Version

	resp, crErr := cli.ContainerCreate(ctx,
		&container.Config{
			Image: imageName,
			Cmd: []string{"/fdps/fmtp_channel", strconv.Itoa(cc.channelSetts.ChPort), strconv.Itoa(chSett.Id), chSett.LocalATC,
				chSett.RemoteATC, chSett.DataType, chSett.URLPath, strconv.Itoa(chSett.URLPort)},
		},
		&container.HostConfig{
			Resources: container.Resources{
				CPUCount:   1,
				CPUPercent: 10,
				Memory:     100 * 1 << 20, // 100 MB
			},
			NetworkMode:   "host",
			RestartPolicy: container.RestartPolicy{Name: "no"},
			AutoRemove:    true,
		},
		&network.NetworkingConfig{},
		nil,
		curContainerName)

	if crErr != nil {
		logger.PrintfErr("Ошибка создания docker контейнера %s. Используемый образ: %s. Ошибка: %v.", curContainerName, imageName, crErr)
		if val, ok := cc.ChannelBinMap.Load(chSett.Id); ok {
			close(val.(channelBin).killChan)
			cc.ChannelBinMap.Delete(chSett.Id)
		}
		return
	} else {
		logger.PrintfDebug("Создан docker контейнер %s. Используемый образ: %s.", curContainerName, imageName)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logger.PrintfErr("Ошибка запуска docker контейнера %s. Ошибка: %v.", curContainerName, err)
		if val, ok := cc.ChannelBinMap.Load(chSett.Id); ok {
			close(val.(channelBin).killChan)
			cc.ChannelBinMap.Delete(chSett.Id)
		}
		return
	} else {
		logger.PrintfDebug("Запущен docker контейнер %s.", curContainerName)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNextExit)

	for {
		select {
		case cntErr := <-errCh:
			if cntErr != nil {
				logger.PrintfErr("Ошибка в работе docker контейнера %s. Ошибка: %v.", curContainerName, cntErr)
			}

			if val, ok := cc.ChannelBinMap.Load(chSett.Id); ok {
				close(val.(channelBin).killChan)
				cc.ChannelBinMap.Delete(chSett.Id)
			}
			return

		case curStatus := <-statusCh:
			if curStatus.Error != nil {
				logger.PrintfErr("Изменен статус docker контейнера %s. Статус: %v. Ошибка: %v.", curContainerName, curStatus.StatusCode, curStatus.Error.Message)
			} else {
				logger.PrintfErr("Изменен статус docker контейнера %s. Статус: %v.", curContainerName, curStatus.StatusCode)
			}
			if val, ok := cc.ChannelBinMap.Load(chSett.Id); ok {
				close(val.(channelBin).killChan)
				cc.ChannelBinMap.Delete(chSett.Id)
			}

			cc.checkChannelWorking(chSett.Id)
			return

		case <-binInfo.killChan:
			logger.PrintfDebug("Команда завершить docker контейнер %s.", curContainerName)

			//var stopDur time.Duration = 100 * time.Millisecond

			if stopErr := cli.ContainerStop(ctx, resp.ID, nil); stopErr == nil {
				logger.PrintfDebug("Остановлен docker контейнер %s.", curContainerName)

				// if rmErr := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{}); rmErr == nil {
				// 	logger.PrintfInfo("Удален docker контейнер %s.", curContainerName)
				// } else {
				// 	logger.PrintfErr("Ошибка удаления docker контейнера %s.Ошибка: %v", curContainerName, rmErr)
				// }
			} else {
				logger.PrintfErr("Ошибка остановки docker контейнера %s. Ошибка %v", curContainerName, stopErr)
			}
			cc.killerChan <- struct{}{}
			return
		}
	}
}

func (cc *ChiefChannelServer) initChannelState(chSett channel_settings.ChannelSettings) {
	var curChannelState string
	if chSett.IsWorking {
		curChannelState = channel_state.ChannelStateError
	} else {
		curChannelState = channel_state.ChannelStateStopped
	}

	// добавляем состояние канала
	cc.chStates[chSett.Id] = сhannelStateTime{ChannelState: channel_state.ChannelState{
		ChannelID:   chSett.Id,
		LocalName:   chSett.LocalATC,
		RemoteName:  chSett.RemoteATC,
		DaemonState: curChannelState,
		FmtpState:   "disabled",
		ChannelURL:  fmt.Sprintf("http://%s:%d/%s", chSett.URLAddress, chSett.URLPort, chSett.URLPath),
	},
		Time: time.Now()}
}

// проверка работыканала. Если возникла ошбка, проверяем в настройках, должен ли работать канал
// если должен, то запускаем его
func (cc *ChiefChannelServer) checkChannelWorking(channelId int) {

	for _, setts := range cc.channelSetts.ChSettings {
		if setts.Id == channelId && setts.IsWorking {
			if _, ok := cc.ChannelBinMap.Load(setts.Id); !ok {
				logger.PrintfWarn("Необходим перезапуск канала с ID = %d", channelId)

				// костыль - не успевает удалиться старый контейнер, при создании нового - конфликн имен
				time.AfterFunc(2*time.Second, func() {
					cc.startChannelsByIDs([]int{channelId})
				})
			}
			break
		}
	}
}
