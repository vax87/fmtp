package fmtp

import (
	//"fmt"
	"encoding/binary"
	//"errors"
)

const (
	FmtpVersion  = 2
	FmtpReserved = 0
	// length of a header field in bytes(3*uint8 + 1*uint16)
	FmtpHeaderLen = 5
	// maxLength that can be indicated in a header, as limited by the size of an uint16
	FmtpPackageMaxLength = 2<<15 - 1 // 65535 is the max length
	// MaxBodyLen is the maximum body len in bytes
	FmtpPackageBodyMaxLen = FmtpPackageMaxLength - FmtpHeaderLen // 65530 is the max body length
)

// заголовок FMTP пакета
type FmtpPacketTCPHeader struct {
	Version  uint8
	Reserved uint8
	PkgLen   uint16
	PkgType  PacketType
	IsValid  bool
}

// формирование FMTP пакета (с заголовком) из сообщения
func MakeFmtpPacket(msg FmtpMessage) []byte {
	curFmtpHeader := FmtpPacketTCPHeader{Version: 2, Reserved: 0, PkgLen: uint16(len(msg.Text)) + FmtpHeaderLen, PkgType: msg.Type}

	retValue, _ := curFmtpHeader.marshalFmtpHeader()
	retValue = append(retValue, []byte(msg.Text)...)
	return retValue
}

// разбор FMTP пакета (с заголовком) в сообщение
//func ParceFmtpPacket(header FmtpPacketTCPHeader, body []byte) (FmtpMessage, error) {
//	if curFmtpHeader, err := ParceFmtpPacketHeader(data[:FmtpHeaderLen]); err != nil {
//		return FmtpMessage{}, err
//	} else {
//		return FmtpMessage{Type: curFmtpHeader.packetType, Text: data[FmtpHeaderLen : FmtpHeaderLen+curFmtpHeader.bodyLen()]}, nil
//	}
//}

// пизженый код
// MarshalBinary marshals a header into binary form
func (fph *FmtpPacketTCPHeader) marshalFmtpHeader() ([]byte, error) {
	// Check
	//err := h.Check()
	//if err != nil {
	//	return nil, err
	//}

	// Get the length in binary
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, fph.PkgLen)

	// Now create the byte slice
	out := []byte{
		byte(fph.Version),
		byte(fph.Reserved),
		byte(lenBuf[0]),
		byte(lenBuf[1]),
		byte(fph.PkgType),
	}
	return out, nil
}

// пизженый код
func ParceFmtpPacketHeader(data []byte) FmtpPacketTCPHeader {
	length := binary.BigEndian.Uint16(data[2:4])
	valid := (data[0] == FmtpVersion && data[1] == FmtpReserved &&
		PacketType(data[4]).ToString() != UnknwnFmtpPacketType && uint16(length) < FmtpPackageBodyMaxLen && uint16(length) > FmtpHeaderLen)

	return FmtpPacketTCPHeader{Version: data[0], Reserved: data[1], PkgLen: uint16(length),
		PkgType: PacketType(data[4]), IsValid: valid}
}

// размер тела сообщения из пакета(байт)
func (fph *FmtpPacketTCPHeader) BodyLen() int {
	if fph.PkgLen == 0 {
		return 0
	}
	return int(fph.PkgLen) - FmtpHeaderLen
}
