package redis_cntrl

import (
	"context"
	"fdps/fmtp/fmtp_log"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

type RedisLogController struct {
	LogMsgChan         chan fmtp_log.LogMessage
	SettsChan          chan RedisLoggerSettings
	msgToSend          []fmtp_log.LogMessage
	redisClnt          *redis.Client
	sendToStreamTicker *time.Ticker
	setts              RedisLoggerSettings
	successConnect     bool
	msgMutex           sync.Mutex
}

func NewRedisController() *RedisLogController {
	return &RedisLogController{
		LogMsgChan:         make(chan fmtp_log.LogMessage, 1024),
		SettsChan:          make(chan RedisLoggerSettings, 10),
		msgToSend:          make([]fmtp_log.LogMessage, 0),
		sendToStreamTicker: time.NewTicker(time.Second),
		successConnect:     false,
	}
}

func (rc *RedisLogController) Run() {
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
				rc.sendToStreamTicker.Reset(time.Duration(rc.setts.SendIntervalMSec))
			}

		case <-rc.sendToStreamTicker.C:
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

	var values []interface{}
	values = append(values, "count")
	values = append(values, len(msgs))

	for idx, val := range msgs {
		values = append(values, fmt.Sprintf("msg%d", idx))
		values = append(values, val)
	}
	err := rc.redisClnt.XAdd(context.Background(),
		&redis.XAddArgs{
			Stream: "FmtpLog",
			MaxLen: rc.setts.StreamMaxCount,
			Approx: true,
			ID:     "",
			Values: values,
		}).Err()

	//fmt.Printf("\t\tEND pushLogsToStream %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
	return err
}
