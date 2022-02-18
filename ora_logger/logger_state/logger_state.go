package logger_state

const (
	StateOk    = "ok"
	StateError = "error"
)

var CommonLoggerState LoggerState

type LoggerState struct {
	LoggerID       string `json:"LoggerID"`
	IPAddr         string `json:"LoggerIP"`
	CommonState    string `json:"CommonState"`
	OraConnected   string `json:"OraConnected"`
	OraError       string `json:"OraError"`
	RedisConnected string `json:"RedisConnected"`
	RedisError     string `json:"RedisError"`
	MetricsState   string `json:"MetricsState"`
	MetricsError   string `json:"MetricsError"`
}

func SetOraState(conn, err string) {
	CommonLoggerState.OraConnected = conn
	CommonLoggerState.OraError = err
	checkCommonState()
}

func SetRedisState(conn, err string) {
	CommonLoggerState.RedisConnected = conn
	CommonLoggerState.RedisError = err
	checkCommonState()
}

func SetMetricsState(state, err string) {
	CommonLoggerState.MetricsState = state
	CommonLoggerState.MetricsError = err
	checkCommonState()
}

func checkCommonState() {
	if CommonLoggerState.OraConnected == StateOk &&
		CommonLoggerState.RedisConnected == StateOk &&
		CommonLoggerState.MetricsState == StateOk {
		CommonLoggerState.CommonState = StateOk
	} else {
		CommonLoggerState.CommonState = StateError
	}
}
