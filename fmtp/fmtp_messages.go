package fmtp

//import "encoding/json"

type PacketType uint8

const (
	_ PacketType = iota
	Operational
	Operator
	Identification
	System
	Status
	Unknown
)

const (
	UnknwnFmtpPacketType = "Unknown Typ"
)

var packetTypeMap = map[PacketType]string{
	Operational:    "operational",
	Operator:       "operator",
	Identification: "identification",
	System:         "system",
	Status:         "status",
	Unknown:        UnknwnFmtpPacketType,
}

func (pt PacketType) ToString() string {
	if curTypeString, ok := packetTypeMap[pt]; ok {
		return curTypeString
	}
	return UnknwnFmtpPacketType
}

func (pt *PacketType) FromString(packetTypeString string) {
	for curKey, curType := range packetTypeMap {
		if curType == packetTypeString {
			*pt = curKey
			return
		}
	}
	*pt = Unknown
}

// тип пакета и текст сообщения.
type FmtpMessage struct {
	Type       PacketType `json:"Omitted,omitempty"`
	//TypeString string     `json:"FmtpType"`
	Text       string     `json:"Text,omitempty"`
}

var (
	StartupMessage   = FmtpMessage{Type: System, Text: "01"}
	ShutdownMessage  = FmtpMessage{Type: System, Text: "00"}
	HeartbeatMessage = FmtpMessage{Type: System, Text: "03"}

	RejectMessage = FmtpMessage{Type: Identification, Text: "REJECT"}
	// сообщение принятия подключения.
	AcceptMessage = FmtpMessage{Type: Identification, Text: "ACCEPT"}
)

// идентификационное сообщение
func CreateIdentificationMessage(localName string, remoteName string, ownMsg bool) FmtpMessage {
	if ownMsg {
		return FmtpMessage{Type: Identification, Text: localName + "-" + remoteName}
	} else {
		return FmtpMessage{Type: Identification, Text: remoteName + "-" + localName}
	}
}
