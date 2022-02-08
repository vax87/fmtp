package main

import (
	"fdps/fmtp/ora_logger/ora_cntrl"
	"sync"

	"github.com/go-redis/redis"
)

// контроллер записи в БД oracle
var oraLogCntrl = ora_cntrl.NewOraController()
var redisClnt *redis.Client

func main() {

	go oraLogCntrl.Run()

	var wg sync.WaitGroup
	wg.Add(1)
	// for {
	// 	select {}
	// }
	wg.Wait()
}
