package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	pb "fdps/fmtp/chief/proto/fmtp"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc"

	prom_metrics "fdps/fmtp/chief/fdps_imit/prom"
)

const (
	address      = ":55566"
	sendInterval = 300 * time.Millisecond
	recvInterval = 300 * time.Millisecond
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
	}

	defer conn.Close()
	gc := pb.NewFmtpServiceClient(conn)

	sendTicker := time.NewTicker(sendInterval)
	recvTicker := time.NewTicker(recvInterval)
	var msgId int64

	testTicker := time.NewTicker(700 * time.Millisecond)
	msgs := pb.MsgList{}

	var expectBuffer []*pb.Msg
	checkExpectedTicker := time.NewTicker(sendInterval * 3)
	var expectMutex sync.Mutex

	go prom_metrics.Work(":9100")

	for {
		select {

		case <-testTicker.C:
			//fmt.Println("<-testTicker.C: BEGIN")
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
			//fmt.Println("<-testTicker.C: END")

		case <-sendTicker.C:
			//fmt.Println("<-sendTicker.C: BEGIN")
			if len(msgs.List) > 0 {
				expectMutex.Lock()
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
				defer cancel()
				_, err := gc.SendMsg(ctx, &msgs)
				if err != nil {
					log.Printf("Error SendMsg: %v", err)
				} else {
					//log.Printf("SendMsg result %s ", r.String())
					//fmt.Println("AddMsgSendCount BEGIN")
					prom_metrics.AddMsgSendCount(len(msgs.List))
					//fmt.Println("AddMsgSendCount END")
					msgs.List = msgs.List[:0]
				}
				expectMutex.Unlock()
			} else {
				fmt.Println("empty list")
			}
			//fmt.Println("<-sendTicker.C: END")

			// if diffTime := time.Now().UTC().Sub(beginWorkTime).Seconds(); diffTime > 0 {
			// 	fmt.Printf("sendCount: %d. SendPerSecond: %f \n", sendedCount, float64(sendedCount)/diffTime)
			// }

		case <-recvTicker.C:
			//fmt.Println("<-recvTicker.C: BEGIN")
			curTime := time.Now().UTC()

			r, err := gc.RecvMsq(context.Background(), &pb.SvcReq{Data: fmt.Sprintf("any data. recv time: %s ", curTime.Format("2006-01-02 15:04:05"))})
			if err != nil {
				log.Printf("Error RecvMsq: %v", err)
			}
			// fmt.Printf("RecvMsq result %s. ", time.Now().UTC().Format("2006-01-02 15:04:05"))

			if r != nil {
				// if len(r.List) > 0 {
				// 	fmt.Printf("\t first id %s  tm %s \n\n", r.List[0].GetTxt(), r.List[0].GetRrtime().AsTime().Format("2006-01-02 15:04:05"))
				// 	fmt.Printf("\t last id %s  tm %s \n\n", r.List[len(r.List)-1].GetTxt(), r.List[len(r.List)-1].GetRrtime().AsTime().Format("2006-01-02 15:04:05"))
				// } else {
				// 	fmt.Println("\t empty list \n\n")
				// }
				expectMutex.Lock()
				prom_metrics.AddMsgRecvCount(len(r.List))

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
			//fmt.Println("<-recvTicker.C: END")
			// if diffTime := time.Now().UTC().Sub(beginWorkTime).Seconds(); diffTime > 0 {
			// 	fmt.Printf("receivedCount: %d. RecvPerSecond: %f \n", receivedCount, float64(receivedCount)/diffTime)
			// }

		case <-checkExpectedTicker.C:
			//fmt.Println("<-checkExpectedTicker.C: BEGIN")
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
				prom_metrics.AddMsgMissedCount(maxDelIdx)
				if maxDelIdx < len(expectBuffer)-1 {
					expectBuffer = expectBuffer[maxDelIdx+1:]
				} else {
					expectBuffer = expectBuffer[:len(expectBuffer)-1]
				}
			}

			expectMutex.Unlock()
			//fmt.Println("<-checkExpectedTicker.C: END")
		}
	}
}
