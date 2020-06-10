package chief_channel

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/fmtp"
	"fdps/fmtp/logger/common"
	"fdps/fmtp/utils"
)

// сведения об исполняемом файле канала
type сhannelBin struct {
	filePath string        // путь к исполняемому файлу
	killChan chan struct{} // канал, исользуемый для завершения выполнения
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
	ChannelStates    chan []channel_state.ChannelState             //канал для передачи состояний каналов

	IncomeAodbPacketChan chan []byte // канал для приема сообщений от провайдера AODB
	OutAodbPacketChan    chan []byte // канал для отправки сообщений провайдеру AODB

	LogChan       chan common.LogMessage // канал для передачи сообщений журнала
	ChannelBinMap map[int]сhannelBin     // ключ - идентификатор канала
	killerChan    chan struct{}          // канал, по которому передается сигнал о завершение работы канала

	//ws *web_sock.WebSockServer

	//wsClients map[int]*web_sock.WebSockServerSocket
	wsClients map[int]*websocket.Conn

	chStates map[int]сhannelStateTime // ключ - ID канала

	aodbIdent  int // идентификатор сообщения, отправляемого AODB cервису
	withDocker bool
}

// состояние FMTP канала, при котором ему отправляем сообщений от AODB
var chValidSt = fmtp.DataReady
var chValidStStr = chValidSt.ToString()

// NewChiefChannelServer конструктор
func NewChiefChannelServer(workWithDocker bool) *ChiefChannelServer {
	return &ChiefChannelServer{
		ChannelSettsChan: make(chan channel_settings.ChannelSettingsWithPort, 10),
		channelSetts: channel_settings.ChannelSettingsWithPort{
			ChSettings: make([]channel_settings.ChannelSettings, 0),
			ChPort:     0,
		},
		ChannelStates:        make(chan []channel_state.ChannelState, 10),
		IncomeAodbPacketChan: make(chan []byte, 1024),
		OutAodbPacketChan:    make(chan []byte, 1024),
		LogChan:              make(chan common.LogMessage, 100),
		killerChan:           make(chan struct{}),
		ChannelBinMap:        make(map[int]сhannelBin),
		//chiefChannelWS:       chief_channel.NewChiefChannelServer(),
		//ws:        web_sock.NewWebSockServer(),
		wsClients:  make(map[int]*websocket.Conn),
		chStates:   make(map[int]сhannelStateTime),
		aodbIdent:  1,
		withDocker: workWithDocker,
	}
}

// Work реализация работы
func (cc *ChiefChannelServer) Work() {
	//go cc.ws.Work(utils.ChannelURLPath)
	//cc.chiefChannelWS.SettChan <- chief_channel.ServerSettings{ChiefPort: cc.ChannelPort}

	for {
		select {
		// получены новые настройки каналов
		case newSetts := <-cc.ChannelSettsChan:

			var needToStopIds, needToStartIds []int // идентификаторы каналов, которые необходимо остановить/запустить

			if cc.channelSetts.ChPort != newSetts.ChPort {
				//cc.ws.SettingsChan <- web_sock.WebSockServerSettings{LocalPort: newSetts.ChPort}

				// обнуляем состояния каналов
				for key := range cc.chStates {
					delete(cc.chStates, key)
				}

				//останавливаем и запускаем все каналы
				for _, val := range cc.channelSetts.ChSettings {
					if val.IsWorking {
						needToStopIds = append(needToStopIds, val.Id)
					}
				}

				//останавливаем и запускаем все каналы
				for _, val := range newSetts.ChSettings {
					if val.IsWorking {
						needToStartIds = append(needToStartIds, val.Id)
					}
				}
			} else if !reflect.DeepEqual(cc.channelSetts.ChSettings, newSetts.ChSettings) {
				// обнуляем состояния каналов
				for key := range cc.chStates {
					delete(cc.chStates, key)
				}

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
						}
					} else { // есть в новых, нет в старых
						if newIt.IsWorking {
							needToStartIds = append(needToStartIds, newIt.Id)
						}
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

			for _, val := range cc.channelSetts.ChSettings {
				var chWork string
				if val.IsWorking {
					chWork = channel_state.ChannelStateError
				} else {
					chWork = channel_state.ChannelStateStopped
				}
				cc.chStates[val.Id] = сhannelStateTime{ChannelState: channel_state.ChannelState{
					ChannelID:   val.Id,
					LocalName:   val.LocalATC,
					RemoteName:  val.RemoteATC,
					DaemonState: chWork,
					FmtpState:   "disabled",
					ChannelURL:  fmt.Sprintf("http://%s:%d/%s", val.URLAddress, val.URLPort, val.URLPath),
				},
					Time: time.Now()}
			}

			// останавливаем каналы FMTP
			if len(needToStopIds) > 0 {
				cc.stopChannelsByIDs(needToStopIds)
			}
			// запускаем каналы FMTP
			if len(needToStartIds) > 0 {
				cc.startChannelsByIDs(needToStartIds)
			}

		// получен новый пакет от провайдера
		case incomeData := <-cc.IncomeAodbPacketChan:
			var fdpsHdr fdps.FdpsHeader
			var unmErr error
			unmErr = json.Unmarshal(incomeData, &fdpsHdr)
			if unmErr == nil {
				switch fdpsHdr.MsgHeader {

				case fdps.FdpsDataText:
					var dataMsg fdps.FdpsAodbPackage
					unmErr = json.Unmarshal(incomeData, &dataMsg)
					if unmErr == nil {
						cc.LogChan <- common.LogCntrlSTDT(common.SeverityInfo, fmtp.Operational.ToString(), common.DirectionIncoming,
							fmt.Sprintf("Получено сообщение от плановой подсистемы(%s) id: <%s>, Лок. ATC: <%s>, Удал. ATC: <%s>, Текст: <%s>.",
								fdps.FdpsAodbService, dataMsg.Ident, dataMsg.LocalAtc, dataMsg.RemoteAtc, dataMsg.Text))
						cc.ProcessAodbPacket(dataMsg)
					} else {
						cc.LogChan <- common.LogCntrlST(common.SeverityError,
							fmt.Sprintf("Получено сообщение от плановой подсистемы(%s) неверного формата, Текст: <%s>. Ошибка: <%s>.",
								fdps.FdpsAodbService, string(incomeData), unmErr.Error()))
					}

				case fdps.FdpsAcknowledge:
					var accMsg fdps.FdpsAodbAcknowledge
					unmErr = json.Unmarshal(incomeData, &accMsg)
					if unmErr == nil {
						cc.LogChan <- common.LogCntrlSTDT(common.SeverityInfo, fmtp.Operational.ToString(), common.DirectionIncoming,
							fmt.Sprintf("Получено подтверждение от плановой подсистемы(%s) id: <%s>.", fdps.FdpsAodbService, accMsg.Ident))
					} else {
						cc.LogChan <- common.LogCntrlST(common.SeverityError,
							fmt.Sprintf("Получено сообщение от плановой подсистемы(%s) неверного формата, Текст: <%s>. Ошибка: <%s>.",
								fdps.FdpsAodbService, string(incomeData), unmErr.Error()))
					}
				default:
					cc.LogChan <- common.LogCntrlST(common.SeverityError,
						fmt.Sprintf("Получено сообщение от плановой подсистемы(%s) неизвестного формата, Текст: <%s>.",
							fdps.FdpsAodbService, string(incomeData)))
				}

			} else {
				cc.LogChan <- common.LogCntrlST(common.SeverityError,
					fmt.Sprintf("Получено сообщение от плановой подсистемы(%s) неверного формата, Текст: <%s>. Ошибка: <%s>.",
						fdps.FdpsAodbService, string(incomeData), unmErr.Error()))

			}

			// получены данные от WS сервера
			// case curWsPkg := <-cc.ws.ReceiveDataChan:
			// 	var curHdr HeaderMsg
			// 	var unmErr error
			// 	if unmErr = json.Unmarshal(curWsPkg.Data, &curHdr); unmErr == nil {
			// 		switch curHdr.Header {

			// 		case RequestSettingsHeader:
			// 			var reqSettsMsg SettingsRequestMsg
			// 			if err := json.Unmarshal(curWsPkg.Data, &reqSettsMsg); err == nil {
			// 				cc.wsClients[reqSettsMsg.ChannelID] = curWsPkg.Sock

			// 				var channelSetts channel_settings.ChannelSettings
			// 			SETTSL:
			// 				for _, curSetts := range cc.channelSetts.ChSettings {
			// 					if curSetts.Id == reqSettsMsg.ChannelID {
			// 						channelSetts = curSetts
			// 						break SETTSL
			// 					}
			// 				}

			// 				if settsData, err := json.Marshal(CreateSettingsAnswerMsg(channelSetts)); err == nil {
			// 					cc.ws.SendDataChan <- web_sock.WsPackage{Data: settsData, Sock: cc.wsClients[channelSetts.Id]}
			// 				}
			// 			}

			// 		case ChannelHeartbeatHeader: //fmt.Println("!ChannelHeartbeatHeader")
			// 			var curHbtMsg ChannelHeartbeatMsg
			// 			if err := json.Unmarshal(curWsPkg.Data, &curHbtMsg); err == nil {
			// 				cc.chStates[curHbtMsg.ChannelID] = сhannelStateTime{ChannelState: curHbtMsg.ChannelState, Time: time.Now()}
			// 			}

			// 			// отправляем heartbeat контроллеру
			// 			var chStateSlice []channel_state.ChannelState
			// 			for key := range cc.chStates {
			// 				chStateSlice = append(chStateSlice, cc.chStates[key].ChannelState)
			// 			}
			// 			cc.ChannelStates <- chStateSlice

			// 		case ChannelLogHeader:
			// 			var curLogMsg ChannelLogMsg
			// 			if err := json.Unmarshal(curWsPkg.Data, &curLogMsg); err == nil {
			// 				cc.LogChan <- curLogMsg.LogMessage
			// 			}

			// 		case ChannelMessageHeader:

			// 			var dataMsg DataMsg
			// 			if err := json.Unmarshal(curWsPkg.Data, &dataMsg); err == nil {
			// 				var localAtc, remoteAtc string
			// 				for _, val := range cc.channelSetts.ChSettings {
			// 					if val.Id == dataMsg.ChannelID {
			// 						localAtc = val.LocalATC
			// 						remoteAtc = val.RemoteATC
			// 						break
			// 					}
			// 				}

			// 				var aodbDataMsg fdps.FdpsAodbPackage
			// 				aodbDataMsg.MsgHeader = fdps.FdpsDataText
			// 				aodbDataMsg.Ident = strconv.Itoa(cc.aodbIdent)
			// 				aodbDataMsg.LocalAtc = localAtc
			// 				aodbDataMsg.RemoteAtc = remoteAtc
			// 				aodbDataMsg.Text = dataMsg.Text

			// 				cc.aodbIdent++
			// 				if cc.aodbIdent > 100000 {
			// 					cc.aodbIdent = 1
			// 				}

			// 				if dataToSend, mrshErr := json.Marshal(aodbDataMsg); mrshErr == nil {
			// 					cc.OutAodbPacketChan <- dataToSend

			// 					cc.LogChan <- common.LogCntrlSTDT(common.SeverityInfo, fmtp.Operational.ToString(), common.DirectionIncoming,
			// 						fmt.Sprintf("Отправлено сообщение плановой подсистеме(%s) id: <%d>, Лок. ATC: <%s>, Удал. ATC: <%s>, Текст: <%s>.",
			// 							fdps.FdpsAodbService, dataMsg.ChannelID, localAtc, remoteAtc, dataMsg.Text))
			// 				}
			// 			}
			// 		}

			// 	} else {
			// 		cc.LogChan <- common.LogCntrlST(common.SeverityError,
			// 			fmt.Sprintf("Ошибка разбора сообщения от приложения FMTP канала. Ошибка: <%s>.", unmErr))
			// 	}

			// получена ошибка от ws сервера
			// case wsErr := <-cc.ws.ErrorChan:
			// 	var curLogMsg common.LogMessage
			// 	if wsErr == nil {
			// 		web.SetWSChannelState(true)
			// 		curLogMsg = common.LogCntrlST(common.SeverityInfo, "Запускаем WS сервер для взаимодействия с FMTP каналами.")
			// 	} else {
			// 		web.SetWSChannelState(false)
			// 		curLogMsg = common.LogCntrlST(common.SeverityError,
			// 			fmt.Sprintf("Возникла ошибка при работе WS сервера взаимодействия с FMTP каналами. Ошибка: <%s>.", wsErr.Error()))
			// 	}
			// 	cc.LogChan <- curLogMsg

			// 	// получено уведомление от ws сервера
			// case wsInfo := <-cc.ws.InfoChan:
			// 	cc.LogChan <- common.LogCntrlST(common.SeverityInfo, "Сервер WS для взаимодействия с FMTP каналами. "+wsInfo)
		}
	}
}

// ProcessAodbPacket обработка пакета AODB
func (cc *ChiefChannelServer) ProcessAodbPacket(pkg fdps.FdpsAodbPackage) {
	var channelID int = -1

	for _, val := range cc.channelSetts.ChSettings {

		if val.DataType == fdps.AODBProvider && val.LocalATC == pkg.LocalAtc && val.RemoteATC == pkg.RemoteAtc {
			channelID = val.Id
			break
		}
	}
	accMsg := fdps.FdpsAodbAcknowledge{FdpsHeader: fdps.FdpsHeader{MsgHeader: fdps.FdpsAcknowledge}, Ident: pkg.Ident}
	if channelID != -1 {
		// if sock, ok := cc.wsClients[channelID]; ok {
		// 	if aodbData, mrshErr := json.Marshal(CreateChiefDataMsg(channelID, pkg.Text)); mrshErr != nil {
		if _, ok := cc.wsClients[channelID]; ok {
			if _, mrshErr := json.Marshal(CreateChiefDataMsg(channelID, pkg.Text)); mrshErr != nil {
				cc.LogChan <- common.LogCntrlST(common.SeverityError,
					fmt.Sprintf("Ошибка формирования сообщения для FMTP канала. Ошибка: <%s>.", mrshErr))
			} else {
				if cc.chStates[channelID].ChannelState.FmtpState == chValidStStr {
					//cc.ws.SendDataChan <- web_sock.WsPackage{Data: aodbData, Sock: sock}
					accMsg.State = fdps.FdpsAckOkText
				} else {
					accMsg.State = fdps.FdpsAckFailedText
				}
			}
		} else {
			accMsg.State = fdps.FdpsAckStoppedText
		}
	} else {
		accMsg.State = fdps.FdpsAckMissedText
	}

	// отправка подтверждения
	if dataToSend, mrshErr := json.Marshal(accMsg); mrshErr != nil {
		cc.LogChan <- common.LogCntrlST(common.SeverityError, fmt.Sprintf("Ошибка формирования сообщения подтверждения. Ошибка: <%s>.", mrshErr))
	} else {
		cc.OutAodbPacketChan <- dataToSend
	}

	cc.LogChan <- common.LogCntrlSTDT(common.SeverityInfo, common.NoneFmtpType, common.DirectionOutcoming,
		fmt.Sprintf("Плановой подсистеме(%s) отправлено подтверждение получения сообщения. Текст: <%+v>.",
			fdps.AODBProvider, accMsg))
}

// останавливаем каналы с указанным ID
func (cc *ChiefChannelServer) stopChannelsByIDs(idsToStop []int) {
STOPL:
	for _, stopID := range idsToStop {
		if channelBin, ok := cc.ChannelBinMap[stopID]; ok {
			channelBin.killChan <- struct{}{}
			<-cc.killerChan

			if !cc.withDocker {
				if err := utils.RemoveChannelBinary(channelBin.filePath); err != nil {
					cc.LogChan <- common.LogCntrlST(common.SeverityError, err.Error())
					continue STOPL
				}
			}
			close(channelBin.killChan)
			delete(cc.ChannelBinMap, stopID)
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
					cc.ChannelBinMap[startID] = сhannelBin{killChan: make(chan struct{})}
					go cc.startChannelContainer(newIt, cc.ChannelBinMap[startID].killChan)

				} else {
					channelFilePath, err := utils.CopyChannelBinary(newIt.Version, newIt.Id, newIt.LocalATC, newIt.RemoteATC)
					if err != nil {
						cc.LogChan <- common.LogCntrlST(common.SeverityError, err.Error())
						continue STARTL
					}
					cc.ChannelBinMap[startID] = сhannelBin{filePath: channelFilePath, killChan: make(chan struct{})}
					go cc.startChannelProcess(channelFilePath, newIt, cc.ChannelBinMap[startID])
				}
				break STARTL2
			}
		}
	}
}

// запуск исполняемого файла fmtp канала
func (cc *ChiefChannelServer) startChannelProcess(channelFilePath string, chSett channel_settings.ChannelSettings, binInfo сhannelBin) {
	cmd := exec.Command(channelFilePath, strconv.Itoa(cc.channelSetts.ChPort), strconv.Itoa(chSett.Id), chSett.LocalATC,
		chSett.RemoteATC, chSett.DataType, chSett.URLPath, strconv.Itoa(chSett.URLPort))

	if err := cmd.Start(); err != nil {
		cc.LogChan <- common.LogCntrlST(common.SeverityError,
			fmt.Sprintf("Ошибка запуска приложения FMTP канала. Исполняемый файл: <%s>. Иденификатор канала: <%d>. Ошибка: <%s>.",
				channelFilePath, chSett.Id, err.Error()))
		return
	}

	// канал, в который ошибка завершения отправкится
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			cc.LogChan <- common.LogCntrlST(common.SeverityError,
				fmt.Sprintf("Нештатное завершение приложения FMTP канала. Исполняемый файл: <%s>. Идентификатор канала: <%d>. Ошибка: <%s>.",
					channelFilePath, chSett.Id, err.Error()))
			if _, ok := cc.chStates[chSett.Id]; ok {
				curState := cc.chStates[chSett.Id]
				curState.ChannelState.DaemonState = channel_state.ChannelStateError
				cc.chStates[chSett.Id] = curState
			}
		}
		return

	case <-binInfo.killChan:
		if err := cmd.Process.Kill(); err != nil {
			cc.LogChan <- common.LogCntrlST(common.SeverityError,
				fmt.Sprintf("Ошибка завершения выполнения приложения FMTP канала. Ошибка: ,<%s>.", err.Error()))
		} else {
			cc.LogChan <- common.LogCntrlST(common.SeverityInfo,
				fmt.Sprintf("Штатное завершение приложения FMTP канала. Исполняемый файл: <%s>. Идентификатор канала: <%d>.",
					channelFilePath, chSett.Id))
		}
		time.Sleep(1 * time.Second)
		cc.killerChan <- struct{}{}
		return
	}
}

// запуск docker контейнера fmtp канала
func (cc *ChiefChannelServer) startChannelContainer(chSett channel_settings.ChannelSettings, killChan chan struct{}) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	defer cli.Close()
	if err != nil {
		cc.LogChan <- common.LogCntrlST(common.SeverityError,
			fmt.Sprintf("Ошибка создания клиента сервиса docker. "+err.Error()))
	}
	cli.NegotiateAPIVersion(ctx)

	portBinding := make(nat.PortMap)
	portSet := make(nat.PortSet)

	if chSett.LocalPort != 0 {
		hostBinding := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(chSett.LocalPort),
		}
		containerPort, bindErr := nat.NewPort("tcp", strconv.Itoa(chSett.LocalPort))
		if bindErr != nil {
			cc.LogChan <- common.LogCntrlST(common.SeverityError, fmt.Sprintf("Ошибка открытия порта docker контейнера. "+bindErr.Error()))
			fmt.Println(" Ошибка открытия порта docker контейнера. " + bindErr.Error())
		}
		portBinding[containerPort] = []nat.PortBinding{hostBinding}
		portSet[containerPort] = struct{}{}
	}
	// exposePorts := make(nat.PortMap)
	// if chSett.LocalPort != 0 {
	// 	//binds := make([]nat.PortBinding, 1)
	// 	binds := []nat.PortBinding{nat.PortBinding{HostIP: "127.0.0.1", HostPort: strconv.Itoa(chSett.LocalPort)}}
	// 	exposePorts[nat.Port(strconv.Itoa(chSett.LocalPort))] = binds
	// }
	curContainerName := "fmtp_channel_" + chSett.LocalATC + "_" + chSett.RemoteATC + "_" + strconv.Itoa(chSett.Id)

	resp, crErr := cli.ContainerCreate(ctx,
		&container.Config{
			ExposedPorts: portSet,
			Image:        chief_configurator.ChiefCfg.DockerRegistry + "/" + chief_configurator.ChannelImageName + ":" + chSett.Version,
			Cmd: []string{strconv.Itoa(cc.channelSetts.ChPort), strconv.Itoa(chSett.Id), chSett.LocalATC,
				chSett.RemoteATC, chSett.DataType, chSett.URLPath, strconv.Itoa(chSett.URLPort)},
		},
		&container.HostConfig{
			Resources: container.Resources{
				CPUCount:   1,
				CPUPercent: 10,
				Memory:     100 * 1 << 20, // 100 MB
			},
			NetworkMode: "host",
			//PortBindings: portBinding,
			RestartPolicy: container.RestartPolicy{Name: "always"},
		},
		&network.NetworkingConfig{},
		curContainerName)

	if crErr != nil {
		cc.LogChan <- common.LogCntrlST(common.SeverityError, fmt.Sprintf("Ошибка создания docker контейнера. "+crErr.Error()))
	} else {
		cc.LogChan <- common.LogCntrlST(common.SeverityInfo, fmt.Sprintf(" Создан docker контейнер "+curContainerName+"."))
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		cc.LogChan <- common.LogCntrlST(common.SeverityError, fmt.Sprintf("Ошибка запуска docker контейнера. "+err.Error()))
	} else {
		cc.LogChan <- common.LogCntrlST(common.SeverityInfo, fmt.Sprintf(" Запущен docker контейнер "+curContainerName+"."))
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	for {
		select {
		case cntErr := <-errCh:
			if cntErr != nil {
				cc.LogChan <- common.LogCntrlST(common.SeverityInfo,
					fmt.Sprintf("Ошибка в работе docker контейнера %s. Текст ошибки: %s.", curContainerName, cntErr.Error()))
			}
		case <-statusCh:

		case <-killChan:
			var stopDur time.Duration = 100 * time.Millisecond

			if stopErr := cli.ContainerStop(ctx, resp.ID, &stopDur); stopErr == nil {
				cc.LogChan <- common.LogCntrlST(common.SeverityInfo, fmt.Sprintf("Остановлен docker контейнер "+curContainerName+"."))
				if rmErr := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{}); rmErr == nil {
					cc.LogChan <- common.LogCntrlST(common.SeverityInfo, fmt.Sprintf("Удален docker контейнер "+curContainerName+"."))
				} else {
					cc.LogChan <- common.LogCntrlST(common.SeverityError, fmt.Sprintf("Ошибка удаления docker контейнера. "+rmErr.Error()))
				}
			} else {
				cc.LogChan <- common.LogCntrlST(common.SeverityError, fmt.Sprintf("Ошибка остановки docker контейнера. "+stopErr.Error()))
			}
			cc.killerChan <- struct{}{}
			return
		}
	}
}
