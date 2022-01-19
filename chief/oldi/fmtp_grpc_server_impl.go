package oldi

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "fdps/fmtp/chief/proto/fmtp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	providerValidDur = 10 * time.Second
	msgValidDur      = 30 * time.Second // время, в течении которого сообщение валидно
	maxMsgToSend     = 1000             // максимальное кол-во соообщений для отправки провайдеру
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

	return &pb.SvcResult{Errormessage: ""}, status.New(codes.OK, "").Err()
}

func (s *fmtpGrpcServerImpl) RecvMsq(ctx context.Context, msg *pb.SvcReq) (*pb.MsgList, error) {
	s.Lock()
	defer s.Unlock()

	p, _ := peer.FromContext(ctx)
	s.clntActivity[p.Addr.String()] = time.Now().UTC()

	toSend := make([]*pb.Msg, 0)

	fmt.Printf("RecvMsq %s. ", time.Now().UTC().Format("2006-01-02 15:04:05"))
	fmt.Printf("\t s.msgToFdps len: %d. ", len(s.msgToFdps))

	if len(s.msgToFdps) > maxMsgToSend {
		toSend = append(toSend, s.msgToFdps[:maxMsgToSend]...)
		s.msgToFdps = s.msgToFdps[maxMsgToSend:]
	} else {
		toSend = make([]*pb.Msg, len(s.msgToFdps))
		copy(toSend, s.msgToFdps)
		s.msgToFdps = s.msgToFdps[:0]
	}
	fmt.Printf("\t toSend len: %d. s.msgToFdps len: %d. \n\n", len(toSend), len(s.msgToFdps))

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

func (s *fmtpGrpcServerImpl) appendMsg(msg *pb.Msg) {
	s.Lock()
	defer s.Unlock()

	s.msgToFdps = append(s.msgToFdps, msg)
}

func (s *fmtpGrpcServerImpl) cleanOldMsg() {
	s.Lock()
	defer s.Unlock()

	maxIdx := -1

	for i, v := range s.msgToFdps {
		if v.Rrtime.AsTime().Add(msgValidDur).Before(time.Now().UTC()) {
			maxIdx = i
		} else {
			// раз попался первый с валидным временем, последующие новее
			break
		}
	}
	fmt.Printf("cleanOldMsg. msgToFdps len: %d, maxIdx: %d", len(s.msgToFdps), maxIdx)

	if maxIdx != -1 {
		for idx := 0; idx <= maxIdx; idx++ {
			s.msgToFdps[idx] = nil
		}
		if maxIdx == len(s.msgToFdps)-1 {
			s.msgToFdps = s.msgToFdps[:0]
		} else {
			s.msgToFdps = s.msgToFdps[maxIdx+1 : len(s.msgToFdps)]
		}
	}
	fmt.Printf("	clear len: %d \n\n", len(s.msgToFdps))
}