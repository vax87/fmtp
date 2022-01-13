package oldi

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
	pb "fdps/fmtp/chief/proto/fmtp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	providerValidDur = 10 * time.Second
	msgValidDur      = 30 * time.Second
)

//////////////////////////////////////////////////////////////////////////////////////////

type queueItem struct {
	message     []byte
	channelTime time.Time // время получения из fmtp канала
}

//////////////////////////////////////////////////////////////////////////////////////////

type fmtpGrpcServer struct {
	sync.Mutex

	msgToFdps []queueItem

	clntActivity map[string]time.Time // ключ - адрес fdps провайдера, значение - время последней активности
}

func (s *fmtpGrpcServer) SendMsg(ctx context.Context, msg *pb.MsgList) (*pb.SvcResult, error) {
	p, _ := peer.FromContext(ctx)
	s.clntActivity[p.Addr.String()] = time.Now().UTC()

	return nil, status.New(codes.OK, "").Err()
}

func (s *fmtpGrpcServer) RecvMsq(ctx context.Context, msg *pb.SvcReq) (*pb.MsgList, error) {
	s.Lock()
	defer s.Unlock()

	p, _ := peer.FromContext(ctx)
	s.clntActivity[p.Addr.String()] = time.Now().UTC()

	s.Unlock()
	return nil, status.New(codes.OK, "").Err()
}

// адреса провайдеров, активных в заданный промежуток времени
func (s *fmtpGrpcServer) getActiveProviders() []string {
	retValue := make([]string, 0)
	nowTime := time.Now().UTC()
	for key, val := range s.clntActivity {
		if val.Add(providerValidDur).After(nowTime) {
			retValue = append(retValue, key)
		}
	}
	return retValue
}

func (s *fmtpGrpcServer) appendMsg(msg []byte) {
	s.Lock()
	defer s.Unlock()

	s.msgToFdps = append(s.msgToFdps, queueItem{
		message:     msg,
		channelTime: time.Now().UTC(),
	})
}

func (s *fmtpGrpcServer) cleanOldMsg() {
	s.Lock()
	defer s.Unlock()

	for i, v := range s.msgToFdps {
		if v.channelTime.Add(msgValidDur).Before(time.Now().UTC()) {
			s.msgToFdps[i] = s.msgToFdps[len(s.msgToFdps)-1]
			s.msgToFdps[len(s.msgToFdps)-1] = queueItem{}
			s.msgToFdps = s.msgToFdps[:len(s.msgToFdps)-1]
		} else {
			// раз попался первый с валидным временем, последующие новее
			break
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////////

// OldiGrpcController контроллер для работы с провайдером OLDI по GRPC
type OldiGrpcController struct {
	SettsChan chan []chief_settings.ProviderSettings // канал для приема настроек провайдеров
	setts     []chief_settings.ProviderSettings      // текущие настройки провайдеров

	FromFdpsChan chan []byte // канал для приема сообщений от провайдера OLDI
	ToFdpsChan   chan []byte // канал для отправки сообщений провайдеру OLDI

	checkStateTicker      *time.Ticker // тикер для проверки состояния контроллера
	checkMsgForFdpsTicker *time.Ticker // тикер проверки валидности сообщений для fdps

	grpcServer *grpc.Server
	fmtpServer *fmtpGrpcServer
	grpcPort   int

	providerEncoding string
}

// NewGrpcController конструктор
func NewOldiGrpcController() *OldiGrpcController {
	return &OldiGrpcController{
		SettsChan:             make(chan []chief_settings.ProviderSettings, 10),
		FromFdpsChan:          make(chan []byte, 1024),
		ToFdpsChan:            make(chan []byte, 1024),
		checkStateTicker:      time.NewTicker(stateTickerInt),
		checkMsgForFdpsTicker: time.NewTicker(msgValidDur),
	}
}

func (c *OldiGrpcController) startGrpcServer() {
	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", c.grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	c.grpcServer = grpc.NewServer()
	c.fmtpServer = new(fmtpGrpcServer)
	pb.RegisterFmtpServiceServer(c.grpcServer, c.fmtpServer)

	if err := c.grpcServer.Serve(lis); err != nil {
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
