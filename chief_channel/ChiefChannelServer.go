package chief_channel

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief_configurator"
	"fdps/fmtp/fmtp"
	"fdps/utils"
	"fdps/utils/logger"
	"fdps/utils/web_sock"
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
	ChannelStates    chan []channel_state.ChannelState             // канал для передачи состояний каналов
	statesSendTicker *time.Ticker                                  // тикер отправки состояния каналов

	IncomeAodbPacketChan chan []byte // канал для приема сообщений от провайдера AODB
	OutAodbPacketChan    chan []byte // канал для отправки сообщений провайдеру AODB

	IncomeOldiPacketChan chan []byte // канал для приема сообщений от провайдера OLDI
	OutOldiPacketChan    chan []byte // канал для отправки сообщений провайдеру OLDI

	ChannelBinMap *sync.Map // ключ - идентификатор каналаб значение типа сhannelBin

	killerChan chan struct{} // канал, по которому передается сигнал о завершение работы канала

	wsServer *web_sock.WebSockServer

	wsClients map[int]*websocket.Conn

	chStates map[int]сhannelStateTime // ключ - ID канала

	aodbIdent    int // идентификатор сообщения, отправляемого AODB cервису
	oldiIdent    int // идентификатор сообщения, отправляемого OLDI cервису
	withDocker   bool
	dataFromOldi []byte // буфер полученных данных OLDI

	LogChan chan common.LogMessage // канал для передачи сообщений от каналов
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
		ChannelStates:        make(chan []channel_state.ChannelState, 10),
		statesSendTicker:     time.NewTicker(time.Second),
		IncomeAodbPacketChan: make(chan []byte, 1024),
		OutAodbPacketChan:    make(chan []byte, 1024),
		IncomeOldiPacketChan: make(chan []byte, 1024),
		OutOldiPacketChan:    make(chan []byte, 1024),
		killerChan:           make(chan struct{}),
		ChannelBinMap:        new(sync.Map),
		wsServer:             web_sock.NewWebSockServer(done),
		wsClients:            make(map[int]*websocket.Conn),
		chStates:             make(map[int]сhannelStateTime),
		aodbIdent:            1,
		oldiIdent:            1,
		withDocker:           workWithDocker,
		LogChan:              make(chan common.LogMessage, 10),
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
					if needStartChannel := cc.initChannelState(val); needStartChannel {
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
							_ = cc.initChannelState(newIt)
						}
					} else { // есть в новых, нет в старых
						if newIt.IsWorking {
							needToStartIds = append(needToStartIds, newIt.Id)
							_ = cc.initChannelState(newIt)
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

			// обнуляем состояния каналов
			// for key := range cc.chStates {
			// 	delete(cc.chStates, key)
			// }

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
						cc.LogChan <- common.LogCntrlSDT(common.SeverityDebug, fdps.AODBProvider,
							fmt.Sprintf("Получено сообщение от плановой подсистемы: %s.", dataMsg.Text))
						cc.ProcessAodbPacket(dataMsg)
					} else {
						logger.PrintfErr("Получено сообщение от плановой подсистемы(AODB) неверного формата, Текст: %s. Ошибка: %s.",
							string(incomeData), unmErr.Error())
					}

				case fdps.FdpsAcknowledge:
					var accMsg fdps.FdpsAodbAcknowledge
					unmErr = json.Unmarshal(incomeData, &accMsg)
					if unmErr == nil {
						cc.LogChan <- common.LogCntrlSDT(common.SeverityDebug, fdps.AODBProvider,
							fmt.Sprintf("Получено подтверждение от плановой подсистемы: %s.", accMsg.Ident))
					} else {
						logger.PrintfErr("Получено сообщение от плановой подсистемы(AODB) неверного формата, Текст: %s. Ошибка: %s.",
							string(incomeData), unmErr.Error())
					}
				default:
					logger.PrintfErr("Получено сообщение от плановой подсистемы(AODB) неизвестного формата, Текст: %s.",
						string(incomeData))
				}

			} else {
				logger.PrintfErr("Получено сообщение от плановой подсистемы(AODB) неверного формата, Текст: %s. Ошибка: %s.",
					string(incomeData), unmErr.Error())
			}

		// получен новый пакет от провайдера OLDI
		case incomeOldiData := <-cc.IncomeOldiPacketChan:
			cc.dataFromOldi = append(cc.dataFromOldi, incomeOldiData...)

			// вырезаем все подтверждения <acc>
			startAccIdx := strings.Index(string(cc.dataFromOldi), startOldiAccTag)
			endAccIdx := strings.Index(string(cc.dataFromOldi), endOldiAccTag)
			hasAcc := (startAccIdx != -1 && endAccIdx != -1 && startAccIdx < endAccIdx)

			for {
				if hasAcc {
					curAccBytes := make([]byte, 2*(endAccIdx+len(endOldiAccTag)-startAccIdx))
					copy(curAccBytes, cc.dataFromOldi[startAccIdx:endAccIdx+len(endOldiAccTag)])

					var oldiAcc fdps.FdpsOldiAcknowledge
					if unmAccErr := xml.Unmarshal(curAccBytes, &oldiAcc); unmAccErr == nil {

						cc.LogChan <- common.LogCntrlSDT(common.SeverityInfo, fdps.OLDIProvider,
							fmt.Sprintf("Получено подтверждение от плановой подсистемы: %s.", oldiAcc.ToString()))
					}

					cc.dataFromOldi = append(cc.dataFromOldi[:startAccIdx], cc.dataFromOldi[endAccIdx+len(endOldiAccTag):]...)

					startAccIdx = strings.Index(string(cc.dataFromOldi), startOldiAccTag)
					endAccIdx = strings.Index(string(cc.dataFromOldi), endOldiAccTag)
					hasAcc = (startAccIdx != -1 && startAccIdx < endAccIdx)
				} else {
					break
				}

			}

			startMsgIdx := strings.Index(string(cc.dataFromOldi), startOldiMsgTag)
			endMsgIdx := strings.Index(string(cc.dataFromOldi), endOldiMsgTag)
			hasMsg := (startMsgIdx != -1 && endMsgIdx != -1 && startMsgIdx < endMsgIdx)

			for {
				if hasMsg {

					curMsgBytes := make([]byte, endMsgIdx+len(endOldiMsgTag)-startMsgIdx)
					copy(curMsgBytes, cc.dataFromOldi[startMsgIdx:endMsgIdx+len(endOldiMsgTag)])

					var oldiPkg fdps.FdpsOldiPackage

					if unmPkgErr := xml.Unmarshal(curMsgBytes, &oldiPkg); unmPkgErr == nil {

						cc.LogChan <- common.LogCntrlSDT(common.SeverityInfo, fdps.OLDIProvider,
							fmt.Sprintf("Получено сообщение от плановой подсистемы: %s.", oldiPkg.Text))

						cc.ProcessOldiPacket(oldiPkg)
					}

					cc.dataFromOldi = append(cc.dataFromOldi[:startMsgIdx], cc.dataFromOldi[endMsgIdx+len(endOldiMsgTag)-startMsgIdx:]...)

					startMsgIdx = strings.Index(string(cc.dataFromOldi), startOldiMsgTag)
					endMsgIdx = strings.Index(string(cc.dataFromOldi), endOldiMsgTag)
					hasMsg = (startMsgIdx != -1 && endMsgIdx != -1 && startMsgIdx < endMsgIdx)

				} else {
					break
				}
			}

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
						cc.LogChan <- curLogMsg.LogMessage
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

						if channelType == fdps.AODBProvider {
							var aodbDataMsg fdps.FdpsAodbPackage
							aodbDataMsg.MsgHeader = fdps.FdpsDataText
							aodbDataMsg.Ident = strconv.Itoa(cc.aodbIdent)
							aodbDataMsg.LocalAtc = localAtc
							aodbDataMsg.RemoteAtc = remoteAtc
							aodbDataMsg.Text = dataMsg.Text

							cc.aodbIdent++
							if cc.aodbIdent > 100000 {
								cc.aodbIdent = 1
							}

							if dataToSend, mrshErr := json.Marshal(aodbDataMsg); mrshErr == nil {
								cc.LogChan <- common.LogCntrlSDT(common.SeverityDebug, fdps.AODBProvider,
									fmt.Sprintf("Плановой подсистеме отправлено сообщение: %s.", dataMsg.Text))

								cc.OutAodbPacketChan <- dataToSend
							}
						} else if channelType == fdps.OLDIProvider {
							var oldiDataMsg fdps.FdpsOldiPackage
							oldiDataMsg.Id = cc.oldiIdent
							oldiDataMsg.LocalAtc = localAtc
							oldiDataMsg.RemoteAtc = remoteAtc
							oldiDataMsg.Text = dataMsg.Text

							cc.oldiIdent++
							if cc.oldiIdent > 1000 {
								cc.oldiIdent = 1
							}

							if dataToSend, mrshErr := xml.Marshal(oldiDataMsg); mrshErr == nil {
								cc.LogChan <- common.LogCntrlSDT(common.SeverityInfo, fdps.OLDIProvider,
									fmt.Sprintf("Плановой подсистеме отправлено сообщение: %s.", dataMsg.Text))

								cc.OutOldiPacketChan <- dataToSend
							}
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

			// если от канала нет сообщений состояния и он должен работать, то возможно он свалился
			// плпытаемся запустить канал снова
			var needToRestartIds []int // идентификаторы каналов, которые необходимо запустить

			// удаляем из мэпки состояний старые состояния
			for channelId, val := range cc.chStates {
				if val.Time.Add(10 * channel_state.StateSendInterval).Before(time.Now()) {
					delete(cc.chStates, channelId)
					logger.PrintfErr("delete(cc.chStates, key) %v", channelId)

					// если состояния удалили и в настройках канал должен быть запущен, то добавляем состояние
					for _, val := range cc.channelSetts.ChSettings {
						if _, ok := cc.chStates[val.Id]; !ok {

							if needRestartChannel := cc.initChannelState(val); needRestartChannel {
								needToRestartIds = append(needToRestartIds, val.Id)
							}
							break
						}
					}
				}
			}

			if len(needToRestartIds) > 0 {
				logger.PrintfWarn("needToRestartIds %v", needToRestartIds)

				cc.stopChannelsByIDs(needToRestartIds)

				// костыль - не успевает удалиться старый контейнер, при создании нового - конфликн имен
				time.AfterFunc(2*time.Second, func() {
					cc.startChannelsByIDs(needToRestartIds)
				})
			}

			// отправляем heartbeat контроллеру
			var chStateSlice []channel_state.ChannelState
			for key := range cc.chStates {
				chStateSlice = append(chStateSlice, cc.chStates[key].ChannelState)
			}
			cc.ChannelStates <- chStateSlice
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
		if sock, ok := cc.wsClients[channelID]; ok {
			if aodbData, mrshErr := json.Marshal(CreateChiefDataMsg(channelID, pkg.Text)); mrshErr != nil {
				logger.PrintfErr("Ошибка формирования сообщения для FMTP канала. Ошибка: %v.", mrshErr)
			} else {
				if cc.chStates[channelID].ChannelState.FmtpState == chValidStStr {
					cc.wsServer.SendDataChan <- web_sock.WsPackage{Data: aodbData, Sock: sock}
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
		logger.PrintfErr("Ошибка формирования сообщения подтверждения. Ошибка: %v.", mrshErr)
	} else {
		cc.OutAodbPacketChan <- dataToSend
	}

	cc.LogChan <- common.LogCntrlSDT(common.SeverityDebug, fdps.AODBProvider,
		fmt.Sprintf("Плановой подсистеме отправлено подтверждение: %+v.", accMsg))
}

// ProcessOldiPacket обработка пакета OLDI
func (cc *ChiefChannelServer) ProcessOldiPacket(pkg fdps.FdpsOldiPackage) {
	var channelID int = -1

	for _, val := range cc.channelSetts.ChSettings {

		// от OLDI только cid приходит
		if pkg.LocalAtc == "" {
			if val.DataType == fdps.OLDIProvider && val.RemoteATC == pkg.RemoteAtc {
				channelID = val.Id
				break
			}
		} else { // данные от провайдера тренажера (все RemoteAtc одинаковые, LocalAtc не пустой)
			if val.DataType == fdps.OLDIProvider && val.LocalATC == pkg.LocalAtc && val.RemoteATC == pkg.RemoteAtc {
				channelID = val.Id
				break
			}
		}
	}
	accMsg := fdps.FdpsOldiAcknowledge{Id: pkg.Id}
	if channelID != -1 {
		if sock, ok := cc.wsClients[channelID]; ok {
			if oldiData, mrshErr := json.Marshal(CreateChiefDataMsg(channelID, pkg.Text)); mrshErr != nil {
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

	// отправка подтверждения
	cc.OutOldiPacketChan <- accMsg.ToString()

	cc.LogChan <- common.LogCntrlSDT(common.SeverityInfo, fdps.OLDIProvider,
		fmt.Sprintf("Плановой подсистеме отправлено подтверждение: %s.", accMsg.ToString()))
}

// останавливаем каналы с указанным ID
func (cc *ChiefChannelServer) stopChannelsByIDs(idsToStop []int) {
	logger.PrintfErr("stopChannelsByIDs %v", idsToStop)

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
	logger.PrintfErr("startChannelsByIDs %v", idsToStart)

STARTL:
	for _, startID := range idsToStart {
	STARTL2:
		for _, newIt := range cc.channelSetts.ChSettings {
			if startID == newIt.Id {
				if cc.withDocker {
					cc.ChannelBinMap.Store(startID, channelBin{killChan: make(chan struct{})})

					if val, ok := cc.ChannelBinMap.Load(startID); ok {
						go cc.startChannelContainer(newIt, val.(channelBin).killChan)
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

			close(binInfo.killChan)
			cc.ChannelBinMap.Delete(chSett.Id)

			if _, ok := cc.chStates[chSett.Id]; ok {
				curState := cc.chStates[chSett.Id]
				curState.ChannelState.DaemonState = channel_state.ChannelStateError
				cc.chStates[chSett.Id] = curState
			}
		}
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
func (cc *ChiefChannelServer) startChannelContainer(chSett channel_settings.ChannelSettings, killChan chan struct{}) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.PrintfErr("Ошибка создания клиента сервиса docker. Ошибка: %v", err)
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
	} else {
		logger.PrintfDebug("Создан docker контейнер %s. Используемый образ: %s.", curContainerName, imageName)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logger.PrintfErr("Ошибка запуска docker контейнера %s. Ошибка: %v.", curContainerName, err)
	} else {
		logger.PrintfDebug("Запущен docker контейнер %s.", curContainerName)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	for {
		select {
		case cntErr := <-errCh:
			if cntErr != nil {
				logger.PrintfErr("Ошибка в работе docker контейнера %s. Ошибка: %v.", curContainerName, cntErr)
			}

		case <-statusCh:

		case <-killChan:
			logger.PrintfDebug("Команда завершить docker контейнер %s.", curContainerName)

			var stopDur time.Duration = 100 * time.Millisecond

			if stopErr := cli.ContainerStop(ctx, resp.ID, &stopDur); stopErr == nil {
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

func (cc *ChiefChannelServer) initChannelState(chSett channel_settings.ChannelSettings) (needStartChannel bool) {
	var curChannelState string
	if chSett.IsWorking {
		needStartChannel = true
		curChannelState = channel_state.ChannelStateError
	} else {
		needStartChannel = false
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

	return needStartChannel
}
