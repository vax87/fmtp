package main

import (
	"fdps/fmtp/ora_logger/metrics_cntrl"
	"fdps/fmtp/ora_logger/ora_cntrl"
	"fdps/fmtp/ora_logger/redis_cntrl"
	"fdps/go_utils/prom_metrics"
)

var (
	metricsCntrl = metrics_cntrl.NewMetricsCntrl()
	oraCntrl     = ora_cntrl.NewOraController()
	redisCntrl   = redis_cntrl.NewRedisController()
)

func main() {
	go metricsCntrl.Run()
	go oraCntrl.Run()
	go redisCntrl.Run()

	//!!! только один раз можно вызвать
	metricsCntrl.SettsChan <- prom_metrics.PusherSettings{
		PusherIntervalSec: 1,
		GatewayUrl:        "http://192.168.1.24:9100", // from lemz
		//GatewayUrl:       "http://127.0.0.1:9100",	// from home
		GatewayJob:       "fmtp",
		CollectNamespace: "fmtp",
		CollectSubsystem: "logger",
		CollectLabels:    map[string]string{"host": "192.168.10.219"},
	}
	//!!!

	oraCntrl.SettsChan <- ora_cntrl.OraCntrlSettings{
		Hostname:         "192.168.1.30",
		Port:             1521,
		ServiceName:      "metplan",
		UserName:         "fmtp_log",
		Password:         "log",
		LogStoreMaxCount: 2400000,
		LogStoreDays:     30,
	}

	redisCntrl.SettsChan <- redis_cntrl.RedisCntrlSettings{
		Hostname: "192.168.1.24", // from lemz
		//Hostname: "127.0.0.1", // from home
		Port:     6389,
		DbId:     0,
		UserName: "",
		Password: "",

		StreamMaxCount:   1000,
		SendIntervalMSec: 20,
		MaxSendCount:     50,
	}

	for {
		select {
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

	// time.Sleep(time.Second)

	// go func() {
	// 	for idx := 0; idx < 10; idx++ {
	// 		redisCntrl.OraRequestMsgChan <- struct{}{}
	// 		time.Sleep(time.Second)
	// 	}
	// }()

	// var wg sync.WaitGroup
	// wg.Add(1)
	// // for {
	// // 	select {}
	// // }
	// wg.Wait()
}
