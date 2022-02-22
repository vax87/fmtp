package chief_logger

import (
	"context"
	"fmt"
	"fmtp/chief/chief_metrics"
	"fmtp/fmtp_log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisLogController struct {
	LogMsgChan     chan fmtp_log.LogMessage
	SettsChan      chan RedisLoggerSettings
	msgToSend      []fmtp_log.LogMessage
	redisClnt      *redis.Client
	setts          RedisLoggerSettings
	successConnect bool
	msgMutex       sync.Mutex
}

func NewRedisController() *RedisLogController {
	return &RedisLogController{
		LogMsgChan:     make(chan fmtp_log.LogMessage, 1024),
		SettsChan:      make(chan RedisLoggerSettings, 10),
		msgToSend:      make([]fmtp_log.LogMessage, 0),
		successConnect: false,
	}
}

func (rc *RedisLogController) Run() {
	sendToStreamTicker := time.NewTicker(time.Second)

	for {
		select {
		case msg := <-rc.LogMsgChan:
			//fmt.Printf("<-rc.LogMsgChan %d %s\n", len(rc.msgToSend), time.Now().Format("2006-01-02 15:04:05.000"))
			if len(rc.msgToSend) < 10000 {
				rc.msgMutex.Lock()
				rc.msgToSend = append(rc.msgToSend, msg)
				rc.msgMutex.Unlock()
			} else {
				fmt.Println("!!!!!!!!!!!!!!!!!!Queue FULL")
			}

		case setts := <-rc.SettsChan:
			if rc.setts != setts {
				rc.setts = setts
				rc.connectToServer()
			}

		case <-sendToStreamTicker.C:
			//fmt.Printf("\tBEGIN rc.sendToStreamTicker.C %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
			countMsg := len(rc.msgToSend)
			if rc.successConnect && len(rc.msgToSend) > 0 {
				rc.msgMutex.Lock()
				toSend := make([]fmtp_log.LogMessage, 0)

				if countMsg > rc.setts.MaxSendCount {
					toSend = append(toSend, rc.msgToSend[:rc.setts.MaxSendCount]...)
					rc.msgToSend = rc.msgToSend[rc.setts.MaxSendCount:]
				} else {
					toSend = make([]fmtp_log.LogMessage, len(rc.msgToSend))
					copy(toSend, rc.msgToSend)
					rc.msgToSend = rc.msgToSend[:0]
				}
				if err := rc.pushLogsToStream(toSend); err != nil {
					fmt.Printf("Ошибка отправки сообщений в Redis: %v", err)
				}

				rc.msgMutex.Unlock()
			}
			//fmt.Printf("\tEND rc.sendToStreamTicker.C %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
		}
	}
}

func (rc *RedisLogController) connectToServer() {
	rc.redisClnt = redis.NewClient(&redis.Options{
		Addr:    fmt.Sprintf("%s:%d", rc.setts.Hostname, rc.setts.Port),
		Network: "tcp",
	})
	_, err := rc.redisClnt.Ping(context.Background()).Result()
	rc.successConnect = err == nil
	if err != nil {
		fmt.Printf("Ошибка подключения к Redis серверу: %v", err)
	}
}

func (rc *RedisLogController) pushLogsToStream(msgs []fmtp_log.LogMessage) error {
	//fmt.Printf("\t\tBEGIN pushLogsToStream  %d %s\n", len(msgs), time.Now().Format("2006-01-02 15:04:05.000"))

	pipe := rc.redisClnt.Pipeline()

	for _, val := range msgs {
		pipe.XAdd(context.Background(),
			&redis.XAddArgs{
				Stream: "FmtpLog",
				MaxLen: rc.setts.StreamMaxCount,
				Approx: true,
				ID:     "",
				Values: []interface{}{"msg", val.MarshalToString()},
			})
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()
	_, errPipe := pipe.Exec(ctx)

	if errPipe == nil {
		chief_metrics.RedisMetricsChan <- chief_metrics.RedisMetrics{
			Keys: 1,
			Msg:  len(msgs),
		}
	} else {
		chief_metrics.RedisMetricsChan <- chief_metrics.RedisMetrics{
			Err: 1,
		}
	}
	//fmt.Printf("\t\tEND pushLogsToStream %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
	return errPipe
}
