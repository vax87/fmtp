package redis_cntrl

import (
	"context"
	"encoding/json"
	"fmt"
	"fmtp/fmtp_log"
	"fmtp/ora_logger/metrics_cntrl"
	"time"

	"github.com/go-redis/redis/v8"
	"lemz.com/fdps/logger"
)

const (
	streamName = "FmtpLog"
	msgKey     = "msg"
)

type RedisCntrl struct {
	RequestMsgChan chan struct{}
	SendMsgChan    chan []fmtp_log.LogMessage
	SettsChan      chan RedisCntrlSettings

	MetricsChan chan metrics_cntrl.RedisMetrics

	redisClnt      *redis.Client
	setts          RedisCntrlSettings
	successConnect bool

	prevReadIds []string
}

func NewRedisController() *RedisCntrl {
	return &RedisCntrl{
		RequestMsgChan: make(chan struct{}, 10),
		SendMsgChan:    make(chan []fmtp_log.LogMessage, 10),
		SettsChan:      make(chan RedisCntrlSettings, 1),
		MetricsChan:    make(chan metrics_cntrl.RedisMetrics, 10),
		successConnect: false,
	}
}

func (c *RedisCntrl) Run() {
	c.MetricsChan <- metrics_cntrl.RedisMetrics{Msg: 0}

	for {
		select {

		case setts := <-c.SettsChan:
			if c.setts != setts {
				c.setts = setts
				c.connectToServer()
			}

		case <-c.RequestMsgChan:
			if len(c.prevReadIds) > 0 {
				if err := c.delLogsFromStream(); err != nil {
					logger.PrintfErr("Ошибка удаления из потока Redis %v", err)
				} else {
					c.prevReadIds = c.prevReadIds[:0]
				}
			}

			if msgs, err := c.readLogsFromStream(); err != nil {
				logger.PrintfErr("Ошибка чтения из потока Redis %v", err)
			} else {
				if len(msgs) > 0 {
					c.SendMsgChan <- msgs
				}
			}
		}
	}
}

func (c *RedisCntrl) connectToServer() {
	c.redisClnt = redis.NewClient(&redis.Options{
		Addr:    fmt.Sprintf("%s:%d", c.setts.Hostname, c.setts.Port),
		Network: "tcp",
	})
	_, err := c.redisClnt.Ping(context.Background()).Result()
	c.successConnect = err == nil
	if err != nil {
		logger.PrintfErr("Ошибка подключения к Redis серверу: %v", err)
	} else {
		logger.PrintfDebug("Успешное подключение к Redis серверу")
	}
}

func (c *RedisCntrl) readLogsFromStream() ([]fmtp_log.LogMessage, error) {
	retMsg := make([]fmtp_log.LogMessage, 0)
	if c.successConnect {
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
		defer cancelFunc()
		readSlice := c.redisClnt.XRead(ctx,
			&redis.XReadArgs{
				Streams: []string{streamName, "0"},
				Count:   int64(c.setts.MaxReadCount),
				Block:   0,
			})

		var readCnt metrics_cntrl.RedisMetrics

		for _, xStreamVal := range readSlice.Val() {
			for _, msgVal := range xStreamVal.Messages {
				c.prevReadIds = append(c.prevReadIds, msgVal.ID)
				msg, msgOk := msgVal.Values[msgKey]
				if msgOk {
					if msgString, ok := msg.(string); ok {
						var logMsg fmtp_log.LogMessage
						if err := json.Unmarshal([]byte(msgString), &logMsg); err == nil {
							retMsg = append(retMsg, logMsg)
						}
					}
				}
			}
		}
		c.MetricsChan <- readCnt
		return retMsg, readSlice.Err()
	}
	return retMsg, fmt.Errorf("Нет подключения к Redis")
}

func (c *RedisCntrl) delLogsFromStream() error {
	if c.successConnect {
		return c.redisClnt.XDel(context.Background(),
			streamName,
			c.prevReadIds...).Err()
	}
	return fmt.Errorf("Нет подключения к Redis")
}
