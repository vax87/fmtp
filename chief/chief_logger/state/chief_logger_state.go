package state

const (
	LoggerStateOk    = "ok"
	LoggerStateError = "error"
)

// состоение логгера
type LoggerState struct {
	LoggerConnected   string `json:"LoggerConnected"`   // признак наличия подключения контроллера к логгеру
	LoggerDbConnected string `json:"LoggerDbConnected"` // признак подключения логгера к БД
	LoggerDbError     string `json:"LoggerDbError"`     // текст ошибки при работе с БД
	LoggerVersion     string `json:"LoggerVersion"`     // версия логгера
}
