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
	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/chief_channel"

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

	curLog := common.LogChannelST(severity, text)
	completeLogMessage(&curLog)

	if dataToSend, err := json.Marshal(chief_channel.CreateChannelLogMsg(curLog)); err == nil {
		chiefClient.SendChan <- dataToSend
	}
}

// дополнение сообщения журнала сведениями о канала Id, RemoteAtc, LocalAtc, DataType
func completeLogMessage(logMsg *common.LogMessage) {
	curLog := *logMsg

	curLog.ChannelId = channelSetts.Id
	curLog.ChannelLocName = channelSetts.LocalATC
	curLog.ChannelRemName = channelSetts.RemoteATC
	curLog.DataType = channelSetts.DataType

	switch curLog.Severity {
	case common.SeverityDebug:
		logger.PrintfDebug("FMTP FORMAT %v", curLog)
	case common.SeverityInfo:
		logger.PrintfInfo("FMTP FORMAT %v", curLog)
	case common.SeverityWarning:
		logger.PrintfWarn("FMTP FORMAT %v", curLog)
	case common.SeverityError:
		logger.PrintfErr("FMTP FORMAT %v", curLog)
	default:
		logger.PrintfDebug("FMTP FORMAT %v", curLog)
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
	logger.WebLogSetTableUserFormat(`
		<div style="display: block;  height: 1000px; position: relative; overflow-x: auto;">
		<table width="100%" border="1" cellspacing="0" cellpadding="4" class="table table-bordered table-striped mb-0">
			<colgroup>
				<col span="1" style="width: 10%;">
				<col span="7" style="width: 5%;">
			</colgroup>
			<tr>
				<th>Дата, время</th>
				<th>Источник</th>
				<th>Серъезность</th>
				<th>Лок ATC</th>
				<th>Уд ATC</th>
				<th>Тип</th>
				<th>FMTP тип</th>
				<th>Направление</th>
				<th>Текст</th>
			</tr>
			{{with .Lr}}
				{{range .}}
					<tr align="center" bgcolor="{{.MsgColor}}">
						<td align="left"> {{.DateTime}}	</td>
						<td align="left"> {{.Source}} </td>
						<td align="left"> {{.Severity}}	</td>
						<td align="left"> {{.ChannelLocName}} </td>
						<td align="left"> {{.ChannelRemName}} </td>
						<td align="left"> {{.DataType}} </td>
						<td align="left"> {{.FmtpType}} </td>
						<td align="left"> {{.Direction}} </td>
						<td align="left"> {{.Text}} </td>
					</tr>
				{{end}}
			{{end}}
		</table>
	`)

	webUserFormatFunc := func(severity string, format string, a ...interface{}) interface{} {

		severityToColor := func(sev string) (msgColor string) {
			switch sev {
			case common.SeverityDebug:
				msgColor = logger.DebugColor
			case common.SeverityInfo:
				msgColor = logger.InfoColor
			case common.SeverityWarning:
				msgColor = logger.WarningColor
			case common.SeverityError:
				msgColor = logger.ErrorColor
			default:
				msgColor = logger.DefaultColor
			}
			return msgColor
		}

		var logValue interface{}
		if len(a) > 0 {
			fmtpLogMsg, ok := a[0].(common.LogMessage)
			if ok {
				logValue = common.LogMessageWithColor{
					LogMessage: fmtpLogMsg,
					MsgColor:   severityToColor(severity),
				}
			} else {
				logValue = common.LogMessageWithColor{
					LogMessage: common.CreateControllerMessage(severity, fmt.Sprintf(format, a...)),
					MsgColor:   severityToColor(severity),
				}
			}
		} else {
			logValue = common.LogMessageWithColor{
				LogMessage: common.CreateControllerMessage(severity, fmt.Sprintf(format, a...)),
				MsgColor:   severityToColor(severity),
			}
		}
		return logValue
	}
	logger.WebLogSetUserFormatFunc(webUserFormatFunc)

	/////////////////

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
							createLogMessage(common.SeverityDebug,
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

		// нет подключения к контроллеру в течинии минуты, завершаем приложение
		case _ = <-chiefClient.CloseChan:
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
