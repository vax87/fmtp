package redis_cntrl

import (
	"context"
	"encoding/json"
	"fdps/fmtp/fmtp_log"
	"fmt"
	"sort"

	"github.com/go-redis/redis"
)

type RedisController struct {
	RequestMsgChan chan struct{}
	SendMsgChan    chan []fmtp_log.LogMessage
	SettsChan      chan RedisCntrlSettings

	redisClnt      *redis.Client
	setts          RedisCntrlSettings
	successConnect bool

	prevReadIds []string
}

func NewRedisController() *RedisController {
	return &RedisController{
		RequestMsgChan: make(chan struct{}, 1),
		SendMsgChan:    make(chan []fmtp_log.LogMessage, 1),
		SettsChan:      make(chan RedisCntrlSettings, 1),
		successConnect: false,
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

		case <-rc.RequestMsgChan:
			if len(rc.prevReadIds) > 0 {
				if err := rc.delLogsFromStream(); err != nil {
					fmt.Printf("Error DEL from stream %v", err)
				} else {
					rc.prevReadIds = rc.prevReadIds[:0]
				}
			}

			if msgs, err := rc.readLogsFromStream(); err != nil {
				fmt.Printf("Error READ from stream %v", err)
			} else {
				if len(msgs) > 0 {
					rc.SendMsgChan <- msgs
				}
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
	retMsg := make([]fmtp_log.LogMessage, 0)
	if rc.successConnect {
		readSlice := rc.redisClnt.XRead(context.Background(),
			&redis.XReadArgs{
				Streams: []string{"FmtpLog", "0"},
				Count:   1,
				Block:   0,
			})

		//fmt.Printf("Stream %s", readSlice.String())

		for _, xStreamVal := range readSlice.Val() {
			//fmt.Printf("Stream %s", xStreamVal.Stream)

			for _, msgVal := range xStreamVal.Messages {
				//	fmt.Printf("\tMsg id %s\n", msgVal.ID)
				rc.prevReadIds = append(rc.prevReadIds, msgVal.ID)

				msgKeys := make([]string, 0)
				for key, _ := range msgVal.Values {
					msgKeys = append(msgKeys, key)
				}
				countMsg, ok := msgVal.Values["count"]
				if ok {
					fmt.Printf("count MSG %d\n", countMsg)
				}

				sort.Strings(msgKeys)
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
		return retMsg, readSlice.Err()
	}
	return retMsg, fmt.Errorf("Нет подключения к Redis")
}

func (rc *RedisController) delLogsFromStream() error {
	if rc.successConnect {
		return rc.redisClnt.XDel(context.Background(),
			"FmtpLog",
			rc.prevReadIds...).Err()
	}
	return fmt.Errorf("Нет подключения к Redis")
}
