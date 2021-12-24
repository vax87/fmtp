package web

// import (
// 	"container/list"
// 	"fmt"
// 	"html/template"
// 	"log"
// 	"sync"
// )

// ChannelSettsWeb настройки FMTP канала для отображения в web (для исключения циклю зависимости)
// type ChannelSettsWeb struct {
// 	DataType      string // тип сообщений поверх FMTP.
// 	NetRole       string // тип TCP подключения.
// 	LocalATC      string // локальный АТС.
// 	RemoteATC     string // удаленный АТС.
// 	RemoteAddress string // удаленный адрес.
// 	RemotePort    int    // удаленный порт (для клиента).
// 	LocalPort     int    // локальный порт	(для сервера).
// 	DataEncoding  string // кодировка сообщений.
// }

// // ChannelPage страниа сервиса FMTP канал
// type ChannelPage struct {
// 	sync.RWMutex
// 	templ     *template.Template
// 	Title     string
// 	TotalTime timeInWork

// 	ChSetts          ChannelSettsWeb // настройки канала
// 	ChiefConn        string          // состояние подключения к контроллеру
// 	ChiefConnColor   string          // цвет состояния подключения к контроллеру
// 	ChFMTPState      string          // FMTP состояние канала
// 	ChFMTPStateColor string          // цвет FMTP состояния канала
// 	TCPState         string          // состояние TCP транспорта
// 	TCPStateColor    string          // цвет состояния TCP транспорта

// 	MemStat        memUsage // параметры использования памяти
// 	GoroutineCount int      // кол-во goroutines
// 	SoftVersion    string   // версия софта

// 	Lr      []*logWithColor
// 	logList *list.List
// }

// // инициализация страницы
// func (m *ChannelPage) initialize() {
// 	m.Lock()
// 	defer m.Unlock()

// 	var err error
// 	if m.templ, err = template.New("MainPage").Parse(channelPageTemplate); err != nil {
// 		log.Println("template Parse ERROR")
// 		fmt.Println(err.Error())
// 		return
// 	}

// 	m.Title = "FMTP Канал"

// 	m.logList = list.New()
// 	m.Lr = make([]*logWithColor, logSizeMax, logSizeMax)

// 	for i := 0; i < logSizeMax; i++ {
// 		m.logList.PushFront(&logWithColor{MsgColor: DefaultColor})
// 	}
// }

// func (m *ChannelPage) setGrCount(curGrCount int) {
// 	m.GoroutineCount = curGrCount
// }

// // задание параметров использования памяти
// func (m *ChannelPage) setMemStat(curMem memUsage) {
// 	m.MemStat = curMem
// }

// // задать версию софта
// func (m *ChannelPage) setVersion(vers string) {
// 	m.SoftVersion = vers
// }

// // задание времени работы
// func (m *ChannelPage) setWorkTime(curTime timeInWork) {
// 	m.TotalTime = curTime
// }

// // шаблон страницы
// func (m *ChannelPage) template() *template.Template {
// 	return m.templ
// }

// // добавление сообщения журнала, отправленного контроллеру
// func (m *ChannelPage) appendLog(message logWithColor) {
// 	m.Lock()
// 	defer m.Unlock()

// 	m.logList.PushFront(&message)

// 	for m.logList.Len() > logSizeMax {
// 		m.logList.Remove(m.logList.Back())
// 	}
// }

// // задание настроек канала
// func (m *ChannelPage) setChannelSettings(chSetts ChannelSettsWeb) {
// 	m.ChSetts = chSetts
// }

// // задание состояния подключения к контроллеру
// func (m *ChannelPage) setChiefConn(chiefConn bool) {
// 	if chiefConn {
// 		m.ChiefConn = string("Подключено.")
// 		m.ChiefConnColor = OkColor
// 	} else {
// 		m.ChiefConn = string("Не подключено.")
// 		m.ChiefConnColor = ErrorColor
// 	}
// }

// // задание FMTP состояния канала
// func (m *ChannelPage) setChannelFMTPState(chFMTPState string) {
// 	m.ChFMTPState = chFMTPState
// 	if m.ChFMTPState == "data_ready" {
// 		m.ChFMTPStateColor = OkColor
// 	} else {
// 		m.ChFMTPStateColor = ErrorColor
// 	}
// }

// // задание состояния TCP транспорта
// func (m *ChannelPage) setTCPState(tcpState bool) {
// 	if tcpState {
// 		m.TCPState = string("OK")
// 		m.TCPStateColor = OkColor
// 	} else {
// 		m.TCPState = string("Ошибка")
// 		m.TCPStateColor = ErrorColor
// 	}
// }

// //
// func (m *ChannelPage) updateLogs() {
// 	m.Lock()
// 	defer m.Unlock()

// 	for e, i := m.logList.Front(), 0; e != nil && i < logSizeMax; e, i = e.Next(), i+1 {
// 		lr, ok := e.Value.(*logWithColor)
// 		if ok {
// 			m.Lr[i] = lr
// 		}
// 	}
// }

// var channelPageTemplate = `{{define "T"}}
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
// <font size="2" face="verdana" color="black">

// <table width="100%" border="1" cellspacing="0" cellpadding="4">
// 	<colgroup>
// 		<col style = "width: 10%;">
// 		<col style = "width: 5%;">
// 		<col style = "width: 10%;">
// 		<col style = "width: 5%;">
// 		<col style = "width: 10%;">
// 		<col style = "width: 5%;">
// 		<col style = "width: 10%;">
// 		<col style = "width: 7%;">
// 	</colgroup>
// 	<tr>
// 		<td>Локальный - удаленный ATC:</td>
// 		<td>{{with .ChSetts}} {{.LocalATC}} - {{.RemoteATC}}{{end}}</td>
// 		<td>Тип подключения:</td>
// 		<td>{{with .ChSetts}} {{if eq .NetRole "server"}} TCP сервер{{ else if eq .NetRole "client"}} TCP клиент{{end}} {{end}}</td>
// 		<td>В работе:</td>
// 		<td>{{with .TotalTime}}{{.Days}} дн. {{.Hours}} ч. {{.Mins}} м. {{.Secs}} с.{{end}}</td>
// 		<td>Сейчас:</td>
// 		<td>{{with .TotalTime}}{{.Now}}(UTC){{end}}</td>

// 	</tr>
// 	<tr>
// 		<td>Удаленный IP адрес:</td>
// 		<td>{{with .ChSetts}} {{.RemoteAddress}} {{end}}</td>
// 		<td>{{with .ChSetts}} {{if eq .NetRole "server"}} Локальный порт: {{ else if eq .NetRole "client"}} Удаленный порт: {{end}} {{end}}</td>
// 		<td>{{with .ChSetts}} {{if eq .NetRole "server"}} {{.LocalPort}} {{ else if eq .NetRole "client"}} {{.RemotePort}} {{end}} {{end}}</td>
// 		<td>Кол-во goroutine:</td>
// 		<td>{{.GoroutineCount}}</td>
// 		<td>Кол-во вызовов GC:</td>
// 		<td>{{with .MemStat}} {{.NumGC}} {{end}}</td>
// 	</tr>
// 	<tr>
// 		<td>Подключение к контролеру:</td>
// 		<td bgcolor="{{.ChiefConnColor}}">{{.ChiefConn}}</td>
// 		<td>Тип данных:</td>
// 		<td>{{with .ChSetts}} {{.DataType}} {{end}}</td>
// 		<td>Память (Sys):</td>
// 		<td>{{with .MemStat}} {{.Sys}} MB {{end}}</td>
// 		<td>Кол-во объектов в памяти:</td>
// 		<td>{{with .MemStat}} {{.HeapObjects}} {{end}}</td>
// 	</tr>
// 	<tr>
// 		<td>{{with .ChSetts}} {{if eq .NetRole "server"}} Состояние TCP сервера: {{ else if eq .NetRole "client"}} Подключение к удаленному TCP серверу: {{end}} {{end}}</td>
// 		<td bgcolor="{{.TCPStateColor}}">{{.TCPState}}</td>
// 		<td>Кодировка:</td>
// 		<td>{{with .ChSetts}} {{.DataEncoding}} {{end}}</td>
// 		<td>Память (HeapInuse)</td>
// 		<td>{{with .MemStat}} {{.HeapInuse}} MB {{end}}</td>
// 		<td>Память (StackInuse):</td>
// 		<td>{{with .MemStat}} {{.StackInuse}} MB {{end}}</td>
// 	</tr>
// 	<tr>
// 		<td>FMTP состояние:</td>
// 		<td bgcolor="{{.ChFMTPStateColor}}">{{.ChFMTPState}}</td>
// 		<td colspan ="6"></td>
// 	</tr>
// </table>

// <p style="font-weight:bold">Сообщения, отправленные контроллеру</p>

// <div style="display: block;  height: 650px; position: relative; overflow-x: auto;">
// <table width="100%" border="1" cellspacing="0" cellpadding="4" class="table table-bordered table-striped mb-0">
// 	<colgroup>
// 		<col span="1" style="width: 12%;">
// 		<col span="4" style="width: 7%;">
// 		<col span="1" style="width: 65%;">
//     </colgroup>
// 	<tr>
// 		<th>Дата, время</th>
// 		<th>Источник</th>
// 		<th>Серъезность</th>
// 		<th>FMTP тип</th>
// 		<th>Направление</th>
// 		<th>Текст</th>
//    	</tr>
// 	{{with .Lr}}
// 		{{range .}}
// 			<tr align="center" bgcolor="{{.MsgColor}}">
// 				<td align="left"> {{.DateTime}}	</td>
// 				<td align="left"> {{.Source}} </td>
// 				<td align="left"> {{.Severity}}	</td>
// 				<td align="left"> {{.FmtpType}} </td>
// 				<td align="left"> {{.Direction}} </td>
// 				<td align="left"> {{.Text}} </td>
// 			</tr>
// 		{{end}}
// 	{{end}}
// </table>

// </font>
// </body>
// </html>
// {{end}}
// `
