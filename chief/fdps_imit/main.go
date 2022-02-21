package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	pb "fmtp/chief/proto/fmtp"

	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc"

	"lemz.com/fdps/prom_metrics"
)

const (
	address      = "192.168.1.24:55566"
	sendInterval = 300 * time.Millisecond
	recvInterval = 300 * time.Millisecond
)

const (
	metricImitMsg = "msg"

	metricTypeLabel = "tp"

	tpSend   = "send"
	tpRecv   = "recv"
	tpMissed = "miss"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("did not connect: %v", err)
	}

	defer conn.Close()
	gc := pb.NewFmtpServiceClient(conn)

	sendTicker := time.NewTicker(sendInterval)
	recvTicker := time.NewTicker(recvInterval)
	var msgId int64

	testTicker := time.NewTicker(500 * time.Millisecond)
	msgs := pb.MsgList{}

	var expectBuffer []*pb.Msg
	checkExpectedTicker := time.NewTicker(sendInterval * 10)
	var expectMutex sync.Mutex

	/////////////////////////////////////////////////////////////////////////

	prom_metrics.SetSettings(prom_metrics.PusherSettings{
		PusherIntervalSec: 1,
		GatewayUrl:        "http://192.168.1.24:9100", // from lemz
		//GatewayUrl:       "http://127.0.0.1:9100",	// from home
		GatewayJob:       "fmtp",
		CollectNamespace: "fmtp",
		CollectSubsystem: "imit",
		CollectLabels:    map[string]string{"host": "192.168.10.219"},
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
			for n := 0; n < 10; n++ {
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
					log.Printf("Error SendMsg: %v", err)
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
				log.Printf("Error RecvMsq: %v", err)
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

					log.Printf("!!!!Error expected msg not recieved %d", expId)

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
