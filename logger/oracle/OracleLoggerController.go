package oracle

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/phf/go-queue/queue"
	_ "gopkg.in/goracle.v2"

	"fdps/fmtp/chief_logger"
	"fdps/fmtp/logger/common"
	"fdps/fmtp/web"
)

// контроллер, выполняющий запись логов в БД
type OracleLoggerController struct {
	SettingsChan chan OracleLoggerSettings            // канал приема новых настроек контроллера
	MessChan     chan common.LogMessage               // канал приема новых сообщений
	StateChan    chan chief_logger.LoggerStateMessage // канал отправки состояния подключения к БД

	currentSettings OracleLoggerSettings
	closeChan       chan struct{} // канал для передачи сигнала закрытия подключения к БД.

	logQueue queue.Queue // очередь сообщений для записи в БД

	needCheckLogCount    bool // необходимость проверить кол-во хранимых логов
	needCheckLogLifetime bool // необходимость проверить логи на время хранения

	db        *sql.DB // объект БД
	dbSuccess bool    // успешность подключения к БД

	canExecute bool // возможность выполнять запросы к БД

	execLogQueryChan   chan string   // канал для передачи запросов записи логов на выполнение
	execCountQueryChan chan struct{} // канал для передачи запросов проверки кол-ва логов на выполнение
	execResultChan     chan error    // канал для передачи результатов выполнения
	execTermChan       chan struct{} // канал для завершения подпрограммы выполнения запросов

	logLifetimeTicker *time.Ticker // тикер проверки времени жизни логов
	pingDbTicker      *time.Ticker // тикер пинга БД
	connectDbTicker   *time.Ticker // тикер подключения к БД

	curInsertCount uint64 // кол-во выполнненых запросов INSERT (для проверки кол-ва хранимых логов)
	curDbErrorText string // текст текущей ошибки при работе с БД
	lastQueryText  string // текст последнего запроса (при возникновении ошибки при упешном подключении выполнить еще раз)

	loggerVersion string // версия софта, считанная из переменной окружения
}

// конструктор
func NewOracleController() *OracleLoggerController {
	return new(OracleLoggerController).Init()
}

// инициализация параметрами по умолчанию
func (rlc *OracleLoggerController) Init() *OracleLoggerController {
	rlc.MessChan = make(chan common.LogMessage, 1024)
	rlc.SettingsChan = make(chan OracleLoggerSettings)
	rlc.StateChan = make(chan chief_logger.LoggerStateMessage)

	rlc.canExecute = false
	rlc.needCheckLogCount = true
	rlc.needCheckLogLifetime = true

	rlc.execLogQueryChan = make(chan string, 1)
	rlc.execCountQueryChan = make(chan struct{}, 1)
	rlc.execResultChan = make(chan error, 1)
	rlc.execTermChan = make(chan struct{}, 1)

	rlc.pingDbTicker = time.NewTicker(3 * time.Second)
	rlc.connectDbTicker = time.NewTicker(10 * time.Second)
	rlc.logLifetimeTicker = time.NewTicker(12 * time.Hour)

	rlc.loggerVersion = os.Getenv("LoggerVersion")
	if rlc.loggerVersion == "" {
		rlc.loggerVersion = "???"
	}

	return rlc
}

func (rlc *OracleLoggerController) Run() {
	rlc.dbSuccess = false

	for {
		select {
		case newSettings := <-rlc.SettingsChan:
			web.SetDbQueueInfo(fmt.Sprintf("%d / %d", rlc.logQueue.Len(), logContainerSize))
			web.SetDbSettings(fmt.Sprintf("%+v", newSettings))

			if isDbEqual, isStorEqual := rlc.currentSettings.equal(newSettings); !isDbEqual || !isStorEqual {
				rlc.currentSettings = newSettings

				if !isDbEqual {
					if rlc.dbSuccess {
						rlc.disconnectFromDb()
					}
					//if rlc.currentSettings.NeedWork {
					curErr := rlc.connectToDb()
					if curErr != nil {
						rlc.StateChan <- chief_logger.CreateLoggerStateMsg(curErr, rlc.loggerVersion)
					}
					//}
				}
				if !isStorEqual {
					rlc.needCheckLogCount = true
					rlc.needCheckLogLifetime = true
					rlc.checkQueryQueue()
				}
			}

			// получено новое сообщение
		case newMessage := <-rlc.MessChan:
			web.SetDbQueueInfo(fmt.Sprintf("%d / %d", rlc.logQueue.Len()+1, logContainerSize))
			if rlc.logQueue.Len() < logContainerSize {
				rlc.logQueue.PushBack(newMessage)
				rlc.checkQueryQueue()
			}

			// пришел результат выполнения запроса из горутины
		case execErr := <-rlc.execResultChan:
			if execErr != nil {
				log.Println("Error executing query ", execErr)
				if strings.Contains(execErr.Error(), "database is closed") ||
					strings.Contains(execErr.Error(), "server is not accepting clients") {
					rlc.disconnectFromDb()
				} else {
					rlc.canExecute = true
				}
				rlc.StateChan <- chief_logger.CreateLoggerStateMsg(execErr, rlc.loggerVersion)
			} else {
				rlc.lastQueryText = ""
				rlc.canExecute = true
			}

			rlc.checkQueryQueue()

			// сработал тикер пингера БД
		case <-rlc.pingDbTicker.C:
			if rlc.dbSuccess { //&& rlc.currentSettings.NeedWork {
				curErr := rlc.heartbeat()

				if curErr != nil {
					rlc.StateChan <- chief_logger.CreateLoggerStateMsg(curErr, rlc.loggerVersion)
				} else {
					rlc.StateChan <- chief_logger.CreateLoggerStateMsg(nil, rlc.loggerVersion)
				}
			}

			// сработал тикер подключения к БД
		case <-rlc.connectDbTicker.C:
			if !rlc.dbSuccess { //&& rlc.currentSettings.NeedWork {
				curErr := rlc.connectToDb()
				if curErr != nil {
					rlc.StateChan <- chief_logger.CreateLoggerStateMsg(curErr, rlc.loggerVersion)
				} else {
					rlc.StateChan <- chief_logger.CreateLoggerStateMsg(nil, rlc.loggerVersion)
				}
			}

			// сработал тикер проверки времени жизни логов
		case <-rlc.logLifetimeTicker.C:
			rlc.needCheckLogLifetime = true
		}
	}
}

func (rlc *OracleLoggerController) connectToDb() error {

	rlc.dbSuccess = false

	// user/pass@(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=hostname)(PORT=port)))(CONNECT_DATA=(SERVICE_NAME=sn)))
	var errOpen error
	var connString string = rlc.currentSettings.UserName + "/" +
		rlc.currentSettings.Password + "@" +
		"(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=" + rlc.currentSettings.Hostname +
		")(PORT=" + strconv.Itoa(rlc.currentSettings.Port) +
		")))(CONNECT_DATA=(SERVICE_NAME=" + rlc.currentSettings.ServiceName + ")))"

	rlc.db, errOpen = sql.Open("goracle", connString)

	if errOpen != nil {
		log.Println("Error opening database: ", errOpen)

		rlc.disconnectFromDb()
		return errOpen
	}

	errPing := rlc.db.Ping()
	if errPing != nil {
		log.Println("Error ping database: ", errPing)
		rlc.disconnectFromDb()
		return errPing
	} else {
		log.Println("Database is opened")
		rlc.initDbStructure()

		rlc.dbSuccess = true
		rlc.canExecute = true

		go rlc.executeQuery()

		rlc.checkQueryQueue()
	}
	return nil
}

func (rlc *OracleLoggerController) heartbeat() error {
	var heartbeatInt int
	errHbt := rlc.db.QueryRow("SELECT 7 FROM dual").Scan(&heartbeatInt)
	if errHbt != nil {
		log.Println("Database heartbeat error: ", errHbt)
		rlc.disconnectFromDb()
	}
	return errHbt
}

func (rlc *OracleLoggerController) disconnectFromDb() {
	rlc.canExecute = false
	rlc.dbSuccess = false
	rlc.db.Close()
}

func (rlc *OracleLoggerController) checkQueryQueue() {
	if rlc.canExecute {
		rlc.canExecute = false

		// необходимо проверить кол-во хранимых логов
		if rlc.needCheckLogCount {
			rlc.needCheckLogCount = false

			rlc.execCountQueryChan <- struct{}{}
			return
		}

		// необходимо проверить время жизни логов
		if rlc.needCheckLogLifetime {
			rlc.needCheckLogLifetime = false

			oldLogDate := time.Now().AddDate(0, 0, -rlc.currentSettings.LogStoreDays)
			web.SetDbCheckLifetime(time.Now().UTC().Format("2006-01-02 15:04:05.000"))
			rlc.execLogQueryChan <- oraCheckLogLifetimeQuery(oldLogDate.Format("2006-01-02 15:04:05"))
			return
		}

		var curQueryText string // текст текущего запроса

		if rlc.lastQueryText != "" {
			curQueryText = rlc.lastQueryText
		} else {
			var insetrCnt int
			for insetrCnt = 0; insetrCnt < oraMaxInsertCount; insetrCnt++ {
				if rlc.logQueue.Len() > 0 {
					oraInsertLogBeginQuery(&curQueryText, common.LogMessage(rlc.logQueue.PopFront().(common.LogMessage)))
					rlc.curInsertCount++
				} else {
					break
				}
			}
			web.AppendDbWroteCount(insetrCnt)
		}

		web.SetDbQueueInfo(fmt.Sprintf("%d / %d", rlc.logQueue.Len(), logContainerSize))

		if curQueryText != "" {
			if rlc.lastQueryText == "" {
				oraInsertLogEndQuery(&curQueryText)
			}
			rlc.lastQueryText = curQueryText

			rlc.execLogQueryChan <- curQueryText

			if rlc.curInsertCount > insertCountCheck {
				rlc.needCheckLogCount = true
				rlc.curInsertCount = 0
			}
		} else {
			rlc.canExecute = true
		}
	}
}

// функция горутины, в которой будут выполняться запросы к БД
func (rlc *OracleLoggerController) executeQuery() {
	for {
		select {
		// пришел запрос добавления сообщения логов
		case logQueryText := <-rlc.execLogQueryChan:
			_, curErr := rlc.db.Exec(logQueryText)
			rlc.execResultChan <- curErr

		// пришел запрос проверки кол-ва хранимых логов
		case <-rlc.execCountQueryChan:
			web.SetDbCheckCount(time.Now().UTC().Format("2006-01-02 15:04:05.000"))
			_, curErr := rlc.db.Exec(oraCheckLogCountQuery(onlineLogMaxCount, rlc.currentSettings.LogStoreMaxCount))
			rlc.execResultChan <- curErr

		// необходимо завершить горутину
		case <-rlc.execTermChan:
			return
		}
	}
}

// инициализация БД (выполнение запросов создания таблиц, представлений)
func (rlc *OracleLoggerController) initDbStructure() {
	var curStmnt string // текст текущего запроса
	var curError error  // последняя ошибка

	// таблица с сообщениями для долговременного хранения.
	curStmnt = oraCreateLogTableQuery(storageLogTableName)
	_, curError = rlc.db.Exec(curStmnt)
	if curError != nil {
		log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
	} else {
		// создание первичного ключа таблицы с сообщениями для долговременного хранения.
		curStmnt = oraCreatePrimaryKeyQuery(storageLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}

		// последовательность для автоинкремента id таблицы storage_log.
		curStmnt = oraCreateSequenceQuery(storageLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}

		// триггер для автоинкремента id таблицы storage_log.
		curStmnt = oraCreateTriggerQuery(storageLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}

		// представление с онлайн сообщениями.
		curStmnt = oraCreateLogViewQuery(storageLogViewName, storageLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}
	}

	// таблица с онлайн сообщениями.
	curStmnt = oraCreateLogTableQuery(onlineLogTableName)
	_, curError = rlc.db.Exec(curStmnt)
	if curError != nil {
		log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
	} else {
		// создание первичного ключа таблицы с онлайн сообщениями .
		curStmnt = oraCreatePrimaryKeyQuery(onlineLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}

		// последовательность для автоинкремента id таблицы online_log.
		curStmnt = oraCreateSequenceQuery(onlineLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}

		// триггер для автоинкремента id таблицы online_log.
		curStmnt = oraCreateTriggerQuery(onlineLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}
		// представление с сообщениями для долговременного хранения.
		curStmnt = oraCreateLogViewQuery(onlineLogViewName, onlineLogTableName)
		_, curError = rlc.db.Exec(curStmnt)
		if curError != nil {
			log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
		}
	}

	// заголовок пакета для удаления старых и лишних логов.
	curStmnt = oraPackageQuery()
	_, curError = rlc.db.Exec(curStmnt)
	if curError != nil {
		log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
	}

	// тело пакета для удаления старых и лишних логов.
	curStmnt = oraPackageBodyQuery()
	_, curError = rlc.db.Exec(curStmnt)
	if curError != nil {
		log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
	}

	// заголовок пакета для удаления старых и лишних логов.
	curStmnt = "COMMIT"
	_, curError = rlc.db.Exec(curStmnt)
	if curError != nil {
		log.Println("Error execute statement: ", curStmnt, "\t Error: ", curError)
	}
}
