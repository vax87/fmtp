package oldi

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"fmtp/chief/chief_metrics"
	"fmtp/chief/chief_settings"
	pb "fmtp/chief/proto/fmtp"
	"fmtp/configurator"
	"fmtp/fmtp_log"

	"lemz.com/fdps/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	providerValidDur = 10 * time.Second // время, после которого, если не приходят сообщения от провайдера, то считаем его недоступным
	msgValidDur      = 30 * time.Second // время, в течении которого сообщение валидно
	maxMsgToSend     = 1000             // максимальное кол-во соообщений для отправки провайдеру
)

// fmtpServerImpl - реализация интерфейса grpc сервера
type fmtpGrpcServerImpl struct {
	sync.Mutex

	msgToFdps    []*pb.Msg
	clntActivity sync.Map              //map[string]time.Time  // ключ - адрес fdps провайдера, значение - время последней активности
	FromFdpsChan chan pb.MsgWithChanId // канал для приема сообщений от провайдера OLDI
}

func newFmtpGrpcServerImpl() *fmtpGrpcServerImpl {
	retValue := fmtpGrpcServerImpl{}
	retValue.msgToFdps = make([]*pb.Msg, 0)
	retValue.FromFdpsChan = make(chan pb.MsgWithChanId, 1024)
	return &retValue
}

func (s *fmtpGrpcServerImpl) SendMsg(ctx context.Context, msg *pb.MsgList) (*pb.SvcResult, error) {
	p, _ := peer.FromContext(ctx)
	if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
		s.clntActivity.Store(host, time.Now().UTC())
	}
	var errorString string

	metric := chief_metrics.ProvMetrics{RecvCount: len(msg.List)}

	for _, val := range msg.List {
		chId := configurator.ChiefCfg.GetChannelIdByCid(val.Cid)
		if chId != -1 {
			s.FromFdpsChan <- pb.MsgWithChanId{PbMsg: val, ChanId: chId}
		} else {
			errNoChannel := fmt.Sprintf("Не найден FMTP канал для отправки сообщения. CID (remote ATC): %s", val.Cid)
			errorString += errNoChannel + "\n"
			logger.PrintfErr(errNoChannel)
			metric.MissedCount++
		}
	}
	chief_metrics.ProvMetricsChan <- metric
	return &pb.SvcResult{Errormessage: errorString}, status.New(codes.OK, "").Err()
}

func (s *fmtpGrpcServerImpl) RecvMsq(ctx context.Context, msg *pb.SvcReq) (*pb.MsgList, error) {
	s.Lock()
	defer s.Unlock()

	p, _ := peer.FromContext(ctx)
	if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
		s.clntActivity.Store(host, time.Now().UTC())
	}

	toSend := make([]*pb.Msg, 0)

	if len(s.msgToFdps) > maxMsgToSend {
		toSend = append(toSend, s.msgToFdps[:maxMsgToSend]...)
		s.msgToFdps = s.msgToFdps[maxMsgToSend:]
	} else {
		toSend = make([]*pb.Msg, len(s.msgToFdps))
		copy(toSend, s.msgToFdps)
		s.msgToFdps = s.msgToFdps[:0]
	}
	chief_metrics.ProvMetricsChan <- chief_metrics.ProvMetrics{SendCount: len(toSend)}
	for _, val := range toSend {
		logger.PrintfInfo("FMTP FORMAT %#v", fmtp_log.LogCntrlSDT(fmtp_log.SeverityInfo, chief_settings.OLDIProvider,
			fmt.Sprintf("Плановой подсистеме отправлено сообщение: %s.", val.Txt)))
	}
	return &pb.MsgList{List: toSend}, status.New(codes.OK, "").Err()
}

// адреса провайдеров, активных в заданный промежуток времени
func (s *fmtpGrpcServerImpl) getActiveProviders() []string {
	retValue := make([]string, 0)
	nowTime := time.Now().UTC()
	s.clntActivity.Range(func(key, value interface{}) bool {
		if value.(time.Time).Add(providerValidDur).After(nowTime) {
			retValue = append(retValue, key.(string))
		}
		return true
	})
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

	for idx, val := range s.msgToFdps {
		if val.Rrtime.AsTime().UTC().Add(msgValidDur).Before(time.Now().UTC()) {
			maxIdx = idx

			logger.PrintfErr("now time %s\ntime grpc %s\ntime grpc TO UTC %s\ntime grpc TO UTC + 30 %s\n",
				time.Now().UTC().Format("2006-01-02 15:04:05"),
				val.Rrtime.AsTime().Format("2006-01-02 15:04:05"),
				val.Rrtime.AsTime().UTC().Format("2006-01-02 15:04:05"),
				val.Rrtime.AsTime().UTC().Add(msgValidDur).Format("2006-01-02 15:04:05"))

			logger.PrintfWarn("FMTP FORMAT %#v", fmtp_log.LogCntrlSDT(fmtp_log.SeverityWarning, chief_settings.OLDIProvider,
				fmt.Sprintf("Сообщение удалено из очереди на отправку провайдеру по истечении 30 сек.: %s", val.Txt)))
		} else {
			// раз попался первый с валидным временем, последующие новее
			break
		}
	}

	if maxIdx != -1 {
		for idx := 0; idx <= maxIdx; idx++ {
			s.msgToFdps[idx] = nil
		}
		if maxIdx == len(s.msgToFdps)-1 {
			s.msgToFdps = s.msgToFdps[:0]
		} else {
			s.msgToFdps = s.msgToFdps[maxIdx+1 : len(s.msgToFdps)]
		}
		chief_metrics.ProvMetricsChan <- chief_metrics.ProvMetrics{TimeoutCount: maxIdx}
	}
}
