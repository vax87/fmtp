package chief_metrics

import (
	"lemz.com/fdps/prom_metrics"
)

const (
	metricChan  = "chan"
	metricFdps  = "fdps"
	metricRedis = "redis"
)

////////////////////////////////////////////////////////////////////////////////////

const RedisTypeLabel = "tp"

const RedisTpMsg = "msg"
const RedisTpErr = "err"

type RedisMetrics struct {
	Msg int
	Err int
}

////////////////////////////////////////////////////////////////////////////////////

const ChanTypeLabel = "tp"
const ChanTpSend = "send"
const ChanTpRecv = "recv"

const ChanLocAtcLabel = "latc"
const ChanRemAtcLabel = "ratc"

type ChanMetrics struct {
	Tp     string
	LocAtc string
	RemAtc string
	Count  int
}

////////////////////////////////////////////////////////////////////////////////////

const ProvTypeLabel = "tp"

const ProvTpSend = "send"
const ProvTpRecv = "recv"
const ProvTpMiss = "miss"
const ProvTpTimeout = "tout"

type ProvMetrics struct {
	SendCount    int
	RecvCount    int
	MissedCount  int // не нашлось канала для отправки
	TimeoutCount int // не отправлены провайдеру в течении 30 сек
}

////////////////////////////////////////////////////////////////////////////////////

type ChiefMetricsCntrl struct {
	SettsChan chan prom_metrics.PusherSettings
}

var (
	ChanMetricsChan  = make(chan ChanMetrics, 10)
	ProvMetricsChan  = make(chan ProvMetrics, 10)
	RedisMetricsChan = make(chan RedisMetrics, 10)
)

func NewChiefMetricsCntrl() *ChiefMetricsCntrl {
	return &ChiefMetricsCntrl{
		SettsChan: make(chan prom_metrics.PusherSettings, 10),
	}
}

func (c *ChiefMetricsCntrl) Run() {

	for {
		select {

		case setts := <-c.SettsChan:
			prom_metrics.SetSettings(setts)
			prom_metrics.AppendCounterVec(metricChan, "Канал", []string{ChanTypeLabel, ChanLocAtcLabel, ChanRemAtcLabel})
			prom_metrics.AppendCounterVec(metricFdps, "Провайдер", []string{ProvTypeLabel})
			prom_metrics.AppendCounterVec(metricRedis, "Redis", []string{RedisTypeLabel})
			prom_metrics.Initialize()

		case chMt := <-ChanMetricsChan:
			prom_metrics.AddToCounterVec(metricChan, chMt.Count, map[string]string{
				ChanTypeLabel:   chMt.Tp,
				ChanLocAtcLabel: chMt.LocAtc,
				ChanRemAtcLabel: chMt.RemAtc,
			})

		case prMt := <-ProvMetricsChan:
			prom_metrics.AddToCounterVec(metricFdps, prMt.SendCount, map[string]string{ProvTypeLabel: ProvTpSend})
			prom_metrics.AddToCounterVec(metricFdps, prMt.RecvCount, map[string]string{ProvTypeLabel: ProvTpRecv})
			prom_metrics.AddToCounterVec(metricFdps, prMt.MissedCount, map[string]string{ProvTypeLabel: ProvTpMiss})
			prom_metrics.AddToCounterVec(metricFdps, prMt.TimeoutCount, map[string]string{ProvTypeLabel: ProvTpTimeout})

		case rdMt := <-RedisMetricsChan:
			if rdMt.Msg > 0 {
				prom_metrics.AddToCounterVec(metricRedis, rdMt.Msg, map[string]string{RedisTypeLabel: RedisTpMsg})
			}
			if rdMt.Err > 0 {
				prom_metrics.AddToCounterVec(metricRedis, rdMt.Err, map[string]string{RedisTypeLabel: RedisTpErr})
			}
		}
	}
}
