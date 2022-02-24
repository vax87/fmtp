package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	pb "fmtp/chief/proto/fmtp"
	"fmtp/fdps_imit/settings"

	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc"

	"lemz.com/fdps/logger"
	"lemz.com/fdps/prom_metrics"
	"lemz.com/fdps/utils"
)

const (
	appName    = "fdps-oldi-imit"
	appVersion = "2022-02-24 15:49"
)

const (
	metricImitMsg = "msg"

	metricTypeLabel = "tp"

	tpSend   = "send"
	tpRecv   = "recv"
	tpMissed = "miss"
)

func main() {

	logger.InitLoggerSettings(utils.AppPath()+"/config/loggers.json", appName, appVersion)
	if logger.LogSettInst.NeedWebLog {
		utils.AppendHandler(logger.WebLogger)
	}

	var setts settings.Settings
	setts.ReadFromFile()

	// Set up a connection to the server.
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", setts.FmtpCntrlAddress, setts.FmtpCntrlPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.PrintfErr("did not connect to GRPC: %v", err)
	}

	defer conn.Close()
	gc := pb.NewFmtpServiceClient(conn)

	sendTicker := time.NewTicker(time.Duration(setts.SendIntervalMsec) * time.Millisecond)
	recvTicker := time.NewTicker(time.Duration(setts.RecvIntervalMsec) * time.Millisecond)
	var msgId int64

	testTicker := time.NewTicker(time.Duration(setts.ImitGenMsgIntervalMsec) * time.Millisecond)
	msgs := pb.MsgList{}

	var expectBuffer []*pb.Msg
	checkExpectedTicker := time.NewTicker(time.Duration(setts.SendIntervalMsec * 10))
	var expectMutex sync.Mutex

	/////////////////////////////////////////////////////////////////////////

	prom_metrics.SetSettings(prom_metrics.PusherSettings{
		PusherIntervalSec: setts.MetricsIntervalSec,
		GatewayUrl:        setts.MetricsGatewayUrl,
		GatewayJob:        "fmtp",
		CollectNamespace:  "fmtp",
		CollectSubsystem:  "imit",
		CollectLabels:     map[string]string{"host": setts.MetricsHostLabel},
	})
	prom_metrics.AppendCounterVec(metricImitMsg, "Имитатор FMTP сообщений", []string{metricTypeLabel})
	prom_metrics.Initialize()

	prom_metrics.AddToCounterVec(metricImitMsg, 0, map[string]string{metricTypeLabel: tpSend})
	prom_metrics.AddToCounterVec(metricImitMsg, 0, map[string]string{metricTypeLabel: tpRecv})
	prom_metrics.AddToCounterVec(metricImitMsg, 0, map[string]string{metricTypeLabel: tpMissed})

	/////////////////////////////////////////////////////////////////////////

	for {
		select {

		case <-testTicker.C:
			curTime := time.Now().UTC()

			expectMutex.Lock()
			for n := 0; n < setts.ImitGenMsgCount; n++ {
				msgId++
				msg1 := pb.Msg{
					Cid:    "UIII",
					Tp:     "operational",
					Txt:    fmt.Sprintf("MSG TO UIII №%d time: %s", msgId, curTime.Format("2006-01-02 15:04:05")),
					Id:     strconv.FormatInt(msgId, 10),
					Rrtime: timestamppb.New(curTime),
					Rqtime: timestamppb.New(curTime),
				}

				msgId++
				msg2 := pb.Msg{
					Cid:    "UEEE",
					Tp:     "operational",
					Txt:    fmt.Sprintf("MSG TO UEEE №%d time: %s", msgId, curTime.Format("2006-01-02 15:04:05")),
					Id:     strconv.FormatInt(msgId, 10),
					Rrtime: timestamppb.New(curTime),
					Rqtime: timestamppb.New(curTime),
				}

				expectBuffer = append(expectBuffer, &msg1)
				msgs.List = append(msgs.List, &msg1)
				expectBuffer = append(expectBuffer, &msg2)
				msgs.List = append(msgs.List, &msg2)

			}
			expectMutex.Unlock()

		case <-sendTicker.C:
			if len(msgs.List) > 0 {
				expectMutex.Lock()
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
				defer cancel()
				_, err := gc.SendMsg(ctx, &msgs)
				if err != nil {
					logger.PrintfErr("Error SendMsg: %v", err)
				} else {
					prom_metrics.AddToCounterVec(metricImitMsg, len(msgs.List), map[string]string{metricTypeLabel: tpSend})
					msgs.List = msgs.List[:0]
				}
				expectMutex.Unlock()
			} else {
				fmt.Println("empty list")
			}

		case <-recvTicker.C:
			curTime := time.Now().UTC()

			r, err := gc.RecvMsq(context.Background(), &pb.SvcReq{Data: fmt.Sprintf("any data. recv time: %s ", curTime.Format("2006-01-02 15:04:05"))})
			if err != nil {
				logger.PrintfErr("Error RecvMsq: %v", err)
			}

			if r != nil {
				// if len(r.List) > 0 {
				// 	fmt.Printf("\t first id %s  tm %s \n\n", r.List[0].GetTxt(), r.List[0].GetRrtime().AsTime().Format("2006-01-02 15:04:05"))
				// 	fmt.Printf("\t last id %s  tm %s \n\n", r.List[len(r.List)-1].GetTxt(), r.List[len(r.List)-1].GetRrtime().AsTime().Format("2006-01-02 15:04:05"))
				// } else {
				// 	fmt.Println("\t empty list \n\n")
				// }
				expectMutex.Lock()
				prom_metrics.AddToCounterVec(metricImitMsg, len(r.List), map[string]string{metricTypeLabel: tpRecv})

				for _, v := range r.List {

				EXPECT:
					for idx := range expectBuffer {
						if expectBuffer[idx].Txt == v.Txt {
							if idx < len(expectBuffer)-1 {
								expectBuffer = append(expectBuffer[:idx], expectBuffer[idx+1:]...)
							} else {
								expectBuffer = expectBuffer[:len(expectBuffer)-1]
							}
							break EXPECT
						}
					}

				}
				expectMutex.Unlock()
			}

		case <-checkExpectedTicker.C:
			expectMutex.Lock()
			maxDelIdx := -1
			for idx, v := range expectBuffer {
				expId, _ := strconv.ParseInt(v.GetId(), 10, 64)
				if expId+1000 < msgId {
					maxDelIdx = idx

					logger.PrintfErr("!!!!Error expected msg not recieved %d", expId)

				} else {
					break
				}
			}

			if maxDelIdx != -1 {
				prom_metrics.AddToCounterVec(metricImitMsg, maxDelIdx, map[string]string{metricTypeLabel: tpMissed})
				if maxDelIdx < len(expectBuffer)-1 {
					expectBuffer = expectBuffer[maxDelIdx+1:]
				} else {
					expectBuffer = expectBuffer[:len(expectBuffer)-1]
				}
			}

			expectMutex.Unlock()
		}
	}
}
