package main

import (
	"sync"

	"fdps/fmtp/chief/chief_logger"
	"fdps/fmtp/chief/chief_web"
	"fdps/fmtp/chief/chief_worker"
	"fdps/utils"

	"fdps/go_utils/logger"
)

const (
	appName    = "fdps-fmtp-chief"
	appVersion = "2020-06-10 15:29"
)

var workWithDocker bool
var dockerVersion string

func initDockerInfo() bool {
	if dockerVersion, dockErr := utils.GetDockerVersion(); dockErr != nil {
		logger.SetDockerVersion("???")
		logger.PrintfErr(dockErr.Error())
		return false
	} else {
		workWithDocker = true
		logger.SetDockerVersion(dockerVersion)
	}
	return true
}

func main() {
	logger.InitLoggerSettings(utils.AppPath()+"/config/local_logger.json", appName, appVersion)
	if logger.LogSettInst.NeedWebLog {
		utils.AppendHandler(logger.WebLogger)
	}

	chief_logger.ChiefLog.SetMinSeverity(logger.SevDebug)
	logger.AppendLogger(chief_logger.ChiefLog)
	go chief_logger.ChiefLog.Work()

	if !initDockerInfo() {
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
