package ora_cntrl

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/godror/godror"
	"github.com/golang-collections/go-datastructures/queue"

	"fdps/fmtp/fmtp_log"
)

const (
	timeFormat        = "2006-01-02 15:04:05"
	insertCountCheck  = 10000 // кол-во запросов INSERT в БД, после чего следует проверить кол-во хранимых сообщений
	onlineLogMaxCount = 1000  // кол-во хранимых логов в таблице онлайн сообщений
	maxQueryExec      = 100   //кол-во логов, записываемых за раз
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type queueItem struct {
	queryText string
	priority  int
}

func (qi queueItem) Compare(other queue.Item) int {
	if cast, ok := other.(queueItem); ok {
		if qi.priority < cast.priority {
			return 1
		}
	}
	return -1
}

const (
	LogPriority = iota
	CheckLogCountPriority
	CheckLogLivetimePriority
	HeartbeatPriority
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// контроллер, выполняющий запись логов в БД
type OraLoggerController struct {
	sync.Mutex

	SettsChan      chan OraCntrlSettings      // канал приема новых настроек контроллера
	ReceiveMsgChan chan []fmtp_log.LogMessage // канал приема новых сообщений
	RequestMsgChan chan struct{}

	curSetts OraCntrlSettings

	logMsgBuffer []fmtp_log.LogMessage // очередь логов
	queryQueue   *queue.PriorityQueue  // очередь сообщений для записи в БД

	db         *sql.DB // объект БД
	dbSuccess  bool    // успешность подключения к БД
	canExecute bool    // возможность выполнять запросы к БД

	execQueryChan  chan string   // канал для передачи запросов записи логов на выполнение
	execResultChan chan error    // канал для передачи результатов выполнения
	execTermChan   chan struct{} // канал для завершения подпрограммы выполнения запросов

	logLifetimeTicker *time.Ticker // тикер проверки времени жизни логов
	pingDbTicker      *time.Ticker // тикер пинга БД
	connectDbTicker   *time.Ticker // тикер подключения к БД
	curInsertCount    uint64       // кол-во выполнненых запросов INSERT (для проверки кол-ва хранимых логов)

	countRecvMsg             int64
	countRecvMsgPrevSecond   int64
	countQueryExec           int
	countQueryExecPrevSecond int
}

// конструктор
func NewOraController() *OraLoggerController {
	return new(OraLoggerController).Init()
}

// инициализация параметрами по умолчанию
func (rlc *OraLoggerController) Init() *OraLoggerController {
	rlc.ReceiveMsgChan = make(chan []fmtp_log.LogMessage, 1024)
	rlc.SettsChan = make(chan OraCntrlSettings, 1)
	rlc.RequestMsgChan = make(chan struct{}, 1)

	rlc.queryQueue = queue.NewPriorityQueue(10000)

	rlc.canExecute = false

	rlc.execQueryChan = make(chan string, 1)
	rlc.execResultChan = make(chan error, 1)
	rlc.execTermChan = make(chan struct{}, 1)

	rlc.pingDbTicker = time.NewTicker(10 * time.Second)
	rlc.connectDbTicker = time.NewTicker(10 * time.Second)
	rlc.logLifetimeTicker = time.NewTicker(12 * time.Hour)

	return rlc
}

func (rlc *OraLoggerController) Run() {
	rlc.dbSuccess = false

	metricTimer := time.NewTicker(1 * time.Second)

	for {
		select {

		case newSettings := <-rlc.SettsChan:
			if isDbEqual, isStorEqual := rlc.curSetts.equal(newSettings); !isDbEqual || !isStorEqual {
				rlc.curSetts = newSettings

				if !isDbEqual {
					if rlc.dbSuccess {
						rlc.disconnectFromDb()
					}
				}
				if !isStorEqual {
					rlc.checkLogCount()
					rlc.checkLogLivetime()
				}
			}

		// получено новое сообщение
		case logMsgs := <-rlc.ReceiveMsgChan:
			rlc.countRecvMsg++

			rlc.Lock()
			rlc.logMsgBuffer = append(rlc.logMsgBuffer, logMsgs...)
			rlc.Unlock()
			rlc.curInsertCount++

			if rlc.curInsertCount > insertCountCheck {
				rlc.checkLogCount()
				rlc.curInsertCount = 0
			}
			rlc.checkQueryQueue()

		// пришел результат выполнения запроса
		case execErr := <-rlc.execResultChan:
			if execErr != nil {
				if strings.Contains(execErr.Error(), "database is closed") ||
					strings.Contains(execErr.Error(), "server is not accepting clients") {
					rlc.disconnectFromDb()
				} else {
					rlc.canExecute = true
				}
			} else {
				rlc.canExecute = true
			}

			rlc.checkQueryQueue()

		// сработал тикер пинга БД
		case <-rlc.pingDbTicker.C:
			rlc.checkHeatrbeat()

		// сработал тикер подключения к БД
		case <-rlc.connectDbTicker.C:
			_ = rlc.connectToDb()

		// сработал тикер проверки времени жизни логов
		case <-rlc.logLifetimeTicker.C:
			rlc.checkLogLivetime()

		case <-metricTimer.C:
			//diffCountRecvPerSecond := rlc.countRecvMsg - rlc.countRecvMsgPrevSecond
			rlc.countRecvMsgPrevSecond = rlc.countRecvMsg
			//logger.SetDebugParam("Recv за секунду", strconv.FormatInt(diffCountRecvPerSecond, 10), logger.DebugColor)
			//logger.SetDebugParam("Recv всего ", strconv.FormatInt(rlc.countRecvMsg, 10), logger.DebugColor)

			//diffCountQueryPerSecond := rlc.countQueryExec - rlc.countQueryExecPrevSecond
			rlc.countQueryExecPrevSecond = rlc.countQueryExec
			//.SetDebugParam("Exec за секунду", strconv.Itoa(diffCountQueryPerSecond), logger.DebugColor)
			//logger.SetDebugParam("Exec всего ", strconv.Itoa(rlc.countQueryExec), logger.DebugColor)
		}
	}
}

func (rlc *OraLoggerController) connectToDb() error {
	rlc.dbSuccess = false

	// user/pass@(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=hostname)(PORT=port)))(CONNECT_DATA=(SERVICE_NAME=sn)))
	var errOpen error
	rlc.db, errOpen = sql.Open("godror", rlc.curSetts.ConnString())

	if errOpen != nil {
		rlc.disconnectFromDb()
		return errOpen
	}

	errPing := rlc.db.Ping()
	if errPing != nil {
		rlc.disconnectFromDb()
		return errPing
	} else {
		rlc.dbSuccess = true
		rlc.canExecute = true

		go rlc.executeQuery()

		rlc.checkQueryQueue()
	}
	return nil
}

func (rlc *OraLoggerController) disconnectFromDb() {
	rlc.canExecute = false
	rlc.dbSuccess = false
	rlc.execTermChan <- struct{}{}
	rlc.db.Close()
}

func (rlc *OraLoggerController) checkLogCount() {
	rlc.queryQueue.Put(queueItem{
		queryText: oraCheckLogCountQuery(onlineLogMaxCount, rlc.curSetts.LogStoreMaxCount),
		priority:  CheckLogCountPriority,
	})
	rlc.checkQueryQueue()
}

func (rlc *OraLoggerController) checkLogLivetime() {
	oldLogDate := time.Now().AddDate(0, 0, -rlc.curSetts.LogStoreDays)

	rlc.queryQueue.Put(queueItem{
		queryText: oraCheckLogLivetimeQuery(oldLogDate.Format(timeFormat)),
		priority:  CheckLogLivetimePriority,
	})
	rlc.checkQueryQueue()
}

func (rlc *OraLoggerController) checkHeatrbeat() {
	if rlc.queryQueue.Empty() {
		rlc.queryQueue.Put(queueItem{
			queryText: oraHeartbeatQuery(),
			priority:  HeartbeatPriority,
		})
		rlc.checkQueryQueue()
	}
}

func (rlc *OraLoggerController) checkQueryQueue() {

	if rlc.canExecute {

		if rlc.queryQueue.Empty() {
			var toQuery []fmtp_log.LogMessage

			if len(rlc.logMsgBuffer) > 0 {
				log.Printf("len(rlc.logMsgBuffer) > 0 : %d", len(rlc.logMsgBuffer))
				rlc.Lock()
				if len(rlc.logMsgBuffer) > maxQueryExec {
					toQuery = append(toQuery, rlc.logMsgBuffer[:maxQueryExec]...)
					log.Printf("\t\tbefore len(rlc.logMsgBuffer) : %d", len(rlc.logMsgBuffer))
					rlc.logMsgBuffer = rlc.logMsgBuffer[maxQueryExec:]
					log.Printf("\t\tafter len(rlc.logMsgBuffer) : %d", len(rlc.logMsgBuffer))

					log.Printf("\tlen(rlc.toQuery) > 0 : %d", len(toQuery))
				} else {
					toQuery = make([]fmtp_log.LogMessage, len(rlc.logMsgBuffer))
					copy(toQuery, rlc.logMsgBuffer)
					log.Printf("\t\tbefore len(rlc.logMsgBuffer) : %d", len(rlc.logMsgBuffer))
					rlc.logMsgBuffer = rlc.logMsgBuffer[:0]
					log.Printf("\t\tafter len(rlc.logMsgBuffer) : %d", len(rlc.logMsgBuffer))

					log.Printf("\tlen(rlc.toQuery) > 0 : %d", len(toQuery))
				}
				rlc.Unlock()

				if len(toQuery) > 0 {
					rlc.queryQueue.Put(queueItem{
						queryText: oraInsertLogQuery(toQuery...),
						priority:  LogPriority,
					})
				}
			} else {
				rlc.RequestMsgChan <- struct{}{}
			}
		}

		if !rlc.queryQueue.Empty() {
			queries, _ := rlc.queryQueue.Get(1)
			if len(queries) == 1 {
				rlc.canExecute = false
				rlc.execQueryChan <- queries[0].(queueItem).queryText
			}
		}
	}
}

// функция горутины, в которой будут выполняться запросы к БД
func (rlc *OraLoggerController) executeQuery() {
	for {
		select {
		// пришел запрос добавления сообщения логов
		case queryText := <-rlc.execQueryChan:
			log.Printf("EXEC %d %s\n\n", len(queryText), queryText)

			_, curErr := rlc.db.Exec(queryText)
			if curErr != nil {
				log.Printf("!!! EXEC err %s\n\n", curErr.Error())
			}

			rlc.countQueryExec++
			rlc.execResultChan <- curErr

		// необходимо завершить горутину
		case <-rlc.execTermChan:
			return
		}
	}
}
