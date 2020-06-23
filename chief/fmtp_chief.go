package main

import (
	"log"
	"sync"

	"fdps/fmtp/chief/chief_web"
	"fdps/fmtp/chief/chief_worker"
	"fdps/fmtp/chief_logger"

	"fdps/utils"
	"fdps/utils/logger"
	"fdps/utils/logger/log_ljack"
	"fdps/utils/logger/log_std"
	"fdps/utils/logger/log_web"
)

const (
	appName    = "fdps-fmtp-chief"
	appVersion = "2020-06-10 15:29"
)

var workWithDocker bool
var dockerVersion string

var done = make(chan struct{}, 1)

func initLoggers() {
	// логгер с web страничкой
	log_web.Initialize(log_web.LogWebSettings{
		StartHttp:    false,
		LogURLPath:   utils.FmtpChiefWebLogPath,
		Title:        appName,
		ShowSetts:    true,
		SettsURLPath: utils.FmtpChiefWebConfigPath,
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

	logger.AppendLogger(chief_logger.ChiefLog)
	go chief_logger.ChiefLog.Work()
}

func initDockerInfo() bool {
	if dockerVersion, dockErr := utils.GetDockerVersion(); dockErr != nil {
		log_web.SetDockerVersion("???")
		logger.PrintfErr(dockErr.Error())
		return false
	} else {
		workWithDocker = true
		log_web.SetDockerVersion(dockerVersion)
	}
	return true
}

func main() {
	initLoggers()
	if initDockerInfo() == false {
		utils.InitFileBinUtils(
			utils.AppPath()+"/versions",
			utils.AppPath()+"/runningChannels",
			".exe",
			"fmtp_channel",
			"FMTP канал",
		)
	}

	done := make(chan struct{})
	chief_web.Start(done)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go chief_worker.Start(workWithDocker, dockerVersion, done, &wg)
	wg.Wait()

}
