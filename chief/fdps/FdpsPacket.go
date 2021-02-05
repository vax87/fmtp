package fdps

import (
	"encoding/xml"
	"fmt"
)

const (
	FdpsDataText    = "data" // текст тип сообщения для ключа fdpsMessageKey.
	FdpsAcknowledge = "acc"  // текст подтверждения для ключа fdpsMessageKey.

	FdpsAckOkText      = "ok"      // FMTP канал есть, его состояние = data_ready, сообщение отправлено.
	FdpsAckStoppedText = "stopped" // FMTP канал есть, его состояние = stopped, сообщение не отправлено.
	FdpsAckFailedText  = "failed"  // FMTP канал есть, его состояние != data_ready, сообщение не отправлено.
	FdpsAckMissedText  = "missed"  // FMTP канала нет.

	FdpsAodbService = "AODB служба"
	FdpsOldiService = "OLDI служба"
)

type FdpsHeader struct {
	MsgHeader string `json:"MgsHeader"` // тип содержимого пакета.
}

// FdpsAodbPackage - пакет для обмена данными с провайдером плановой AODB.
type FdpsAodbPackage struct {
	FdpsHeader
	Ident     string `json:"Id"`     // идентификатор для подтверждения обмена с плановой.
	LocalAtc  string `json:"LocATC"` // локальный ATC канала.
	RemoteAtc string `json:"RemATC"` // удаленный АТС канала.
	Text      string `json:"Text"`   // содержание пакета.
}

// FdpsAodbAcknowledge - подтверждение получения данных
type FdpsAodbAcknowledge struct {
	FdpsHeader
	Ident string `json:"Id"`    // идентификатор для подтверждения обмена с плановой.
	State string `json:"State"` // признак отправки сообщения.
}

// FdpsOldiPackage - пакет для обмена данными с провайдером плановой OLDI
type FdpsOldiPackage struct {
	XMLName   xml.Name `xml:"msg"`
	Id        int      `xml:"id"`
	LocalAtc  string   `xml:"loc_atc"`
	RemoteAtc string   `xml:"cid"`
	Text      string   `xml:"txt"`
}

// FdpsOldiAcknowledge - подтверждение получения данных
type FdpsOldiAcknowledge struct {
	Id int `xml:"acc"`
}

func (acc *FdpsOldiAcknowledge) ToString() []byte {
	return []byte(fmt.Sprintf("<acc>%d</acc>", acc.Id))
}
