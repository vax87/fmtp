package main

import (
	cfg "fmtp/configurator"

	"fmtp/ora_logger/metrics_cntrl"
	"fmtp/ora_logger/ora_cntrl"
	"fmtp/ora_logger/redis_cntrl"

	"lemz.com/fdps/logger"
	"lemz.com/fdps/prom_metrics"
	"lemz.com/fdps/utils"
)

const (
	appName    = "fdps-ora-logger"
	appVersion = "2022-02-18 15:49"
)

var (
	metricsCntrl = metrics_cntrl.NewMetricsCntrl()
	oraCntrl     = ora_cntrl.NewOraController()
	redisCntrl   = redis_cntrl.NewRedisController()
)

func main() {
	logger.InitLoggerSettings(utils.AppPath()+"/config/loggers.json", appName, appVersion)
	if logger.LogSettInst.NeedWebLog {
		utils.AppendHandler(logger.WebLogger)
	}

	loggerConfClient := cfg.NewLoggerClient()

	go metricsCntrl.Run()
	go oraCntrl.Run()
	go redisCntrl.Run()

	go loggerConfClient.Work()
	go loggerConfClient.Start()

	for {
		select {

		case <-loggerConfClient.LoggerSettChangedChan:

			metricsCntrl.SettsChan <- prom_metrics.PusherSettings{
				PusherIntervalSec: cfg.LoggerCfg.MetricsIntervalSec,
				GatewayUrl:        cfg.LoggerCfg.MetricsGatewayUrl,
				GatewayJob:        "fmtp",
				CollectNamespace:  "fmtp",
				CollectSubsystem:  "logger",
				CollectLabels:     map[string]string{"host": cfg.LoggerCfg.IPAddr},
			}

			oraCntrl.SettsChan <- ora_cntrl.OraCntrlSettings{
				Hostname:         cfg.LoggerCfg.OraHostname,
				Port:             cfg.LoggerCfg.OraPort,
				ServiceName:      cfg.LoggerCfg.OraServiceName,
				UserName:         cfg.LoggerCfg.OraUser,
				Password:         cfg.LoggerCfg.OraPassword,
				LogStoreMaxCount: cfg.LoggerCfg.OraMaxLogStoreCount,
				LogStoreDays:     cfg.LoggerCfg.OraStoreDays,
			}

			redisCntrl.SettsChan <- redis_cntrl.RedisCntrlSettings{
				Hostname: cfg.LoggerCfg.RedisHostname,
				Port:     cfg.LoggerCfg.RedisPort,
				DbId:     cfg.LoggerCfg.RedisDbId,
				UserName: cfg.LoggerCfg.RedisUserName,
				Password: cfg.LoggerCfg.RedisPassword,

				StreamMaxCount: cfg.LoggerCfg.RedisStreamMaxCount,
			}

		case <-oraCntrl.RequestMsgChan:
			redisCntrl.RequestMsgChan <- struct{}{}

		case logMsg := <-redisCntrl.SendMsgChan:
			oraCntrl.ReceiveMsgChan <- logMsg

		case redisMt := <-redisCntrl.MetricsChan:
			metricsCntrl.RedisMetricsChan <- redisMt

		case oraMt := <-oraCntrl.MetricsChan:
			metricsCntrl.OraMetricsChan <- oraMt
		}
	}
}
