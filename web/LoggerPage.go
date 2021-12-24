package web

// import (
// 	"container/list"
// 	"html/template"
// 	"log"
// 	"sync"
// )

// // LoggerPage web страничка логгера
// type LoggerPage struct {
// 	sync.RWMutex
// 	templ     *template.Template
// 	Title     string
// 	TotalTime timeInWork
// 	Lr        []*logWithColor
// 	logList   *list.List

// 	ReceiveLogCount uint64 // кол-во принятых логов

// 	DbParameters        string // параметры подключения к БД
// 	DbState             string // состояние подключения к БД
// 	DbConnColor         string // цвет состояния подкобчения к БД
// 	DbLastError         string // тест последней ошибки БД
// 	DbWroteCount        uint64 // кол-во логов, записанных в БД с момента старта
// 	DbLogQueueInfo      string // размер очереди логов / max размер очереди
// 	DbCheckCountTime    string // время проверки кол-ва хранимых логов
// 	DbCheckLifetimeTime string // время проверки времени хранения логов

// 	ConnectedClients []string // подключенные клиентыS

// 	GoroutineCount int      // кол-во goroutines
// 	MemStat        memUsage // параметры использования памяти
// 	SoftVersion    string   // версия софта
// }

// func (m *LoggerPage) initialize() {
// 	m.Lock()
// 	defer m.Unlock()

// 	m.logList = list.New()

// 	var err error
// 	if m.templ, err = template.New("MainPage").Parse(mainPageTemplate); err != nil {
// 		log.Println("template Parse ERROR")
// 		return
// 	}

// 	m.Title = "FMTP Logger"

// 	m.Lr = make([]*logWithColor, logSizeMax, logSizeMax)

// 	for i := 0; i < logSizeMax; i++ {
// 		m.logList.PushFront(&logWithColor{MsgColor: DefaultColor})
// 	}
// }

// func (m *LoggerPage) setGrCount(curGrCount int) {
// 	m.GoroutineCount = curGrCount
// }

// func (m *LoggerPage) setMemStat(curMem memUsage) {
// 	m.MemStat = curMem
// }

// // задать версию софта
// func (m *LoggerPage) setVersion(vers string) {
// 	m.SoftVersion = vers
// }

// func (m *LoggerPage) setWorkTime(curTime timeInWork) {
// 	m.TotalTime = curTime
// }

// func (m *LoggerPage) template() *template.Template {
// 	return m.templ
// }

// func (m *LoggerPage) appendLog(message logWithColor) {
// 	m.Lock()
// 	defer m.Unlock()

// 	m.logList.PushFront(&message)

// 	for m.logList.Len() > logSizeMax {
// 		m.logList.Remove(m.logList.Back())
// 	}

// 	m.ReceiveLogCount++
// }

// func (m *LoggerPage) setDbSettings(dbSetts string) {
// 	m.DbParameters = dbSetts
// }

// func (m *LoggerPage) setDbQueueInfo(queueInfo string) {
// 	m.DbLogQueueInfo = queueInfo
// }

// func (m *LoggerPage) setDbState(dbState string) {
// 	m.DbState = dbState
// 	if m.DbState == "ok" {
// 		m.DbConnColor = OkColor
// 	} else {
// 		m.DbConnColor = ErrorColor
// 	}
// }

// func (m *LoggerPage) setDbLastError(dbError string) {
// 	m.DbLastError = dbError
// }

// func (m *LoggerPage) appendDbWroteCount(wroteCount int) {
// 	m.DbWroteCount += uint64(wroteCount)
// }

// func (m *LoggerPage) setDbCheckCount(dbCheckTime string) {
// 	m.DbCheckCountTime = dbCheckTime
// }

// func (m *LoggerPage) setDbCheckLifetime(dbCheckTime string) {
// 	m.DbCheckLifetimeTime = dbCheckTime
// }

// func (m *LoggerPage) updateLogs() {
// 	for e, i := m.logList.Front(), 0; e != nil && i < logSizeMax; e, i = e.Next(), i+1 {
// 		lr, ok := e.Value.(*logWithColor)
// 		if ok {
// 			m.Lr[i] = lr
// 		}
// 	}
// }

// var mainPageTemplate = `{{define "T"}}
// <!DOCTYPE html>
// <html lang="en">
// <head>
//     <meta charset="UTF-8">
//     <title>{{.Title}}</title>
// <script>
// //    window.onload = function() {
// //        setTimeout(function () {
// //            location.reload()
// //        }, 1000);
// //     };
// </script>
// </head>
// <body>

// <font size="3" face="verdana" color="black">
// 	<table width="100%" border="1" cellspacing="0" cellpadding="4">
// 		<colgroup>
// 			<col style = "width: 10%;">
// 			<col style = "width: 10%;">
// 			<col style = "width: 10%;">
// 			<col style = "width: 10%;">
// 			<col style = "width: 5%;">
// 			<col style = "width: 5%;">
// 			<col style = "width: 7%;">
// 			<col style = "width: 7%;">
// 		</colgroup>
// 		<tr>
// 			<td>Состояние подключенияк БД:</td>
// 			<td bgcolor="{{.DbConnColor}}">{{.DbState}}</td>
// 			<td>Текст последней ошибки:</td>
// 			<td>{{.DbLastError}}</td>
// 			<td>В работе:</td>
// 			<td>{{with .TotalTime}}{{.Days}} дн. {{.Hours}} ч. {{.Mins}} м. {{.Secs}} с.{{end}}</td>
// 			<td>Сейчас:</td>
// 			<td>{{with .TotalTime}}{{.Now}}(UTC){{end}}</td>
// 		</tr>
// 		<tr>
// 			<td>Очередь записи (размер / max размер):</td>
// 			<td>{{.DbLogQueueInfo}}</td>
// 			<td>Кол-во принятых логов с начала работы:</td>
// 			<td>{{.ReceiveLogCount}}</td>
// 			<td>Кол-во goroutine:</td>
// 			<td>{{.GoroutineCount}}</td>
// 			<td>Кол-во вызовов GC:</td>
// 			<td>{{with .MemStat}} {{.NumGC}} {{end}}</td>
// 		</tr>
// 		<tr>
// 			<td>Время последней проверки кол-ва хранимых логов (UTC):</td>
// 			<td>{{.DbCheckCountTime}}</td>
// 			<td>Кол-во записанных логов с начала работы:</td>
// 			<td>{{.DbWroteCount}}</td>
// 			<td>Память (Sys):</td>
// 			<td>{{with .MemStat}} {{.Sys}} MB {{end}}</td>
// 			<td>Кол-во объектов в памяти:</td>
// 			<td>{{with .MemStat}} {{.HeapObjects}} {{end}}</td>
// 		</tr>
// 		<tr>
// 			<td>Время последней проверки времени хранения логов (UTC):</td>
// 			<td>{{.DbCheckLifetimeTime}}</td>
// 			<td>Версия приложения:</td>
// 			<td>{{.SoftVersion}}</td>
// 			<td>Память (HeapInuse)</td>
// 			<td>{{with .MemStat}} {{.HeapInuse}} MB {{end}}</td>
// 			<td>Память (StackInuse):</td>
// 			<td>{{with .MemStat}} {{.StackInuse}} MB {{end}}</td>
// 		</tr>
// 	</table>

// 	<p style="font-weight:bold">Сообщения, полученные от контроллера</p>

// 	<div style="display: block;  height: 1000px; position: relative; overflow-x: auto;">
// 	<table width="100%" border="1" cellspacing="0" cellpadding="4" class="table table-bordered table-striped mb-0">
// 		<colgroup>
// 			<col span="1" style="width: 10%;">
// 			<col span="7" style="width: 5%;">
// 		</colgroup>
// 		<tr>
// 			<th>Дата, время</th>
// 			<th>Источник</th>
// 			<th>Серъезность</th>
// 			<th>Лок ATC</th>
// 			<th>Уд ATC</th>
// 			<th>Тип</th>
// 			<th>FMTP тип</th>
// 			<th>Направление</th>
// 			<th>Текст</th>
// 		</tr>
// 		{{with .Lr}}
// 			{{range .}}
// 				<tr align="center" bgcolor="{{.MsgColor}}">
// 					<td align="left"> {{.DateTime}}	</td>
// 					<td align="left"> {{.Source}} </td>
// 					<td align="left"> {{.Severity}}	</td>
// 					<td align="left"> {{.ChannelLocName}} </td>
// 					<td align="left"> {{.ChannelRemName}} </td>
// 					<td align="left"> {{.DataType}} </td>
// 					<td align="left"> {{.FmtpType}} </td>
// 					<td align="left"> {{.Direction}} </td>
// 					<td align="left"> {{.Text}} </td>
// 				</tr>
// 			{{end}}
// 		{{end}}
// 	</table>
// </body>
// </html>
// {{end}}
// `
