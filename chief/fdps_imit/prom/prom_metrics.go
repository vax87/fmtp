package prom_metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	countMsgSend = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "fmtp",
			Subsystem: "fdps_imit",
			Name:      "count_send",
			Help:      "Кол-во отправленных FMTP сообщений",
		})

	countMsgRecv = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "fmtp",
			Subsystem: "fdps_imit",
			Name:      "count_recv",
			Help:      "Кол-во полученных FMTP сообщений",
		})

	msgSendPerSecond = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "fmtp",
			Subsystem: "fdps_imit",
			Name:      "count_send_per_second",
			Help:      "Кол-во отправленных FMTP сообщений в секунду",
		})

	msgRecvPerSecond = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "fmtp",
			Subsystem: "fdps_imit",
			Name:      "count_recv_per_second",
			Help:      "Кол-во полученных FMTP сообщений в секунду",
		})

	countMsgMissed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "fmtp",
			Subsystem: "fdps_imit",
			Name:      "count_missed",
			Help:      "Кол-во отправленных и не принятых FMTP сообщений",
		})

	avgSendPerSecond = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "fmtp",
			Subsystem: "fdps_imit",
			Name:      "avg_send_per_second",
			Help:      "В среднем отправлено FMTP сообщений в секунду",
		})

	avgRecvPerSecond = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "fmtp",
			Subsystem: "fdps_imit",
			Name:      "avg_recv_per_second",
			Help:      "В среднем получено FMTP сообщений в секунду",
		})

	registry = prometheus.NewRegistry()
)

var metricsMutex sync.Mutex
var sendMsgCount, recvMsgCount int64
var missedMsgCount int64
var pusher *push.Pusher

func init() {
	registry.MustRegister(countMsgSend,
		countMsgRecv,
		msgSendPerSecond,
		msgRecvPerSecond,
		countMsgMissed,
		avgSendPerSecond,
		avgRecvPerSecond)

	pusher = push.New("http://192.168.1.24:9100", "pushgateway").Gatherer(registry)
}

func AddMsgSendCount(sendCount int) {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	countMsgSend.Add(float64(sendCount))
	sendMsgCount += int64(sendCount)
}

func AddMsgRecvCount(recvCount int) {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	countMsgRecv.Add(float64(recvCount))
	recvMsgCount += int64(recvCount)
}

func AddMsgMissedCount(missedCount int) {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	countMsgMissed.Add(float64(missedCount))
	missedMsgCount += int64(missedCount)
}

func Work(addr string) {
	secondTicker := time.NewTicker(time.Second)

	var prevSecondSendCount, prevSecondRecvCount int64

	// go func() {
	// 	http.Handle("/metrics", promhttp.Handler())

	// 	log.Printf("Starting web server at %s\n", addr)
	// 	err := http.ListenAndServe(addr, nil)
	// 	if err != nil {
	// 		log.Printf("http.ListenAndServer: %v\n", err)
	// 	}

	beginWorkTime := time.Now().UTC()

	for {
		select {
		case <-secondTicker.C:
			metricsMutex.Lock()

			msgSendPerSecond.Set(float64(sendMsgCount - prevSecondSendCount))
			prevSecondSendCount = sendMsgCount

			msgRecvPerSecond.Set(float64(recvMsgCount - prevSecondRecvCount))
			prevSecondRecvCount = recvMsgCount

			if diffTime := time.Now().UTC().Sub(beginWorkTime).Seconds(); diffTime > 0 {
				avgSendPerSecond.Set(float64(sendMsgCount) / diffTime)
				avgRecvPerSecond.Set(float64(recvMsgCount) / diffTime)
			} else {
				avgSendPerSecond.Set(0.0)
				avgRecvPerSecond.Set(0.0)
			}

			// _ = push.New("http://192.168.1.24:9100", "pushgateway").Collector(countMsgSend).Grouping("fmtp", "1").Push()
			// _ = push.New("http://192.168.1.24:9100", "pushgateway").Collector(countMsgRecv).Grouping("fmtp", "2").Push()
			// _ = push.New("http://192.168.1.24:9100", "pushgateway").Collector(msgSendPerSecond).Grouping("fmtp", "3").Push()
			// _ = push.New("http://192.168.1.24:9100", "pushgateway").Collector(msgRecvPerSecond).Grouping("fmtp", "4").Push()
			// _ = push.New("http://192.168.1.24:9100", "pushgateway").Collector(countMsgMissed).Grouping("db", "5").Push()
			// _ = push.New("http://192.168.1.24:9100", "pushgateway").Collector(avgSendPerSecond).Grouping("db", "6").Push()
			// _ = push.New("http://192.168.1.24:9100", "pushgateway").Collector(avgRecvPerSecond).Grouping("db", "7").Push()
			//pusher.Collector(countMsgSend)
			//pusher.Collector(countMsgRecv)

			if err := pusher.Add(); err != nil {
				fmt.Println("Could not push to Pushgateway:", err)
			}

			metricsMutex.Unlock()
		}
	}
}
