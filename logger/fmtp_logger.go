package main

import (
	"log"
	"sync"

	"fdps/fmtp/logger/logger_worker"
	"fdps/utils"
	"fdps/utils/logger"
	"fdps/utils/logger/log_ljack"
	"fdps/utils/logger/log_std"
	"fdps/utils/logger/log_web"
)

const (
	appName    = "fdps-fmtp-logger"
	appVersion = "2020-06-25 20:04"
)

func initLoggers() {
	// логгер с web страничкой
	log_web.Initialize(log_web.LogWebSettings{
		StartHttp:  true,
		LogURLPath: utils.FmtpLoggerWebPath,
		NetPort:    utils.FmtpLoggerWebPort,
		Title:      appName,
		ShowSetts:  false,
	})
	logger.AppendLogger(log_web.WebLogger)
	utils.AppendHandler(log_web.WebLogger)
	log_web.SetVersion(appVersion)

	// логгер с сохранением в файлы
	var ljackSetts log_ljack.LjackSettings
	ljackSetts.GenDefault()
	ljackSetts.FilesName = appName + ".log"
	if err := log_ljack.Initialize(ljackSetts); err != nil {
		logger.PrintfErr("Ошибка инициализации логгера lumberjack. Ошибка: %s", err.Error())
	}
	logger.AppendLogger(log_ljack.LogLjack)

	logger.AppendLogger(log_std.LogStd)

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC | log.Llongfile)
}

func initDockerInfo() bool {
	if dockerVersion, dockErr := utils.GetDockerVersion(); dockErr != nil {
		log_web.SetDockerVersion("???")
		logger.PrintfErr(dockErr.Error())
		return false
	} else {
		log_web.SetDockerVersion(dockerVersion)
	}
	return true
}

func main() {
	initLoggers()

	done := make(chan struct{})

	wg := sync.WaitGroup{}
	wg.Add(1)

	go logger_worker.Start(done, &wg)
	wg.Wait()
}
