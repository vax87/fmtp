package ora_cntrl

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/godror/godror"
	"github.com/golang-collections/go-datastructures/queue"

	"fmtp/fmtp_log"
	"fmtp/ora_logger/metrics_cntrl"
)

const (
	timeFormat        = "2006-01-02 15:04:05"
	insertCountCheck  = 1000 // кол-во запросов INSERT в БД, после чего следует проверить кол-во хранимых сообщений
	onlineLogMaxCount = 1000 // кол-во хранимых логов в таблице онлайн сообщений
	maxQueryExec      = 100  //кол-во логов, записываемых за раз
	errMetricLabel    = "error"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	LogPriority = iota
	CheckLogCountPriority
	CheckLogLivetimePriority
	HeartbeatPriority
)

type queueItem struct {
	queryText string
	priority  int
	countMsg  int
}

func (qi queueItem) Compare(other queue.Item) int {
	if cast, ok := other.(queueItem); ok {
		if qi.priority < cast.priority {
			return 1
		}
	}
	return -1
}

func PriorToString(val int) (priorStr string) {
	switch val {
	case LogPriority:
		priorStr = "log"
	case CheckLogCountPriority:
		priorStr = "check_count"
	case CheckLogLivetimePriority:
		priorStr = "check_time"
	case HeartbeatPriority:
		priorStr = "heartbeat"
	}
	return
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// контроллер, выполняющий запись логов в БД
type OraCntrlr struct {
	sync.Mutex

	SettsChan      chan OraCntrlSettings      // канал приема новых настроек контроллера
	ReceiveMsgChan chan []fmtp_log.LogMessage // канал приема новых сообщений
	RequestMsgChan chan struct{}

	MetricsChan chan metrics_cntrl.OraMetrics

	curSetts OraCntrlSettings

	logMsgBuffer []fmtp_log.LogMessage // очередь логов
	queryQueue   *queue.PriorityQueue  // очередь сообщений для записи в БД

	db         *sql.DB // объект БД
	dbSuccess  bool    // успешность подключения к БД
	canExecute bool    // возможность выполнять запросы к БД

	execQueryChan  chan queueItem // канал для передачи запросов записи логов на выполнение
	execResultChan chan error     // канал для передачи результатов выполнения
	execTermChan   chan struct{}  // канал для завершения подпрограммы выполнения запросов

	logLifetimeTicker *time.Ticker // тикер проверки времени жизни логов
	pingDbTicker      *time.Ticker // тикер пинга БД
	curInsertCount    uint64       // кол-во выполнненых запросов INSERT (для проверки кол-ва хранимых логов)
}

func NewOraController() *OraCntrlr {
	return &OraCntrlr{
		ReceiveMsgChan:    make(chan []fmtp_log.LogMessage, 1024),
		SettsChan:         make(chan OraCntrlSettings, 1),
		RequestMsgChan:    make(chan struct{}, 10),
		MetricsChan:       make(chan metrics_cntrl.OraMetrics, 10),
		queryQueue:        queue.NewPriorityQueue(10000),
		canExecute:        false,
		execQueryChan:     make(chan queueItem, 1),
		execResultChan:    make(chan error, 1),
		execTermChan:      make(chan struct{}, 1),
		pingDbTicker:      time.NewTicker(3 * time.Second),
		logLifetimeTicker: time.NewTicker(12 * time.Hour),
	}
}

func (c *OraCntrlr) Run() {
	c.dbSuccess = false

	c.MetricsChan <- metrics_cntrl.OraMetrics{
		Count:  0,
		Labels: map[string]string{metrics_cntrl.OraTypeLabel: PriorToString(LogPriority)},
	}
	c.MetricsChan <- metrics_cntrl.OraMetrics{
		Count:  0,
		Labels: map[string]string{metrics_cntrl.OraTypeLabel: PriorToString(CheckLogCountPriority)},
	}
	c.MetricsChan <- metrics_cntrl.OraMetrics{
		Count:  0,
		Labels: map[string]string{metrics_cntrl.OraTypeLabel: PriorToString(CheckLogLivetimePriority)},
	}
	c.MetricsChan <- metrics_cntrl.OraMetrics{
		Count:  0,
		Labels: map[string]string{metrics_cntrl.OraTypeLabel: PriorToString(HeartbeatPriority)},
	}
	c.MetricsChan <- metrics_cntrl.OraMetrics{
		Count:  0,
		Labels: map[string]string{metrics_cntrl.OraTypeLabel: errMetricLabel},
	}

	for {
		select {

		case newSettings := <-c.SettsChan:
			if isDbEqual, isStorEqual := c.curSetts.equal(newSettings); !isDbEqual || !isStorEqual {
				c.curSetts = newSettings

				if !isDbEqual {
					if c.dbSuccess {
						c.disconnectFromDb()
					}
					if err := c.connectToDb(); err != nil {
						log.Printf("Fail connect to DB: %v\n", err)
					} else {
						log.Println("Success connect to DB")
					}
				}
				if !isStorEqual {
					c.checkLogCount()
					c.checkLogLivetime()
				}
			}

		case logMsgs := <-c.ReceiveMsgChan:
			c.Lock()
			c.logMsgBuffer = append(c.logMsgBuffer, logMsgs...)
			c.Unlock()
			c.curInsertCount++

			if c.curInsertCount > insertCountCheck {
				c.checkLogCount()
				c.curInsertCount = 0
			}
			c.checkQueryQueue()

		case execErr := <-c.execResultChan:
			if execErr != nil {
				if strings.Contains(execErr.Error(), "database is closed") ||
					strings.Contains(execErr.Error(), "server is not accepting clients") {
					c.disconnectFromDb()
				} else {
					c.canExecute = true
				}
			} else {
				c.canExecute = true
			}
			c.checkQueryQueue()

		case <-c.pingDbTicker.C:
			c.checkHeatrbeat()

		case <-c.logLifetimeTicker.C:
			c.checkLogLivetime()
		}
	}
}

func (c *OraCntrlr) connectToDb() error {
	c.dbSuccess = false

	var errOpen error
	c.db, errOpen = sql.Open("godror", c.curSetts.ConnString())

	if errOpen != nil {
		c.disconnectFromDb()
		return errOpen
	}

	errPing := c.db.Ping()
	if errPing != nil {
		c.disconnectFromDb()
		return errPing
	} else {
		c.dbSuccess = true
		c.canExecute = true

		go c.executeQuery()

		c.checkQueryQueue()
	}
	return nil
}

func (c *OraCntrlr) disconnectFromDb() {
	c.canExecute = false
	c.dbSuccess = false
	c.execTermChan <- struct{}{}
	c.db.Close()
}

func (c *OraCntrlr) checkLogCount() {
	c.queryQueue.Put(queueItem{
		queryText: oraCheckLogCountQuery(onlineLogMaxCount, c.curSetts.LogStoreMaxCount),
		priority:  CheckLogCountPriority,
		countMsg:  1,
	})
	c.checkQueryQueue()
}

func (c *OraCntrlr) checkLogLivetime() {
	oldLogDate := time.Now().AddDate(0, 0, -c.curSetts.LogStoreDays)

	c.queryQueue.Put(queueItem{
		queryText: oraCheckLogLivetimeQuery(oldLogDate.Format(timeFormat)),
		priority:  CheckLogLivetimePriority,
		countMsg:  1,
	})
	c.checkQueryQueue()
}

func (c *OraCntrlr) checkHeatrbeat() {
	if c.queryQueue.Empty() {
		c.RequestMsgChan <- struct{}{}

		c.queryQueue.Put(queueItem{
			queryText: oraHeartbeatQuery(),
			priority:  HeartbeatPriority,
			countMsg:  1,
		})
	}
	c.checkQueryQueue()
}

func (c *OraCntrlr) checkQueryQueue() {
	if c.canExecute {

		if c.queryQueue.Empty() {
			var toQuery []fmtp_log.LogMessage

			if len(c.logMsgBuffer) > 0 {
				c.Lock()
				if len(c.logMsgBuffer) > maxQueryExec {
					toQuery = append(toQuery, c.logMsgBuffer[:maxQueryExec]...)
					c.logMsgBuffer = c.logMsgBuffer[maxQueryExec:]
				} else {
					toQuery = make([]fmtp_log.LogMessage, len(c.logMsgBuffer))
					copy(toQuery, c.logMsgBuffer)
					c.logMsgBuffer = c.logMsgBuffer[:0]
				}
				c.Unlock()

				if len(toQuery) > 0 {
					c.queryQueue.Put(queueItem{
						queryText: oraInsertLogQuery(toQuery...),
						priority:  LogPriority,
						countMsg:  len(toQuery),
					})
				}
			} else {
				c.RequestMsgChan <- struct{}{}
			}
		}

		if !c.queryQueue.Empty() {
			queries, _ := c.queryQueue.Get(1)
			if len(queries) == 1 {
				c.canExecute = false
				c.execQueryChan <- queries[0].(queueItem)
			}
		}
	}
}

func (c *OraCntrlr) executeQuery() {
	for {
		select {

		case queueIt := <-c.execQueryChan:
			_, curErr := c.db.Exec(queueIt.queryText)
			if curErr != nil {
				log.Printf("!!! EXEC err %s\n\n", curErr.Error())
				c.MetricsChan <- metrics_cntrl.OraMetrics{
					Count:  queueIt.countMsg,
					Labels: map[string]string{metrics_cntrl.OraTypeLabel: errMetricLabel},
				}
			} else {
				c.MetricsChan <- metrics_cntrl.OraMetrics{
					Count:  queueIt.countMsg,
					Labels: map[string]string{metrics_cntrl.OraTypeLabel: PriorToString(queueIt.priority)},
				}
			}
			c.execResultChan <- curErr

		case <-c.execTermChan:
			return
		}
	}
}
