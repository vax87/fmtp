package web

import (
	"html/template"
	// 	"runtime"
	// 	"fdps/fmtp/chief/chief_logger/common"
)

// // MB кол-во байт в мегабайте
// const MB float64 = 1 << 20

// // PageInterface итнерфейс web странички для отладки сервисов
type PageInterface interface {
	initialize()

	Lock()
	Unlock()

	template() *template.Template

	//setGrCount(int)         // задать кол-во горутин
	//setMemStat(memUsage)    // задать параметры использования памяти
	//setVersion(string)      // задать версию софта
	//setWorkTime(timeInWork) // задать время работы

	//appendLog(message logWithColor)

	//updateLogs()
}

// // структура для хранения значений времени работы с запуска и текущего времени
// type timeInWork struct {
// 	Days  string
// 	Hours string
// 	Mins  string
// 	Secs  string
// 	Now   string
// }

// // структура для хранения параметров использования памяти
// type memUsage struct {
// 	Sys          float64 // MB
// 	TotalAlloc   float64 // MB
// 	Alloc        float64 // MB
// 	HeapAlloc    float64 // MB
// 	HeapSys      float64 // MB
// 	HeapIdle     float64 // MB
// 	HeapInuse    float64 // MB
// 	HeapReleased float64 // MB
// 	StackInuse   float64 // MB
// 	StackSys     float64 // MB
// 	OtherSys     float64 // MB
// 	NumGC        uint32
// 	HeapObjects  uint64
// }

// func (mu *memUsage) fromRuntimeMemStat(memStat runtime.MemStats) {
// 	mu.Sys = float64(memStat.Sys) / MB
// 	mu.TotalAlloc = float64(memStat.TotalAlloc) / MB
// 	mu.Alloc = float64(memStat.Alloc) / MB
// 	mu.HeapAlloc = float64(memStat.HeapAlloc) / MB
// 	mu.HeapSys = float64(memStat.HeapSys) / MB
// 	mu.HeapIdle = float64(memStat.HeapIdle) / MB
// 	mu.HeapInuse = float64(memStat.HeapInuse) / MB
// 	mu.HeapReleased = float64(memStat.HeapReleased) / MB
// 	mu.StackInuse = float64(memStat.StackInuse) / MB
// 	mu.StackSys = float64(memStat.StackSys) / MB
// 	mu.OtherSys = float64(memStat.OtherSys) / MB
// 	mu.NumGC = memStat.NumGC
// 	mu.HeapObjects = memStat.HeapObjects
// }

// // сообщение журнала с цветом
// type logWithColor struct {
// 	common.LogMessage
// 	MsgColor string // цвет в таБлице логов
// }

// const (
// 	//logSizeMax = 100
// 	// OkColor цвет состояния 'ОК'
// 	OkColor = "#eaf4e3"
// 	// ErrorColor цвет состояния 'Ошибка'
// 	ErrorColor = "#e7cfce"
// 	// StopColor цвет состояния 'Остановлено'
// 	StopColor = "#c1c1bf"
// 	// DefaultColor цвет неопределенного состояния
// 	DefaultColor = "#EAECEE"

// 	// logDebugColor   = "#e1e8f6"
// 	// logInfoColor    = "#eaf4e3"
// 	// logWarningColor = "#edecc3"
// 	// logErrorColor   = "#e7cfce"
// )
