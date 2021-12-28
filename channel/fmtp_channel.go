package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/channel/fmtp_states"
	"fdps/fmtp/chief_channel"
	"fdps/fmtp/fmtp_logger"

	"fdps/go_utils/logger"
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

	curLog := fmtp_logger.LogChannelST(severity, text)
	completeLogMessage(&curLog)

	if dataToSend, err := json.Marshal(chief_channel.CreateChannelLogMsg(curLog)); err == nil {
		chiefClient.SendChan <- dataToSend
	}
}

// дополнение сообщения журнала сведениями о канала Id, RemoteAtc, LocalAtc, DataType
func completeLogMessage(logMsg *fmtp_logger.LogMessage) {
	logMsg.ChannelId = channelSetts.Id
	logMsg.ChannelLocName = channelSetts.LocalATC
	logMsg.ChannelRemName = channelSetts.RemoteATC
	logMsg.DataType = channelSetts.DataType

	switch logMsg.Severity {
	case fmtp_logger.SeverityDebug:
		logger.PrintfDebug("FMTP FORMAT %v", *logMsg)
	case fmtp_logger.SeverityInfo:
		logger.PrintfInfo("FMTP FORMAT %v", *logMsg)
	case fmtp_logger.SeverityWarning:
		logger.PrintfWarn("FMTP FORMAT %v", *logMsg)
	case fmtp_logger.SeverityError:
		logger.PrintfErr("FMTP FORMAT %v", *logMsg)
	default:
		logger.PrintfDebug("FMTP FORMAT %v", *logMsg)
	}
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

	logger.InitializeWebLog(
		logger.LogWebSettings{
			StartHttp:  true,
			NetPort:    webPort,
			LogURLPath: webPath,
			Title:      "FMTP Канал",
			ShowSetts:  false,
		})
	logger.AppendLogger(logger.WebLogger)

	// свой формат вывода логов на web страницу
	fmtp_logger.SetUserLogFormatForWeb()

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
							createLogMessage(fmtp_logger.SeverityError,
								fmt.Sprintf("Получены некорректные настройки. Настройки: <%s>. Ошибка: <%s>", channelSetts.ToLogMessage(), checkErr.Error()))

						} else {
							createLogMessage(fmtp_logger.SeverityDebug,
								fmt.Sprintf("Получены настройки. Настройки: <%s>", channelSetts.ToLogMessage()))

							go fmtpStateCntrl.Work(channelSetts)

							logger.SetDebugParam("Локальный - удаленный ATC:", fmt.Sprintf("%s - %s", channelSetts.LocalATC, channelSetts.RemoteATC), channel_state.WebDefaultColor)
							logger.SetDebugParam("Тип данных:", channelSetts.DataType, channel_state.WebDefaultColor)
							logger.SetDebugParam("Кодировка:", channelSetts.DataEncoding, channel_state.WebDefaultColor)

							if channelSetts.NetRole == "server" {
								logger.SetDebugParam("Тип подключения:", "TCP сервер", channel_state.WebDefaultColor)
								logger.SetDebugParam("Локальный порт:", strconv.Itoa(channelSetts.LocalPort), channel_state.WebDefaultColor)
							} else {
								logger.SetDebugParam("Тип подключения:", "TCP клиент", channel_state.WebDefaultColor)
								logger.SetDebugParam("Удаленный IP адрес:", channelSetts.RemoteAddress, channel_state.WebDefaultColor)
								logger.SetDebugParam("Удаленный порт:", strconv.Itoa(channelSetts.RemotePort), channel_state.WebDefaultColor)
							}

							logger.SetVersion(channelSetts.Version)
						}
					} else {
						createLogMessage(fmtp_logger.SeverityError,
							fmt.Sprintf("Получено сообщение неизвестного формата. Сообщение: <%s>. Ошибка: <%s>.", string(curData), err.Error()))
					}
				} else if headerMsg.Header == chief_channel.FdpsMessageHeader {
					var curDataMsg chief_channel.DataMsg

					if err := json.Unmarshal(curData, &curDataMsg); err == nil {
						fmtpStateCntrl.FmtpDataSendChan <- curDataMsg.FmtpMessage
					} else {
						createLogMessage(fmtp_logger.SeverityError,
							fmt.Sprintf("От контроллера получено сообщение неизвестного формата. Сообщение: <%s>. Ошибка: <%s>.",
								string(curData), err.Error()))
					}
				}
			} else {
				createLogMessage(fmtp_logger.SeverityError,
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

		// нет подключения к контроллеру в течинии минуты, завершаем приложение
		case <-chiefClient.CloseChan:
			return chief_channel.FailToConnect

		// получено текущее состояние канала
		case curState := <-fmtpStateCntrl.FmtpStateChan:
			if curState.FmtpState == "data_ready" {
				logger.SetDebugParam("FMTP состояние:", curState.FmtpState, channel_state.WebOkColor)
			} else {
				logger.SetDebugParam("FMTP состояние:", curState.FmtpState, channel_state.WebErrorColor)
			}

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
