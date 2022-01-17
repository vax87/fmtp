package oldi

import (
	"context"
	"sync"
	"time"

	pb "fdps/fmtp/chief/proto/fmtp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	providerValidDur = 10 * time.Second
	msgValidDur      = 30 * time.Second
	maxMsgToSend     = 10
)

// fmtpServerImpl - реализация интерфейса grpc сервера
type fmtpGrpcServerImpl struct {
	sync.Mutex

	msgToFdps []*pb.Msg

	clntActivity map[string]time.Time // ключ - адрес fdps провайдера, значение - время последней активности
}

func newFmtpGrpcServerImpl() *fmtpGrpcServerImpl {
	retValue := fmtpGrpcServerImpl{}
	retValue.msgToFdps = make([]*pb.Msg, 0)
	retValue.clntActivity = make(map[string]time.Time)
	return &retValue
}

func (s *fmtpGrpcServerImpl) SendMsg(ctx context.Context, msg *pb.MsgList) (*pb.SvcResult, error) {
	p, _ := peer.FromContext(ctx)
	s.clntActivity[p.Addr.String()] = time.Now().UTC()

	// \todo обрабатывать собщения для отправки в канал
	//logger.PrintfInfo("FMTP FORMAT %#v", fmtp_logger.LogCntrlSDT(fmtp_logger.SeverityInfo, chief_settings.OLDIProvider,
	//fmt.Sprintf("Получено сообщение от плановой подсистемы: %s.", oldiPkg.Text)))

	//accMsg := fdps.FdpsOldiAcknowledge{Id: pkg.Id}
	// отправка подтверждения
	//cc.OutOldiPacketChan <- accMsg.ToString()
	// logger.PrintfInfo("FMTP FORMAT %#v", fmtp_logger.LogCntrlSDT(fmtp_logger.SeverityInfo, chief_settings.OLDIProvider,
	// 	fmt.Sprintf("Плановой подсистеме отправлено подтверждение: %s.", accMsg.ToString())))

	// \todo обрабатывать подтвержденияот плановой
	// logger.PrintfInfo("FMTP FORMAT %#v", fmtp_logger.LogCntrlSDT(fmtp_logger.SeverityInfo, chief_settings.OLDIProvider,
	// 	fmt.Sprintf("Получено подтверждение от плановой подсистемы: %s.", string(curAccBytes))))

	return nil, status.New(codes.OK, "").Err()
}

func (s *fmtpGrpcServerImpl) RecvMsq(ctx context.Context, msg *pb.SvcReq) (*pb.MsgList, error) {
	s.Lock()
	defer s.Unlock()

	p, _ := peer.FromContext(ctx)
	s.clntActivity[p.Addr.String()] = time.Now().UTC()

	toSend := make([]*pb.Msg, maxMsgToSend)
	if len(s.msgToFdps) > maxMsgToSend {
		toSend = append(toSend, s.msgToFdps[:maxMsgToSend]...)
		s.msgToFdps = s.msgToFdps[maxMsgToSend:]
	} else {
		copy(toSend, s.msgToFdps)
	}

	return &pb.MsgList{List: toSend}, status.New(codes.OK, "").Err()
}

// адреса провайдеров, активных в заданный промежуток времени
func (s *fmtpGrpcServerImpl) getActiveProviders() []string {
	retValue := make([]string, 0)
	nowTime := time.Now().UTC()
	for key, val := range s.clntActivity {
		if val.Add(providerValidDur).After(nowTime) {
			retValue = append(retValue, key)
		}
	}
	return retValue
}

func (s *fmtpGrpcServerImpl) appendMsg(msg pb.Msg) {
	s.Lock()
	defer s.Unlock()

	s.msgToFdps = append(s.msgToFdps, &msg)
}

func (s *fmtpGrpcServerImpl) cleanOldMsg() {
	s.Lock()
	defer s.Unlock()

	for i, v := range s.msgToFdps {
		if v.Rrtime.AsTime().Add(msgValidDur).Before(time.Now().UTC()) {
			s.msgToFdps[i] = s.msgToFdps[len(s.msgToFdps)-1]
			s.msgToFdps[len(s.msgToFdps)-1] = nil
			s.msgToFdps = s.msgToFdps[:len(s.msgToFdps)-1]
		} else {
			// раз попался первый с валидным временем, последующие новее
			break
		}
	}
}
