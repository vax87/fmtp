module fdps/fmtp/chief

go 1.17

replace (
	lemz.com/fdps/logger => ../../go_utils/logger
	lemz.com/fdps/prom_metrics => ../../go_utils/prom_metrics
	lemz.com/fdps/web_sock => ../../go_utils/web_sock
)

require (
	github.com/go-redis/redis v6.15.9+incompatible
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
	lemz.com/fdps/logger v1.0.2
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.18.1 // indirect
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781 // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)
