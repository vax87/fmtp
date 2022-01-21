package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	pb "fdps/fmtp/chief/proto/fmtp"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc"
)

const (
	address      = "localhost:55566"
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
	msgId := 0

	testTicker := time.NewTicker(200 * time.Millisecond)
	msgs := pb.MsgList{}

	var sendedCount, receivedCount int64

	beginWorkTime := time.Now().UTC()

	for {
		select {

		case <-testTicker.C:
			curTime := time.Now().UTC()

			for n := 0; n < 10; n++ {
				msgId++
				msgs.List = append(msgs.List, &pb.Msg{
					Cid:    "UIII",
					Tp:     "operational",
					Txt:    fmt.Sprintf("MSG TO UIII №%d time: %s", msgId, curTime.Format("2006-01-02 15:04:05")),
					Id:     strconv.Itoa(msgId),
					Rrtime: timestamppb.New(curTime),
					Rqtime: timestamppb.New(curTime),
				})

				msgId++
				msgs.List = append(msgs.List, &pb.Msg{
					Cid:    "UEEE",
					Tp:     "operational",
					Txt:    fmt.Sprintf("MSG TO UEEE №%d time: %s", msgId, curTime.Format("2006-01-02 15:04:05")),
					Id:     strconv.Itoa(msgId),
					Rrtime: timestamppb.New(curTime),
					Rqtime: timestamppb.New(curTime),
				})
			}

		case <-sendTicker.C:
			r, err := gc.SendMsg(context.Background(), &msgs)
			if err != nil {
				log.Printf("Error SendMsg: %v", err)
			}
			log.Printf("SendMsg result %s ", r.String())
			sendedCount += int64(len(msgs.List))

			msgs.List = msgs.List[:0]

			if diffTime := time.Now().UTC().Sub(beginWorkTime).Seconds(); diffTime > 0 {
				fmt.Printf("sendCount: %d. SendPerSecond: %f \n", sendedCount, float64(sendedCount)/diffTime)
			}

		case <-recvTicker.C:
			curTime := time.Now().UTC()

			r, err := gc.RecvMsq(context.Background(), &pb.SvcReq{Data: fmt.Sprintf("any data. recv time: %s ", curTime.Format("2006-01-02 15:04:05"))})
			if err != nil {
				log.Printf("Error RecvMsq: %v", err)
			}
			fmt.Printf("RecvMsq result %s. ", time.Now().UTC().Format("2006-01-02 15:04:05"))
			if r != nil {
				if len(r.List) > 0 {
					fmt.Printf("\t first id %s  tm %s \n\n", r.List[0].GetId(), r.List[0].GetRrtime().AsTime().Format("2006-01-02 15:04:05"))
					fmt.Printf("\t last id %s  tm %s \n\n", r.List[len(r.List)-1].GetId(), r.List[len(r.List)-1].GetRrtime().AsTime().Format("2006-01-02 15:04:05"))
				} else {
					fmt.Println("\t empty list \n\n")
				}
				receivedCount += int64(len(r.List))
			}
			if diffTime := time.Now().UTC().Sub(beginWorkTime).Seconds(); diffTime > 0 {
				fmt.Printf("receivedCount: %d. RecvPerSecond: %f \n", receivedCount, float64(receivedCount)/diffTime)
			}
		}
	}

}
