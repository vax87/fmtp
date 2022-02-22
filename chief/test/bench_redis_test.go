package bench_test_redis

import (
	"fmtp/fmtp_log"
	"testing"
)

var (
	msg = fmtp_log.LogMessage{
		ControllerIP:   "127.0.0.1",
		Source:         "Контроллер",
		ChannelId:      1,
		ChannelLocName: "LOCN",
		ChannelRemName: "REMN",
		DataType:       "OLDI",
		Severity:       "Информация",
		FmtpType:       "operational",
		Direction:      "Входящее",
		Text:           "Получено сообщение от плановой подсистемы: MSG TO UEEE №699408 time: 2022-02-21 09:33:38",
		DateTime:       "2022-02-21 09:33:37.812",
	}
)

func BenchmarkToJson(b *testing.B) {
	//msgList := make([]fmtp_log.LogMessage, 0)

	msg := fmtp_log.LogMessage{
		ControllerIP:   "127.0.0.1",
		Source:         "Контроллер",
		ChannelId:      1,
		ChannelLocName: "LOCN",
		ChannelRemName: "REMN",
		DataType:       "OLDI",
		Severity:       "Информация",
		FmtpType:       "operational",
		Direction:      "Входящее",
		Text:           "Получено сообщение от плановой подсистемы: MSG TO UEEE №699408 time: 2022-02-21 09:33:38",
		DateTime:       "2022-02-21 09:33:37.812",
	}

	for i := 0; i < b.N; i++ {
		//msg = append(msgList, msg)
		dt, err := msg.MarshalBinaryJson()
		if err != nil {
			b.Fatalf("Marshall error: %v", err)
		}

		err2 := msg.UnmarshalBinaryJson(dt, &msg)
		if err2 != nil {
			b.Fatalf("Unmarshall error: %v", err2)
		}
	}
}

func BenchmarkToString(b *testing.B) {

	for i := 0; i < b.N; i++ {
		//msg = append(msgList, msg)
		dt := msg.MarshalToString()

		_, err2 := fmtp_log.UnmarshalFromString(dt)
		if err2 != nil {
			b.Fatalf("Unmarshall error: %v", err2)
		}
	}
}
