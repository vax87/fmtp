package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/channel/fmtp_states"
	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/web"
)

// клиент для связи с контроллером каналов
var chiefClient *chief_channel.Client

// контроллер состояния канала FMTP
var fmtpStateCntrl = fmtp_states.NewStateController()

// порт подключения к контроллеру
var chiefPort int

// path web странички
var webPath string

// порт web странички
var webPort int

// настройки FMTP канала
var channelSetts channel_settings.ChannelSettings

// создание сообщения журнала для отправки контроллеру
func createLogMessage(severity string, text string) {

	curLog := common.LogChannelST(severity, text)
	completeLogMessage(&curLog)

	if dataToSend, err := json.Marshal(chief_channel.CreateChannelLogMsg(curLog)); err == nil {
		chiefClient.SendChan <- dataToSend
	}
}

// дополнение сообщения журнала сведениями о канала Id, RemoteAtc, LocalAtc, DataType
func completeLogMessage(logMsg *common.LogMessage) {

	logMsg.ChannelId = channelSetts.Id
	logMsg.ChannelLocName = channelSetts.LocalATC
	logMsg.ChannelRemName = channelSetts.RemoteATC
	logMsg.DataType = channelSetts.DataType

	curLog := common.LogMessage(*logMsg)
	web.AppendLog(curLog)
}

func main() { os.Exit(mainReturnWithCode()) }

func mainReturnWithCode() int {

	log.Println("ARGS", os.Args)

	if len(os.Args) == 8 {
		var err error
		chiefPort, err = strconv.Atoi(os.Args[1])
		if err != nil {
			log.Println("FATAL. Invalid channels chief port.")
			return chief_channel.InvalidNetPort
		}
		channelSetts.Id, err = strconv.Atoi(os.Args[2])
		if err != nil {
			log.Println("FATAL. Invalid channel ID.")
			return chief_channel.InvalidDaemonID
		}
		channelSetts.LocalATC = os.Args[3]
		channelSetts.RemoteATC = os.Args[4]
		channelSetts.DataType = os.Args[5]
		webPath = os.Args[6]
		webPort, err = strconv.Atoi(os.Args[7])
		if err != nil {
			log.Println("FATAL. Invalid web port.")
			return chief_channel.InvalidWebPort
		}

		done := make(chan struct{})

		// контроллер для работы с контроллером каналов
		chiefClient = chief_channel.NewChiefChannelClient(done)
	} else {
		log.Println("FATAL. Invalid ARG count. Get: " + strconv.Itoa(len(os.Args)) + ". Need 8 (_ ChiefPort, ID, LocATC, RemATC, Type, WebPath, WebPort).")
		return chief_channel.InvalidParamCount
	}

	web.Initialize(webPath, webPort, new(web.ChannelPage))
	go web.Start()

	go chiefClient.Work()
	chiefClient.SettChan <- chief_channel.ClientSettings{ChiefAddress: "127.0.0.1", ChiefPort: chiefPort, ChannelID: channelSetts.Id}

	for {
		select {
		// получены считанные данные от сетевого контроллера
		case curData := <-chiefClient.ReceiveChan:
			var headerMsg chief_channel.HeaderMsg

			if err := json.Unmarshal(curData, &headerMsg); err == nil {
				if headerMsg.Header == chief_channel.AnswerSettingsHeader {
					if err := json.Unmarshal(curData, &channelSetts); err == nil {
						if checkErr := channelSetts.CheckSettings(); checkErr != nil {
							createLogMessage(common.SeverityError,
								fmt.Sprintf("Получены некорректные настройки. Настройки: <%s>. Ошибка: <%s>", channelSetts.ToLogMessage(), checkErr.Error()))

						} else {
							createLogMessage(common.SeverityInfo,
								fmt.Sprintf("Получены настройки. Настройки: <%s>", channelSetts.ToLogMessage()))

							go fmtpStateCntrl.Work(channelSetts)

							web.SetChannelSetts(web.ChannelSettsWeb{
								DataType:      channelSetts.DataType,
								NetRole:       channelSetts.NetRole,
								LocalATC:      channelSetts.LocalATC,
								RemoteATC:     channelSetts.RemoteATC,
								RemoteAddress: channelSetts.RemoteAddress,
								RemotePort:    channelSetts.RemotePort,
								LocalPort:     channelSetts.LocalPort,
								DataEncoding:  channelSetts.DataEncoding,
							})
							web.SetVersion(channelSetts.Version)
						}
					} else {
						createLogMessage(common.SeverityError,
							fmt.Sprintf("Получено сообщение неизвестного формата. Сообщение: <%s>. Ошибка: <%s>.", string(curData), err.Error()))
					}
				} else if headerMsg.Header == chief_channel.FdpsMessageHeader {
					var curDataMsg chief_channel.DataMsg

					if err := json.Unmarshal(curData, &curDataMsg); err == nil {
						fmtpStateCntrl.FmtpDataSendChan <- curDataMsg.FmtpMessage
					} else {
						createLogMessage(common.SeverityError,
							fmt.Sprintf("От контроллера получено сообщение неизвестного формата. Сообщение: <%s>. Ошибка: <%s>.",
								string(curData), err.Error()))
					}
				}
			} else {
				createLogMessage(common.SeverityError,
					fmt.Sprintf("От контроллера получено сообщение неизвестного формата. Сообщение: <%s>. Ошибка: <%s>.",
						string(curData), err.Error()))
				continue
			}

		// получено сообщение для журнала от контроллера связи управляющей службой
		case curLogMsg := <-chiefClient.LogChan:
			completeLogMessage(&curLogMsg)

			if dataToSend, err := json.Marshal(chief_channel.CreateChannelLogMsg(curLogMsg)); err == nil {
				chiefClient.SendChan <- dataToSend
			}

		// получено текущее состояние канала
		case curState := <-fmtpStateCntrl.FmtpStateChan:
			web.SetChannelFMTPState(curState.FmtpState)

			// отправляем heartbeat сообщение контроллеру (chief)
			if dataToSend, err := json.Marshal(chief_channel.CreateChannelHeartbeatMsg(curState)); err == nil {
				chiefClient.SendChan <- dataToSend
			}

		// получено сообщение для журнала от контроллера состояния
		case curLogMsg := <-fmtpStateCntrl.LogMessageChan:
			completeLogMessage(&curLogMsg)
			if dataToSend, err := json.Marshal(chief_channel.CreateChannelLogMsg(curLogMsg)); err == nil {
				chiefClient.SendChan <- dataToSend
			}

		// получено сообщение поверх FMTP от контроллера состояния
		case curDataMessage := <-fmtpStateCntrl.FmtpDataReceiveChan:
			if dataToSend, err := json.Marshal(chief_channel.CreateChannelDataMsg(channelSetts.Id, curDataMessage)); err == nil {
				chiefClient.SendChan <- dataToSend
			}
		}
	}
}
