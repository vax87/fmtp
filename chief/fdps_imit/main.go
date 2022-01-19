package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "fdps/fmtp/chief/proto/fmtp"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc"
)

const (
	address      = "localhost:55544"
	sendInterval = 3 * time.Second
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

	for {
		select {

		case <-sendTicker.C:
			curTime := time.Now().UTC()

			msgs := pb.MsgList{}
			msgs.List = append(msgs.List,
				&pb.Msg{
					Cid:    "UIII",
					Tp:     "operational",
					Txt:    fmt.Sprintf("TEXT text OLDI text %s", curTime.Format("2006-01-02 15:04:05")),
					Id:     "12345678",
					Rrtime: timestamppb.New(curTime.Add(-1 * time.Second)),
					Rqtime: timestamppb.Now(),
				})

			msgs.List = append(msgs.List,
				&pb.Msg{
					Cid:    "UEEE",
					Tp:     "operational",
					Txt:    fmt.Sprintf("TEXT text OLDI text from UEEE %s", curTime.Format("2006-01-02 15:04:05")),
					Id:     "3432134",
					Rrtime: timestamppb.New(curTime.Add(-2 * time.Second)),
					Rqtime: timestamppb.Now(),
				})

			r, err := gc.SendMsg(context.Background(), &msgs)
			if err != nil {
				log.Printf("Error SendMsg: %v", err)
			}
			log.Printf("SendMsg result %s ", r.String())

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
			}
		}
	}

}
