package redis_cntrl

import (
	"context"
	"encoding/json"
	"fdps/fmtp/fmtp_log"
	"fdps/fmtp/ora_logger/metrics_cntrl"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/go-redis/redis"
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
					fmt.Printf("Error DEL from stream %v", err)
				} else {
					c.prevReadIds = c.prevReadIds[:0]
				}
			}

			if msgs, err := c.readLogsFromStream(); err != nil {
				fmt.Printf("Error READ from stream %v", err)
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
		fmt.Printf("Ошибка подключения к Redis серверу: %v", err)
	}
}

func (c *RedisCntrl) readLogsFromStream() ([]fmtp_log.LogMessage, error) {
	retMsg := make([]fmtp_log.LogMessage, 0)
	if c.successConnect {
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
		defer cancelFunc()
		readSlice := c.redisClnt.XRead(ctx,
			&redis.XReadArgs{
				Streams: []string{"FmtpLog", "0"},
				Count:   10,
				Block:   0,
			})

		//fmt.Printf("Stream %s", readSlice.String())
		var readCnt metrics_cntrl.RedisMetrics

		for _, xStreamVal := range readSlice.Val() {
			//fmt.Printf("Stream %s", xStreamVal.Stream)

			readCnt.Keys += len(xStreamVal.Messages)
			for _, msgVal := range xStreamVal.Messages {
				//	fmt.Printf("\tMsg id %s\n", msgVal.ID)
				c.prevReadIds = append(c.prevReadIds, msgVal.ID)

				msgKeys := make([]string, 0)
				for key, _ := range msgVal.Values {
					msgKeys = append(msgKeys, key)
				}
				countMsg, ok := msgVal.Values["count"]
				if ok {
					if val, err := strconv.Atoi(countMsg.(string)); err == nil {
						readCnt.Msg += val
					}
				}
				sort.Strings(msgKeys)

				//readCnt.Keys += len(msgKeys)

				for _, key := range msgKeys {
					if msgString, ok := msgVal.Values[key].(string); ok {
						var logMsg fmtp_log.LogMessage
						if err := json.Unmarshal([]byte(msgString), &logMsg); err == nil {
							retMsg = append(retMsg, logMsg)
							//fmt.Printf("\t\tMsg key %v value %v\n", key, logMsg)
						}
					} else {
						fmt.Printf("\t\t--Msg key %v value %v\n", key, msgVal.Values[key])
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
			"FmtpLog",
			c.prevReadIds...).Err()
	}
	return fmt.Errorf("Нет подключения к Redis")
}
