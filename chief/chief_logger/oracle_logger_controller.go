package chief_logger

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/godror/godror"
	"github.com/golang-collections/go-datastructures/queue"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_state"

	"fdps/fmtp/fmtp_logger"
)

const (
	timeFormat        = "2006-01-02 15:04:05"
	insertCountCheck  = 1000 // кол-во запросов INSERT в БД, после чего следует проверить кол-во хранимых сообщений
	onlineLogMaxCount = 1000 // кол-во хранимых логов в таблице онлайн сообщений
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func createLoggerState(dbErr error) chief_state.LoggerState {
	retValue := chief_state.LoggerState{
		LoggerConnected:   chief_state.StateOk,
		LoggerDbConnected: chief_state.StateOk,
		LoggerVersion:     "1.0.0",
	}
	if dbErr != nil {
		retValue.LoggerDbConnected = chief_state.StateError
		retValue.LoggerDbError = dbErr.Error()
	}
	return retValue
}

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
	StatesPriority
	HeartbeatPriority
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// контроллер, выполняющий запись логов в БД
type OracleLoggerController struct {
	SettingsChan chan OracleLoggerSettings    // канал приема новых настроек контроллера
	MessChan     chan fmtp_logger.LogMessage  // канал приема новых сообщений
	StateChan    chan chief_state.LoggerState // канал отправки состояния подключения к БД

	currentSettings OracleLoggerSettings

	queryQueue *queue.PriorityQueue // очередь сообщений для записи в БД

	db         *sql.DB // объект БД
	dbSuccess  bool    // успешность подключения к БД
	canExecute bool    // возможность выполнять запросы к БД

	execQueryChan  chan string   // канал для передачи запросов записи логов на выполнение
	execResultChan chan error    // канал для передачи результатов выполнения
	execTermChan   chan struct{} // канал для завершения подпрограммы выполнения запросов

	logLifetimeTicker *time.Ticker // тикер проверки времени жизни логов
	pingDbTicker      *time.Ticker // тикер пинга БД
	connectDbTicker   *time.Ticker // тикер подключения к БД

	lastChannelStates []channel_state.ChannelState
	checkStatesTckr   *time.Ticker // тикер записи состояния каналов в БД

	curInsertCount  uint64 // кол-во выполнненых запросов INSERT (для проверки кол-ва хранимых логов)
	writeStatesToDb bool
}

// конструктор
func NewOracleController() *OracleLoggerController {
	return new(OracleLoggerController).Init()
}

// инициализация параметрами по умолчанию
func (rlc *OracleLoggerController) Init() *OracleLoggerController {
	rlc.MessChan = make(chan fmtp_logger.LogMessage, 1024)
	rlc.SettingsChan = make(chan OracleLoggerSettings)
	rlc.StateChan = make(chan chief_state.LoggerState, 1)

	rlc.queryQueue = queue.NewPriorityQueue(10000)

	rlc.canExecute = false

	rlc.execQueryChan = make(chan string, 1)
	rlc.execResultChan = make(chan error, 1)
	rlc.execTermChan = make(chan struct{}, 1)

	rlc.pingDbTicker = time.NewTicker(10 * time.Second)
	rlc.connectDbTicker = time.NewTicker(10 * time.Second)
	rlc.logLifetimeTicker = time.NewTicker(12 * time.Hour)
	rlc.checkStatesTckr = time.NewTicker(5 * time.Second)

	return rlc
}

func (rlc *OracleLoggerController) Run() {
	rlc.dbSuccess = false

	for {
		select {
		case newSettings := <-rlc.SettingsChan:

			if isDbEqual, isStorEqual := rlc.currentSettings.equal(newSettings); !isDbEqual || !isStorEqual {
				rlc.currentSettings = newSettings

				if !isDbEqual {
					if rlc.dbSuccess {
						rlc.disconnectFromDb()
					}
					rlc.StateChan <- createLoggerState(rlc.connectToDb())
				}
				if !isStorEqual {
					rlc.checkLogCount()
					rlc.checkLogLivetime()
				}
			}

		// получено новое сообщение
		case logMsg := <-rlc.MessChan:
			// проверяем длину текста (max 2000)
			if len(logMsg.Text) > maxTextLen {
				logMsg.Text = logMsg.Text[:maxTextLen]
			}

			rlc.queryQueue.Put(queueItem{
				queryText: oraInsertLogQuery(logMsg),
				priority:  LogPriority,
			})
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

				rlc.StateChan <- createLoggerState(execErr)
			} else {
				rlc.canExecute = true
			}

			rlc.checkQueryQueue()

		// сработал тикер пинга БД
		case <-rlc.pingDbTicker.C:
			rlc.checkHeatrbeat()

		// сработал тикер подключения к БД
		case <-rlc.connectDbTicker.C:
			if !rlc.dbSuccess {
				rlc.StateChan <- createLoggerState(rlc.connectToDb())
			}

		// сработал тикер проверки времени жизни логов
		case <-rlc.logLifetimeTicker.C:
			rlc.checkLogLivetime()

		// тикер проверки состояния каналов
		case <-rlc.checkStatesTckr.C:
			if rlc.writeStatesToDb {
				if !channel_state.ChannelStatesEqual(rlc.lastChannelStates, chief_state.CommonChiefState.ChannelStates) {
					rlc.lastChannelStates = chief_state.CommonChiefState.ChannelStates
					for _, stateVal := range rlc.lastChannelStates {
						rlc.queryQueue.Put(queueItem{
							queryText: oraUpdateChannelStateQuery(stateVal),
							priority:  StatesPriority,
						})
					}
					rlc.checkQueryQueue()
				}
			}
		}
	}
}

func (rlc *OracleLoggerController) connectToDb() error {
	rlc.dbSuccess = false

	// user/pass@(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=hostname)(PORT=port)))(CONNECT_DATA=(SERVICE_NAME=sn)))
	var errOpen error
	rlc.db, errOpen = sql.Open("godror", rlc.currentSettings.ConnString())

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

func (rlc *OracleLoggerController) disconnectFromDb() {
	rlc.canExecute = false
	rlc.dbSuccess = false
	rlc.execTermChan <- struct{}{}
	rlc.db.Close()
}

func (rlc *OracleLoggerController) checkLogCount() {
	rlc.queryQueue.Put(queueItem{
		queryText: oraCheckLogCountQuery(onlineLogMaxCount, rlc.currentSettings.LogStoreMaxCount),
		priority:  CheckLogCountPriority,
	})
	rlc.checkQueryQueue()
}

func (rlc *OracleLoggerController) checkLogLivetime() {
	oldLogDate := time.Now().AddDate(0, 0, -rlc.currentSettings.LogStoreDays)

	rlc.queryQueue.Put(queueItem{
		queryText: oraCheckLogLivetimeQuery(oldLogDate.Format(timeFormat)),
		priority:  CheckLogLivetimePriority,
	})
	rlc.checkQueryQueue()
}

func (rlc *OracleLoggerController) checkHeatrbeat() {
	if rlc.queryQueue.Empty() {
		rlc.queryQueue.Put(queueItem{
			queryText: oraHeartbeatQuery(),
			priority:  HeartbeatPriority,
		})
		rlc.checkQueryQueue()
	}
}

func (rlc *OracleLoggerController) checkQueryQueue() {
	if rlc.canExecute {
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
func (rlc *OracleLoggerController) executeQuery() {
	for {
		select {
		// пришел запрос добавления сообщения логов
		case queryText := <-rlc.execQueryChan:
			_, curErr := rlc.db.Exec(queryText)
			rlc.execResultChan <- curErr

		// необходимо завершить горутину
		case <-rlc.execTermChan:
			return
		}
	}
}

func (rlc *OracleLoggerController) SetWriteStatesToDb(writeState bool) {
	rlc.writeStatesToDb = writeState
}
