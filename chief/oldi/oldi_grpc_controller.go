package oldi

import (
	"fmt"
	"log"
	"net"
	"time"

	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
	pb "fdps/fmtp/chief/proto/fmtp"
	chief_cfg "fdps/fmtp/chief_configurator"
	"fdps/fmtp/fmtp_logger"
	"fdps/go_utils/logger"

	"google.golang.org/grpc"
)

// OldiGrpcController контроллер для работы с провайдером OLDI по GRPC
type OldiGrpcController struct {
	SettsChangedChan chan struct{} // канал для приема настроек провайдеров

	FromFdpsChan chan pb.MsgWithChanId // канал для приема сообщений от провайдера OLDI
	ToFdpsChan   chan *pb.Msg          // канал для отправки сообщений провайдеру OLDI

	checkStateTicker      *time.Ticker // тикер для проверки состояния контроллера
	checkMsgForFdpsTicker *time.Ticker // тикер проверки валидности сообщений для fdps

	grpcServer  *grpc.Server
	fmtpServer  *fmtpGrpcServerImpl
	grpcAddress string
	grpsServed  bool
}

// NewGrpcController конструктор
func NewOldiGrpcController() *OldiGrpcController {
	return &OldiGrpcController{
		SettsChangedChan:      make(chan struct{}, 10),
		FromFdpsChan:          make(chan pb.MsgWithChanId, 1024),
		ToFdpsChan:            make(chan *pb.Msg, 1024),
		checkStateTicker:      time.NewTicker(stateTickerInt),
		checkMsgForFdpsTicker: time.NewTicker(msgValidDur),
		fmtpServer:            newFmtpGrpcServerImpl(),
		grpsServed:            false,
	}
}

func (c *OldiGrpcController) startGrpcServer() {
	lis, err := net.Listen("tcp4", c.grpcAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	c.grpcServer = grpc.NewServer()
	pb.RegisterFmtpServiceServer(c.grpcServer, c.fmtpServer)
	c.grpsServed = true
	if err := c.grpcServer.Serve(lis); err != nil {
		c.grpsServed = false
		log.Fatalf("failed to serve: %v", err)
	}
}

func (c *OldiGrpcController) stopGrpcServer() {
	c.grpcServer.Stop()
}

// Work реализация работы
func (c *OldiGrpcController) Work() {

	for {
		select {

		// получены новые настройки каналов
		case <-c.SettsChangedChan:
			newGrpcAddress := fmt.Sprintf(":%d", chief_cfg.ChiefCfg.OldiProviderPort)

			if newGrpcAddress != c.grpcAddress {
				c.grpcAddress = newGrpcAddress
				if c.grpsServed {
					c.stopGrpcServer()
				}
				go c.startGrpcServer()
			}

		// получен новый пакет для отправки провайдеру
		case incomeData := <-c.ToFdpsChan:
			c.fmtpServer.appendMsg(incomeData)

		// сработал тикер проверки состояния контроллера
		case <-c.checkStateTicker.C:
			var states []chief_state.ProviderState

			activeProviders := c.fmtpServer.getActiveProviders()

			for _, val := range chief_cfg.ChiefCfg.ProvidersSetts {
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

		case msgFromFdps := <-c.fmtpServer.FromFdpsChan:
			c.FromFdpsChan <- msgFromFdps

			logger.PrintfInfo("FMTP FORMAT %#v", fmtp_logger.LogCntrlSDT(fmtp_logger.SeverityInfo, chief_settings.OLDIProvider,
				fmt.Sprintf("Получено сообщение от плановой подсистемы: %s", msgFromFdps.PbMsg.Txt)))
		}
	}
}
