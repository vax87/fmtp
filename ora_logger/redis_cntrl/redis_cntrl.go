package redis_cntrl

import (
	"context"
	"fdps/fmtp/fmtp_log"
	"fmt"

	"github.com/go-redis/redis"
)

type RedisController struct {
	OraRequestMsgChan chan struct{}
	MsgToOraChan      chan []fmtp_log.LogMessage
	SettsChan         chan RedisCntrlSettings

	redisClnt      *redis.Client
	setts          RedisCntrlSettings
	successConnect bool
}

func NewRedisController() *RedisController {
	return &RedisController{
		OraRequestMsgChan: make(chan struct{}, 1),
		MsgToOraChan:      make(chan []fmtp_log.LogMessage, 1),
		SettsChan:         make(chan RedisCntrlSettings, 1),
		successConnect:    false,
	}
}

func (rc *RedisController) Run() {
	for {
		select {

		case setts := <-rc.SettsChan:
			if rc.setts != setts {
				rc.setts = setts
				rc.connectToServer()
			}

		case <-rc.OraRequestMsgChan:
			if _, err := rc.readLogsFromStream(); err != nil {
				fmt.Printf("Error READ from stream %v", err)
			}
		}
	}
}

func (rc *RedisController) connectToServer() {
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

func (rc *RedisController) readLogsFromStream() ([]fmtp_log.LogMessage, error) {
	if rc.successConnect {
		readSlice := rc.redisClnt.XRead(context.Background(),
			&redis.XReadArgs{
				Streams: []string{"FmtpLog", "0"},
				Count:   1,
				Block:   0,
			})
		for _, xStreamVal := range readSlice.Val() {
			fmt.Printf("Stream %s", xStreamVal.Stream)
			for _, msgVal := range xStreamVal.Messages {
				fmt.Printf("\tMsg id %s\n", msgVal.ID)
				for key, _ := range msgVal.Values {
					fmt.Printf("\t\tMsg key %v\n", key)
				}
			}
		}
		return []fmtp_log.LogMessage{}, readSlice.Err()
	}
	return []fmtp_log.LogMessage{}, fmt.Errorf("Нет подключения к Redis")
}
