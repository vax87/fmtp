package logger_worker

import (
	"encoding/json"
	"fdps/fmtp/chief_logger"
	"fdps/fmtp/logger/common"
	"fdps/fmtp/logger/file"
	"fdps/fmtp/logger/oracle"
	"fdps/fmtp/web"
	"fdps/utils"
	"fdps/utils/logger"
	"fdps/utils/web_sock"
	"fmt"
	"log"
	"sync"
)

func Start(done chan struct{}, wg *sync.WaitGroup) {
	// контроллер записи в файлы
	var fileLogCntrl = &file.FileLoggerController{MessChan: make(chan common.LogMessage, 1024),
		SettingsChan: make(chan file.FileLoggerSettings)}

	// контроллер записи в БД oracle
	var oracleLogCntrl = oracle.NewOracleController()

	// контроллер для работы с сетью
	var netController = web_sock.NewWebSockServer(done)

	readLoggerSettingsFunc := func() {
		var setts common.LoggerSettings
		if errRead := setts.ReadFromFile(); errRead != nil {
			logger.PrintfErr("Ошибка чтения настроек из файла: %v", errRead)
			setts.DefaultInit()
			logger.PrintfInfo("Настройки инициализирован значениями по умолчанию: %#v", setts)
			if errSave := setts.SaveToFile(); errSave != nil {
				logger.PrintfErr("Ошибка сохранения настроек в файл: %v", errSave)
			} else {
				logger.PrintfErr("Настройки сохранены в файл")
			}
		}

		fileLogCntrl.SettingsChan <- file.FileLoggerSettings{
			LogFileSizeKb:   setts.FileSizeKB,
			LogFolderSizeGb: setts.FolderSizeGB,
			LogStoreDays:    setts.DbStoreDays,
		}

		oracleLogCntrl.SettingsChan <- oracle.OracleLoggerSettings{
			Hostname:         setts.DbHostname,
			Port:             setts.DbPort,
			ServiceName:      setts.DbServiceName,
			UserName:         setts.DbUser,
			Password:         setts.DbPassword,
			LogStoreMaxCount: setts.DbMaxLogStoreCount,
			LogStoreDays:     setts.DbStoreDays,
		}

		netController.SettingsChan <- web_sock.WebSockServerSettings{
			Port: setts.LoggerPort,
		}
	}

	// подпрограмма для контроллера сети
	go netController.Work("/" + utils.FmtpLoggerWsPath)

	// подпрограмма для контроллера логов(запись в файлы)
	go fileLogCntrl.Run()

	// подпрограмма для контроллера логов(запись в БД oracle)
	go oracleLogCntrl.Run()

	readLoggerSettingsFunc()

	for {
		select {
		// получено сообщение о состоянии контроллера записи в БД oracle
		case curState := <-oracleLogCntrl.StateChan:
			stateData, stateErr := json.Marshal(curState)
			if stateErr != nil {
				log.Println("State message marshal error. ", stateErr)
			} else {
				netController.SendDataChan <- web_sock.WsPackage{Data: stateData}
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
							//logger.PrintfInfo(msgIt.Text)

							fileLogCntrl.MessChan <- msgIt
							oracleLogCntrl.MessChan <- msgIt
						}
					} else {
						log.Println("Error unmarshal log message. ", string(curWsPkg.Data), " Error: ", err)
					}
				} else if msgHeader.Header == chief_logger.LoggerSettingsHeader {
					readLoggerSettingsFunc()
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
