module fdps/fmtp/channel

go 1.17

replace (
	lemz.com/fdps/logger => ../../../fdps/go_utils/logger
	lemz.com/fdps/prom_metrics => ../../../fdps/go_utils/prom_metrics
	lemz.com/fdps/web_sock => ../../../fdps/go_utils/web_sock
)

require lemz.com/fdps/logger v1.0.2

require (
	github.com/gorilla/mux v1.8.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)
