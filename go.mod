module fmtp

go 1.17

replace (
	lemz.com/fdps/logger => ../mods/logger
	lemz.com/fdps/prom_metrics => ../mods/prom_metrics
	lemz.com/fdps/utils => ../mods/utils
	lemz.com/fdps/web_sock => ../mods/web_sock
)

require (
	github.com/docker/docker v20.10.12+incompatible
	github.com/go-redis/redis/v8 v8.11.4
	github.com/godror/godror v0.30.2
	github.com/golang-collections/go-datastructures v0.0.0-20150211160725-59788d5eb259
	github.com/gorilla/websocket v1.5.0
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
	lemz.com/fdps/logger v1.0.2
	lemz.com/fdps/prom_metrics v1.0.2
	lemz.com/fdps/utils v1.0.0
	lemz.com/fdps/web_sock v1.0.1
)

require (
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/containerd v1.6.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/godror/knownpb v0.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/marcsauter/single v0.0.0-20201009143647-9f8d81240be2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.18.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)
