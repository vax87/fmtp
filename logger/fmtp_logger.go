package main

import (
	"encoding/json"
	"fdps/fmtp/chief_logger"
	"fdps/utils/web_sock"

	"fdps/fmtp/logger/common"
	"fdps/fmtp/logger/file"
	"fdps/fmtp/logger/logger_settings"
	"fdps/fmtp/logger/oracle"
	"fdps/fmtp/web"
	"fmt"
	"log"
)

// контроллер записи в файлы
var fileLogCntrl = &file.FileLoggerController{MessChan: make(chan common.LogMessage, 1024),
	SettingsChan: make(chan file.FileLoggerSettings)}

// контроллер записи в БД oracle
var oracleLogCntrl = oracle.NewOracleController()

var done = make(chan struct{})

// контроллер для работы с сетью
var netController = web_sock.NewWebSockServer(done)

func settingsRoutine() {
	var setCntrl = &logger_settings.SettingsController{
		NetSettingsChan:          make(chan web_sock.WebSockServerSettings),
		FileLoggerSettingsChan:   make(chan file.FileLoggerSettings),
		OracleLoggerSettingsChan: make(chan oracle.OracleLoggerSettings),
		DoneChan:                 make(chan error)}
	go setCntrl.Load()

	for {
		select {
		case newFileLoggerSettings := <-setCntrl.FileLoggerSettingsChan:
			fileLogCntrl.SettingsChan <- newFileLoggerSettings
		case newOracleLoggerSettings := <-setCntrl.OracleLoggerSettingsChan:
			oracleLogCntrl.SettingsChan <- newOracleLoggerSettings
		case newNetworkSettings := <-setCntrl.NetSettingsChan:
			netController.SettingsChan <- newNetworkSettings
		case settsError := <-setCntrl.DoneChan:
			if settsError != nil {
				log.Println("Error when read settings from file", settsError)
			}
			return
		}
	}
}

func main() {
	web.Initialize("/log", 13001, new(web.LoggerPage))
	go web.Start()

	// подпрограмма для контроллера сети
	go netController.Work("/logger")

	// подпрограмма для контроллера логов(запись в файлы)
	go fileLogCntrl.Run()

	// подпрограмма для контроллера логов(запись в БД oracle)
	go oracleLogCntrl.Run()

	// инициализация контроллеров настройками
	go settingsRoutine()

	for {
		select {
		// получено сообщение о состоянии контроллера записи в БД oracle
		case curState := <-oracleLogCntrl.StateChan:
			stateData, stateErr := json.Marshal(curState)
			if stateErr != nil {
				log.Println("State message marshal error. ", stateErr)
			} else {
				netController.SendDataChan <- web_sock.WsPackage{Data: stateData}
				web.SetDbState(curState.LoggerDbConnected)
				web.SetDbLastError(curState.LoggerDbError)
			}

		// получены считанные данные от сетевого контроллера
		case curWsPkg := <-netController.ReceiveDataChan:
			var msgHeader chief_logger.MessageHeader

			if err := json.Unmarshal(curWsPkg.Data, &msgHeader); err == nil {
				if msgHeader.Header == chief_logger.LogMessagesHeader {
					var logSMsg chief_logger.LoggerMsgSlice
					if err := json.Unmarshal(curWsPkg.Data, &logSMsg); err == nil {
						for _, msgIt := range logSMsg.Messages {
							web.AppendLog(msgIt)

							fileLogCntrl.MessChan <- msgIt
							oracleLogCntrl.MessChan <- msgIt
						}
					} else {
						log.Println("Error unmarshal log message. ", string(curWsPkg.Data), " Error: ", err)
					}
				} else if msgHeader.Header == chief_logger.LoggerSettingsHeader {
					go settingsRoutine()
				} else {
					log.Printf("Unknown message type. Data: %s.", string(curWsPkg.Data))
				}
			} else {
				log.Printf("Error unmarshal message Data: %s, Error: %s", string(curWsPkg.Data), err)
				continue
			}

		// получена ошибка от WS сервера
		case curWsErr := <-netController.ErrorChan:
			if curWsErr == nil {
				web.AppendLog(common.LogMessage{Text: fmt.Sprintf("Запускаем WS сервер для приема сообщений журнала."),
					Severity: common.SeverityInfo})
			} else {
				log.Printf("Web socket error: %s", curWsErr.Error())
				web.AppendLog(common.LogMessage{
					Text:     fmt.Sprintf("Возникла ошибка при работе WS сервера взаимодействия с FMTP каналами. Ошибка: <%s>.", curWsErr.Error()),
					Severity: common.SeverityError})
			}
			// получено уведомление от WS сервера
			//case wsInfo := <-netController.InfoChan:
			//	web.AppendLog(common.LogCntrlST(common.SeverityInfo, "Сервер WS для взаимодействия с FMTP каналами. "+wsInfo))
		}
	}
}
