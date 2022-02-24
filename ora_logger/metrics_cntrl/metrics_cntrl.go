package metrics_cntrl

import (
	"fmtp/ora_logger/logger_state"

	prom_metrics "lemz.com/fdps/prom_metrics"
)

const (
	metricRedisMsg        = "redis_read_msg"
	metricOraQueries      = "ora_exec_queries"
	metricOraMsgBuffer    = "ora_msg_buffer"
	metricOraQueriesQueue = "ora_queries_queue"
)

type RedisMetrics struct {
	Msg int
}

const OraTypeLabel = "tp"

type OraMetrics struct {
	Count        int
	Labels       map[string]string
	MsgBuffer    int
	QueriesQueue int
}

type MetricsCntrl struct {
	SettsChan        chan prom_metrics.PusherSettings
	RedisMetricsChan chan RedisMetrics
	OraMetricsChan   chan OraMetrics
}

func NewMetricsCntrl() *MetricsCntrl {
	return &MetricsCntrl{
		SettsChan:        make(chan prom_metrics.PusherSettings, 10),
		RedisMetricsChan: make(chan RedisMetrics, 10),
		OraMetricsChan:   make(chan OraMetrics, 10),
	}
}

func (c *MetricsCntrl) Run() {

	checkErrFunc := func() {
		if err := prom_metrics.GetPushError(); err == nil {
			logger_state.SetMetricsState(logger_state.StateOk, "")
		} else {
			logger_state.SetMetricsState(logger_state.StateError, err.Error())
		}
	}

	for {
		select {

		case setts := <-c.SettsChan:
			prom_metrics.SetSettings(setts)
			prom_metrics.AppendCounter(metricRedisMsg, "Кол-во считанных сообщений журнала из потока Redis")
			prom_metrics.AppendCounterVec(metricOraQueries, "Кол-во выполненных запросов к Oracle", []string{OraTypeLabel})
			prom_metrics.AppendGauge(metricOraMsgBuffer, "Размер буфера сообщений журнала контроллера Oracle")
			prom_metrics.AppendGauge(metricOraQueriesQueue, "Размер очереди запросов контроллера Oracle")
			prom_metrics.Initialize()

		case rdMt := <-c.RedisMetricsChan:
			prom_metrics.AddToCollector(metricRedisMsg, rdMt.Msg)
			checkErrFunc()

		case oraMt := <-c.OraMetricsChan:
			prom_metrics.AddToCounterVec(metricOraQueries, oraMt.Count, oraMt.Labels)
			prom_metrics.SetToGauge(metricOraMsgBuffer, oraMt.MsgBuffer)
			prom_metrics.SetToGauge(metricOraQueriesQueue, oraMt.QueriesQueue)
			checkErrFunc()
		}
	}
}
