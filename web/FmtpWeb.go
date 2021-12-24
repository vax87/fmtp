package web

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// FmtpWeb базования часть web странички
type FmtpWeb struct {
	http.Server
	page PageInterface
	//startTime time.Time
	//totalTime timeInWork
}

var webSrv FmtpWeb

func (s *FmtpWeb) handlerMain(w http.ResponseWriter, r *http.Request) {
	s.updateTime()
	//s.page.setWorkTime(s.totalTime)

	var curMemStat runtime.MemStats
	runtime.ReadMemStats(&curMemStat)

	//var curMemUsage memUsage
	//curMemUsage.fromRuntimeMemStat(curMemStat)

	//s.page.setMemStat(curMemUsage)

	//s.page.setGrCount(runtime.NumGoroutine())

	//s.page.updateLogs()

	if err := s.page.template().ExecuteTemplate(w, "T", s.page); err != nil {
		log.Println("template ExecuteTemplate ERROR")
	}
}

// Initialize web инициализация страницы
func Initialize(handleURL string, netPort int, curPage PageInterface) {
	r := mux.NewRouter()
	r.HandleFunc("/"+handleURL, webSrv.handlerMain)

	webSrv = FmtpWeb{
		Server: http.Server{
			Handler:      r,
			Addr:         fmt.Sprintf(":%d", netPort),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}

	webSrv.page = curPage
	//webSrv.startTime = time.Now().UTC()
	webSrv.page.initialize()
}

// Start начало исполнения
func Start() {

	err := webSrv.ListenAndServe()
	if err != nil {
		fmt.Printf("Listen and serve: %v", err)
	}
}

func (s *FmtpWeb) updateTime() {
	s.page.Lock()
	defer s.page.Unlock()

	// secDuration := uint64(time.Since(s.startTime).Seconds())
	// s.totalTime.Days = strconv.FormatUint(secDuration/(24*60*60), 10)
	// s.totalTime.Hours = strconv.FormatUint((secDuration/(60*60))%24, 10)
	// s.totalTime.Mins = strconv.FormatUint((secDuration/60)%60, 10)
	// s.totalTime.Secs = strconv.FormatUint(secDuration%60, 10)
	// s.totalTime.Now = time.Now().UTC().Format("2006-01-02 15:04:05")
}

// AppendLog добавить сообщение журнала
// func AppendLog(message common.LogMessage) {
// 	webSrv.page.appendLog(appendColorToMsg(message))
// }

// SetVersion задать версию софта
// func SetVersion(vers string) {
// 	webSrv.page.setVersion(vers)
// }

//-------------------------ChannelPage

// SetChannelSetts задание настроек FMTP канала (только для ChannelPage)
// func SetChannelSetts(chSetts ChannelSettsWeb) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChannelPage") {
// 		//webSrv.page.(*ChannelPage).setChannelSettings(chSetts)
// 	}
// }

// SetChiefConn задание состояние подключения к конторллеру (только для ChannelPage)
// func SetChiefConn(chiefConn bool) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChannelPage") {
// 		//webSrv.page.(*ChannelPage).setChiefConn(chiefConn)
// 	}
// }

// SetChannelFMTPState задание FMTP состояние канала (только для ChannelPage)
// func SetChannelFMTPState(chFMTPState string) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChannelPage") {
// 		//webSrv.page.(*ChannelPage).setChannelFMTPState(chFMTPState)
// 	}
// }

// SetTCPState задание состояние TCP транспорта
// func SetTCPState(tcpState bool) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChannelPage") {
// 		//webSrv.page.(*ChannelPage).setTCPState(tcpState)
// 	}
// }

//-------------------------LoggerPage

// SetDbSettings задание настроек подключения в БД (только для LoggerPage)
// func SetDbSettings(dbSetts string) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "LoggerPage") {
// 		//webSrv.page.(*LoggerPage).setDbSettings(dbSetts)
// 	}
// }

// SetDbQueueInfo задание параметров очередди сообщений (только для LoggerPage)
// func SetDbQueueInfo(queueInfo string) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "LoggerPage") {
// 		//webSrv.page.(*LoggerPage).setDbQueueInfo(queueInfo)
// 	}
// }

// SetDbState задание состояния БД (только для LoggerPage)
// func SetDbState(dbState string) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "LoggerPage") {
// 		//webSrv.page.(*LoggerPage).setDbState(dbState)
// 	}
// }

// SetDbLastError последняя ошибка при работе с БД (только для LoggerPage)
// func SetDbLastError(dbError string) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "LoggerPage") {
// 		//webSrv.page.(*LoggerPage).setDbLastError(dbError)
// 	}
// }

// AppendDbWroteCount кол-во записанных логов с начала работы (только для LoggerPage)
// func AppendDbWroteCount(wroteCount int) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "LoggerPage") {
// 		//webSrv.page.(*LoggerPage).appendDbWroteCount(wroteCount)
// 	}
// }

// SetDbCheckCount время проверки кол-ва хранимых логов (только для LoggerPage)
// func SetDbCheckCount(dbCheckTime string) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "LoggerPage") {
// 		//webSrv.page.(*LoggerPage).setDbCheckCount(dbCheckTime)
// 	}
// }

// SetDbCheckLifetime время проверки хранения логов (только для LoggerPage)
// func SetDbCheckLifetime(dbCheckTime string) {
// 	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "LoggerPage") {
// 		//webSrv.page.(*LoggerPage).setDbCheckLifetime(dbCheckTime)
// 	}
// }

//-------------------------ChiefPage

// SetConfigConn задание состояния подключения к конфигуратору
func SetConfigConn(conn bool) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setConfigConn(conn)
	}
}

// SetWSChannelState задание состояния WS сервера каналов
func SetWSChannelState(conn bool) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setWSChannelState(conn)
	}
}

// SetLoggerState задание состояния подключения к логгеру
func SetLoggerState(conn bool) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setLoggerState(conn)
	}
}

// SetAODBState задание состояния  WS сервера AODB провайдера
func SetAODBState(conn bool) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setAODBState(conn)
	}
}

// SetOLDIState задание состояния  WS сервера OLDI провайдера
func SetOLDIState(conn bool) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setOLDIState(conn)
	}
}

// SetChannelVersions задание версии каналов
func SetChannelVersions(vers string) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setChannelVersions(vers)
	}
}

// SetDockerVersion задание версии Docker
func SetDockerVersion(vers string) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setDockerVersion(vers)
	}
}

// SetIPAddresses задание списка IP адресов
func SetIPAddresses(ips string) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setIPAddresses(ips)
	}
}

// SetChannelStates задание состояния каналов
func SetChannelStates(states []ChannelStateWeb) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setChannelStates(states)
	}
}

// SetProviderStates задание состояния провайдеров
func SetProviderStates(states []ProviderStateWeb) {
	if strings.ContainsAny(reflect.TypeOf(webSrv.page).String(), "ChiefPage") {
		webSrv.page.(*ChiefPage).setProviderStates(states)
	}
}

///////////////////////////

// дополняем сообщенияе журнала цветом
// func appendColorToMsg(msg common.LogMessage) logWithColor {
// 	retValue := logWithColor{LogMessage: msg}

// 	switch msg.Severity {
// 	case common.SeverityDebug:
// 		retValue.MsgColor = logDebugColor
// 	case common.SeverityInfo:
// 		retValue.MsgColor = logInfoColor
// 	case common.SeverityWarning:
// 		retValue.MsgColor = logWarningColor
// 	case common.SeverityError:
// 		retValue.MsgColor = logErrorColor
// 	default:
// 		retValue.MsgColor = DefaultColor
// 	}

// 	return retValue
// }
