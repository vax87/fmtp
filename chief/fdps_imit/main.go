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
	recvInterval = 2 * time.Second
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
					Txt:    "TEXT text OLDI text",
					Id:     "12345678",
					Rrtime: timestamppb.New(curTime.Add(-1 * time.Second)),
					Rqtime: timestamppb.Now(),
				})

			msgs.List = append(msgs.List,
				&pb.Msg{
					Cid:    "UEEE",
					Tp:     "operational",
					Txt:    "TEXT text OLDI text from UEEE",
					Id:     "3432134",
					Rrtime: timestamppb.New(curTime.Add(-2 * time.Second)),
					Rqtime: timestamppb.Now(),
				})

			r, err := gc.SendMsg(context.Background(), &msgs)
			if err != nil {
				log.Printf("Could not send messages: %v", err)
			}
			log.Printf("SendMsq result %s ", r.String())

		case <-recvTicker.C:
			curTime := time.Now().UTC()

			r, err := gc.RecvMsq(context.Background(), &pb.SvcReq{Data: fmt.Sprintf("any data. recv time: %s ", curTime.Format("2006-01-02 15:04:05"))})
			if err != nil {
				log.Printf("Could not send messages: %v", err)
			}
			log.Printf("RecvMsq result %s ", r.String())
		}
	}

}
