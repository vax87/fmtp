package oldi

import (
	"fmt"
	"log"
	"net"
	"time"

	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
	pb "fdps/fmtp/chief/proto/fmtp"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OldiGrpcController контроллер для работы с провайдером OLDI по GRPC
type OldiGrpcController struct {
	SettsChan chan []chief_settings.ProviderSettings // канал для приема настроек провайдеров
	setts     []chief_settings.ProviderSettings      // текущие настройки провайдеров

	FromFdpsChan chan pb.Msg // канал для приема сообщений от провайдера OLDI
	ToFdpsChan   chan pb.Msg // канал для отправки сообщений провайдеру OLDI

	checkStateTicker      *time.Ticker // тикер для проверки состояния контроллера
	checkMsgForFdpsTicker *time.Ticker // тикер проверки валидности сообщений для fdps

	grpcServer *grpc.Server
	fmtpServer *fmtpGrpcServerImpl
	grpcPort   int

	providerEncoding string
}

// NewGrpcController конструктор
func NewOldiGrpcController() *OldiGrpcController {
	return &OldiGrpcController{
		SettsChan:             make(chan []chief_settings.ProviderSettings, 10),
		FromFdpsChan:          make(chan pb.Msg, 1024),
		ToFdpsChan:            make(chan pb.Msg, 1024),
		checkStateTicker:      time.NewTicker(stateTickerInt),
		checkMsgForFdpsTicker: time.NewTicker(msgValidDur),
		fmtpServer:            newFmtpGrpcServerImpl(),
	}
}

func (c *OldiGrpcController) startGrpcServer() {
	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", c.grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	c.grpcServer = grpc.NewServer()
	pb.RegisterFmtpServiceServer(c.grpcServer, c.fmtpServer)

	if err := c.grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (c *OldiGrpcController) stopGrpcServer() {
	//c.grpcServer.GracefulStop()
}

// Work реализация работы
func (c *OldiGrpcController) Work() {

	testTicker := time.NewTicker(300 * time.Millisecond)

	for {
		select {

		case <-testTicker.C:
			c.fmtpServer.appendMsg(pb.Msg{
				Cid:       "FROM",
				RemoteAtc: "TOTO",
				Tp:        "operational",
				Txt:       fmt.Sprintf("TEST MESSAGE %s", time.Now().UTC().Format("2006-01-02 15:04:05")),
				Id:        "456",
				Rrtime:    timestamppb.New(time.Now().UTC()),
				Rqtime:    timestamppb.New(time.Now().UTC()),
			})

		// получены новые настройки каналов
		case c.setts = <-c.SettsChan:
			var localPort int
			for _, val := range c.setts {
				localPort = val.LocalPort
				c.providerEncoding = val.ProviderEncoding
			}

			if c.grpcPort != localPort {
				c.grpcPort = localPort

				c.stopGrpcServer()
				c.startGrpcServer()
			}

		// получен новый пакет для отправки провайдеру
		case incomeData := <-c.ToFdpsChan:
			c.fmtpServer.appendMsg(incomeData)

		// сработал тикер проверки состояния контроллера
		case <-c.checkStateTicker.C:
			var states []chief_state.ProviderState

			activeProviders := c.fmtpServer.getActiveProviders()

			for _, val := range c.setts {
				curState := chief_state.ProviderState{
					ProviderID:    val.ID,
					ProviderType:  val.DataType,
					ProviderIPs:   val.IPAddresses,
					ProviderState: chief_state.StateError,
				}

			ACTPROVLOOP:
				for _, actPrVal := range activeProviders {
					for _, setPrVal := range curState.ProviderIPs {
						if actPrVal == setPrVal {
							curState.ProviderState = chief_state.StateOk
							break ACTPROVLOOP
						}
					}
				}

				states = append(states, curState)
			}
			chief_state.SetOldiProviderState(states)

		// сработал тикер проверки валидности сообщений для fdps
		case <-c.checkMsgForFdpsTicker.C:
			c.fmtpServer.cleanOldMsg()

		}
	}
}
