package web

import (
	"html/template"
	"log"
	"sync"
)

const (
	//logSizeMax = 100
	// OkColor цвет состояния 'ОК'
	OkColor = "#eaf4e3"
	// ErrorColor цвет состояния 'Ошибка'
	ErrorColor = "#e7cfce"
	// StopColor цвет состояния 'Остановлено'
	StopColor = "#c1c1bf"
	// DefaultColor цвет неопределенного состояния
	DefaultColor = "#EAECEE"

	// logDebugColor   = "#e1e8f6"
	// logInfoColor    = "#eaf4e3"
	// logWarningColor = "#edecc3"
	// logErrorColor   = "#e7cfce"
)

// ChiefPage страница контроллера
type ChiefPage struct {
	sync.RWMutex
	templ *template.Template
	Title string
	//TotalTime timeInWork

	//Lr      []*logWithColor
	//logList *list.List

	//GoroutineCount int      // кол-во goroutines
	//MemStat        memUsage // параметры использования памяти
	SoftVersion string // версия софта

	ConfConn      string // состояние подключения к конфигуратору
	ConfConnColor string // цвет состояния подключения к конфигуратору

	ChannelWSConn      string // состояние WS сервера каналов
	ChannelWSConnColor string // цвет состояния WS сервера каналов

	LoggerConn      string // состояние подключения к логгеру
	LoggerConnColor string // цвет состояния подключения к логгеру

	AODBConn      string // состояние WS сервера AODB провайдера
	AODBConnColor string // цвет состояния WS сервера AODB провайдера

	OLDIConn      string // состояние сервера OLDI провайдера
	OLDIConnColor string // цвет состояния сервера OLDI провайдера

	ChannelVersions string // версии каналов
	DockerVersion   string // версия Docker
	IPAddresses     string // список собственных IP

	ChannelStates  []ChannelStateWeb  // состояния каналов
	ProviderStates []ProviderStateWeb // состояния провайдеров
}

// ChannelStateWeb состояние FMTP канала
type ChannelStateWeb struct {
	ChannelID   int    // идентификатор канала *Не переменовывать в ChannelId
	ChannelURL  string // URL web странички канала
	LocalName   string // локальный ATC
	RemoteName  string // удаленный ATC
	DaemonState string // состояние канала *Не переменовывать в ChannelState
	FmtpState   string // FMTP состояние канала
	StateColor  string // цвет состояния канала
}

// ProviderStateWeb состояние провайдера плановой информации
type ProviderStateWeb struct {
	ProviderID    int    // идентификатор провайдера
	ProviderURL   string // URL web странички провайдера
	ProviderType  string // тип провайдера (OLID | AODB)
	ProviderState string // состояние провайдера
	StateColor    string // цвет состояния провайдера
}

func (m *ChiefPage) initialize() {
	m.Lock()
	defer m.Unlock()

	//m.logList = list.New()

	var err error
	if m.templ, err = template.New("ChiefPage").Parse(ChiefPageTemplate); err != nil {
		log.Println("template Parse ERROR")
		return
	}

	m.Title = "FMTP Chief"

	// m.Lr = make([]*logWithColor, logSizeMax, logSizeMax)

	// for i := 0; i < logSizeMax; i++ {
	// 	m.logList.PushFront(&logWithColor{MsgColor: DefaultColor})
	// }

	m.ConfConn = "???"
	m.ConfConnColor = ErrorColor

	m.ChannelWSConn = "???"
	m.ChannelWSConnColor = ErrorColor

	m.LoggerConn = "???"
	m.LoggerConnColor = ErrorColor

	m.AODBConn = "???"
	m.AODBConnColor = ErrorColor

	m.OLDIConn = "???"
	m.OLDIConnColor = ErrorColor

	m.SoftVersion = "???"
}

// func (m *ChiefPage) setGrCount(curGrCount int) {
// 	m.GoroutineCount = curGrCount
// }

// func (m *ChiefPage) setMemStat(curMem memUsage) {
// 	m.MemStat = curMem
// }

// задать версию софта
func (m *ChiefPage) setVersion(vers string) {
	m.SoftVersion = vers
}

// func (m *ChiefPage) setWorkTime(curTime timeInWork) {
// 	m.TotalTime = curTime
// }

func (m *ChiefPage) template() *template.Template {
	return m.templ
}

// func (m *ChiefPage) appendLog(message logWithColor) {
// 	m.Lock()
// 	defer m.Unlock()

// 	m.logList.PushFront(&message)

// 	for m.logList.Len() > logSizeMax {
// 		m.logList.Remove(m.logList.Back())
// 	}
// }

// func (m *ChiefPage) updateLogs() {
// 	for e, i := m.logList.Front(), 0; e != nil && i < logSizeMax; e, i = e.Next(), i+1 {
// 		lr, ok := e.Value.(*logWithColor)
// 		if ok {
// 			m.Lr[i] = lr
// 		}
// 	}
// }

// задание состояния подключения к конфигуратору
func (m *ChiefPage) setConfigConn(conn bool) {
	if conn {
		m.ConfConn = string("Подключено.")
		m.ConfConnColor = OkColor
	} else {
		m.ConfConn = string("Не подключено.")
		m.ConfConnColor = ErrorColor
	}
}

// задание состояния WS сервера каналов
func (m *ChiefPage) setWSChannelState(conn bool) {
	if conn {
		m.ChannelWSConn = string("Подключено.")
		m.ChannelWSConnColor = OkColor
	} else {
		m.ChannelWSConn = string("Не подключено.")
		m.ChannelWSConnColor = ErrorColor
	}
}

// задание состояния подключения к логгеру
func (m *ChiefPage) setLoggerState(conn bool) {
	if conn {
		m.LoggerConn = string("Подключено.")
		m.LoggerConnColor = OkColor
	} else {
		m.LoggerConn = string("Не подключено.")
		m.LoggerConnColor = ErrorColor
	}
}

// задание состояния  WS сервера AODB провайдера
func (m *ChiefPage) setAODBState(conn bool) {
	if conn {
		m.AODBConn = string("Подключено.")
		m.AODBConnColor = OkColor
	} else {
		m.AODBConn = string("Не подключено.")
		m.AODBConnColor = ErrorColor
	}
}

// задание состояния  WS сервера OLDI провайдера
func (m *ChiefPage) setOLDIState(conn bool) {
	if conn {
		m.OLDIConn = string("Подключено.")
		m.OLDIConnColor = OkColor
	} else {
		m.OLDIConn = string("Не подключено.")
		m.OLDIConnColor = ErrorColor
	}
}

// задание версии каналов
func (m *ChiefPage) setChannelVersions(vers string) {
	m.ChannelVersions = vers
}

// задание версии Docker
func (m *ChiefPage) setDockerVersion(vers string) {
	m.DockerVersion = vers
}

// задание списка IP адресов
func (m *ChiefPage) setIPAddresses(ips string) {
	m.IPAddresses = ips
}

// задание состояния каналов
func (m *ChiefPage) setChannelStates(states []ChannelStateWeb) {
	m.ChannelStates = states
}

// задание состояния провайдеров
func (m *ChiefPage) setProviderStates(states []ProviderStateWeb) {
	m.ProviderStates = states
}

var ChiefPageTemplate = `{{define "T"}}
<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
		<script>
		//    window.onload = function() {
		//        setTimeout(function () {
		//            location.reload()
		//        }, 1000);
		//     };
		</script>
	</head>
	<body>
		<font size="3" face="verdana" color="black">
		<table width="100%" border="1" cellspacing="0" cellpadding="4">
			<colgroup>
				<col style = "width: 10%;">
				<col style = "width: 10%;">
				<col style = "width: 10%;">
				<col style = "width: 10%;">
				<col style = "width: 5%;">
				<col style = "width: 5%;">
				<col style = "width: 7%;">
				<col style = "width: 7%;">
			</colgroup>	
			<tr>
				<td>Подключение к конфигуратору:</td>
				<td bgcolor="{{.ConfConnColor}}">{{.ConfConn}}</td>
				<td>Версии FMTP каналов:</td>
				<td>{{.ChannelVersions}}</td>
				<td>В работе:</td>
				<td>{{with .TotalTime}}{{.Days}} дн. {{.Hours}} ч. {{.Mins}} м. {{.Secs}} с.{{end}}</td>
				<td>Сейчас:</td>
				<td>{{with .TotalTime}}{{.Now}}(UTC){{end}}</td>
			</tr>
			<tr>
				<td>WS сервер FMTP каналов:</td>
				<td bgcolor="{{.ChannelWSConnColor}}">{{.ChannelWSConn}}</td>
				<td>Версия Docker:</td>		
				<td>{{.DockerVersion}}</td>
				<td>Кол-во goroutine:</td>
				<td>{{.GoroutineCount}}</td>
				<td>Кол-во вызовов GC:</td>
				<td>{{with .MemStat}} {{.NumGC}} {{end}}</td>
			</tr>
			<tr>
				<td>Подключение к логгеру:</td>
				<td bgcolor="{{.LoggerConnColor}}">{{.LoggerConn}}</td>
				<td>Собственные IP:</td>			
				<td>{{.IPAddresses}}</td>
				<td>Память (Sys):</td>
				<td>{{with .MemStat}} {{.Sys}} MB {{end}}</td>
				<td>Кол-во объектов в памяти:</td>
				<td>{{with .MemStat}} {{.HeapObjects}} {{end}}</td>
			</tr>
			<tr>
				<td>WS сервер AODB провайдера:</td>
				<td bgcolor="{{.AODBConnColor}}">{{.AODBConn}}</td>
				<td>Версия приложения:</td>
				<td>{{.SoftVersion}}</td>	
				<td>Память (HeapInuse)</td>
				<td>{{with .MemStat}} {{.HeapInuse}} MB {{end}}</td>
				<td>Память (StackInuse):</td>
				<td>{{with .MemStat}} {{.StackInuse}} MB {{end}}</td>
			</tr>	
			<tr>
				<td>TCP сервер OLDI провайдера:</td>
				<td bgcolor="{{.OLDIConnColor}}">{{.OLDIConn}}</td>	
				<td colspan ="6"></td>
			</tr>		
		</table>


		<table width="100%" border="1" cellspacing="0" cellpadding="4" >
			<caption style="font-weight:bold">FMTP каналы</caption>
			<tr>
				<th>ID</th>
				<th>URL</th>
				<th>Лок ATC</th>
				<th>Уд ATC</th>
				<th>Состояние</th>		
				<th>FMTP состояние</th>			
			</tr>
			{{with .ChannelStates}}
				{{range .}}
					<tr align="center" bgcolor="{{.StateColor}}">	
						<td align="left"> {{.ChannelID}} </td>	
						<td align="left"> <a href="{{.ChannelURL}}" style="display:block;">{{.ChannelURL}}</a> </td>
						<td align="left"> {{.LocalName}} </td>
						<td align="left"> {{.RemoteName}} </td>
						<td align="left"> {{.DaemonState}} </td>
						<td align="left"> {{.FmtpState}} </td>					
					</tr>
				{{end}}
			{{end}}
		</table>

		<table width="100%" border="1" cellspacing="0" cellpadding="4" >
			<caption style="font-weight:bold">Провайдеры</caption>			
			<tr>
				<th>ID</th>
				<th>URL</th>
				<th>Тип</th>
				<th>Состояние</th>				
			</tr>
			{{with .ProviderStates}}
				{{range .}}
					<tr align="center" bgcolor="{{.StateColor}}">	
						<td align="left"> {{.ProviderID}} </td>	
						<td align="left"> {{.ProviderURL}} </td>
						<td align="left"> {{.ProviderType}} </td>
						<td align="left"> {{.ProviderState}} </td>				
					</tr>
				{{end}}
			{{end}}
		</table>
		</font>
	</body>
</html>
{{end}}
`
